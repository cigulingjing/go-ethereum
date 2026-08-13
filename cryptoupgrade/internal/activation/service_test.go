package activation

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

type fakeRepository struct {
	order      *[]string
	active     map[string]model.AlgorithmInfo
	ensureErr  error
	saveErr    error
	saveCalls  int
	sourcePath string
	pluginPath string
}

func (r *fakeRepository) EnsureDirs() error {
	*r.order = append(*r.order, "ensure")
	return r.ensureErr
}

func (r *fakeRepository) SourcePath(string) string { return r.sourcePath }
func (r *fakeRepository) PluginPath(string) string { return r.pluginPath }

func (r *fakeRepository) Active(name string) (model.AlgorithmInfo, bool) {
	info, ok := r.active[name]
	return info, ok
}

func (r *fakeRepository) SetActive(name string, info model.AlgorithmInfo) {
	r.active[name] = info
}

func (r *fakeRepository) DeleteActive(name string) {
	delete(r.active, name)
}

func (r *fakeRepository) Save() error {
	*r.order = append(*r.order, "save")
	r.saveCalls++
	return r.saveErr
}

func TestServiceActivateRunsStagesInOrder(t *testing.T) {
	var order []string
	repository := &fakeRepository{
		order:      &order,
		active:     make(map[string]model.AlgorithmInfo),
		sourcePath: "src/Add.go",
		pluginPath: "so/Add.so",
	}
	info := model.AlgorithmInfo{Code: "encoded", Gas: 7}
	service := NewService(
		CodecFunc(func(encoded, path string) error {
			order = append(order, "decode")
			if encoded != info.Code || path != repository.sourcePath {
				t.Fatalf("unexpected decode input: %q %q", encoded, path)
			}
			return nil
		}),
		CompilerFunc(func(_ context.Context, sourcePath, pluginPath string) error {
			order = append(order, "compile")
			if sourcePath != repository.sourcePath || pluginPath != repository.pluginPath {
				t.Fatalf("unexpected compile paths: %q %q", sourcePath, pluginPath)
			}
			return nil
		}),
		repository,
		LoaderFunc(func(pluginPath, symbol string) error {
			order = append(order, "load")
			if pluginPath != repository.pluginPath || symbol != "Add" {
				t.Fatalf("unexpected load request: %q %q", pluginPath, symbol)
			}
			return nil
		}),
	)
	if err := service.Activate(context.Background(), "add", info); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if want := []string{"ensure", "decode", "compile", "save", "load"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected stage order: want %v got %v", want, order)
	}
	if got, ok := repository.active["Add"]; !ok || got != info {
		t.Fatalf("algorithm was not activated: ok=%t info=%#v", ok, got)
	}
}

func TestServiceActivatePropagatesStageFailures(t *testing.T) {
	stageErr := errors.New("stage failed")
	tests := []struct {
		name      string
		failStage string
		wantStage string
	}{
		{name: "decode", failStage: "decode", wantStage: "decode"},
		{name: "compile", failStage: "compile", wantStage: "compile"},
		{name: "persist", failStage: "save", wantStage: "persist"},
		{name: "load", failStage: "load", wantStage: "load"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var order []string
			old := model.AlgorithmInfo{Code: "old", Gas: 1}
			repository := &fakeRepository{
				order:      &order,
				active:     map[string]model.AlgorithmInfo{"Add": old},
				sourcePath: "src/Add.go",
				pluginPath: "so/Add.so",
			}
			if test.failStage == "save" {
				repository.saveErr = stageErr
			}
			service := NewService(
				CodecFunc(func(string, string) error {
					order = append(order, "decode")
					if test.failStage == "decode" {
						return stageErr
					}
					return nil
				}),
				CompilerFunc(func(context.Context, string, string) error {
					order = append(order, "compile")
					if test.failStage == "compile" {
						return stageErr
					}
					return nil
				}),
				repository,
				LoaderFunc(func(string, string) error {
					order = append(order, "load")
					if test.failStage == "load" {
						return stageErr
					}
					return nil
				}),
			)
			err := service.Activate(context.Background(), "Add", model.AlgorithmInfo{Code: "new", Gas: 2})
			if err == nil || !errors.Is(err, stageErr) || !strings.Contains(err.Error(), test.wantStage) {
				t.Fatalf("unexpected stage error: %v", err)
			}
			if got := repository.active["Add"]; got != old {
				t.Fatalf("failed activation changed active metadata: %#v", got)
			}
		})
	}
}

func TestServiceLoadFailureRemovesFirstActivationMetadata(t *testing.T) {
	var order []string
	repository := &fakeRepository{
		order:      &order,
		active:     make(map[string]model.AlgorithmInfo),
		sourcePath: "src/Add.go",
		pluginPath: "so/Add.so",
	}
	service := NewService(
		CodecFunc(func(string, string) error { return nil }),
		CompilerFunc(func(context.Context, string, string) error { return nil }),
		repository,
		LoaderFunc(func(string, string) error { return errors.New("invalid plugin") }),
	)
	if err := service.Activate(context.Background(), "Add", model.AlgorithmInfo{Code: "new"}); err == nil {
		t.Fatal("expected load failure")
	}
	if _, ok := repository.active["Add"]; ok {
		t.Fatal("load failure left first activation metadata active")
	}
	if repository.saveCalls != 2 {
		t.Fatalf("expected activation save and rollback save, got %d", repository.saveCalls)
	}
}
