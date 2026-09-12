package activation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activationtrace"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

type fakeRepository struct {
	order      *[]string
	active     map[string]model.AlgorithmInfo
	versions   map[string]model.AlgorithmVersionInfo
	ensureErr  error
	saveErr    error
	saveCalls  int
	sourcePath string
	pluginPath string
}

func (r *fakeRepository) EnsureDirs() error {
	if r.order != nil {
		*r.order = append(*r.order, "ensure")
	}
	return r.ensureErr
}

func (r *fakeRepository) SourcePath(string) string { return r.sourcePath }
func (r *fakeRepository) PluginPath(string) string { return r.pluginPath }
func (r *fakeRepository) VersionSourcePath(string, uint64) string {
	return r.sourcePath
}
func (r *fakeRepository) VersionPluginPath(string, uint64) string {
	return r.pluginPath
}

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

func (r *fakeRepository) ActiveVersion(name string) (model.AlgorithmVersionInfo, bool) {
	if r.versions != nil {
		if info, ok := r.versions[name]; ok {
			return info, true
		}
	}
	info, ok := r.active[name]
	if !ok {
		return model.AlgorithmVersionInfo{}, false
	}
	return model.LegacyVersion(info), true
}

func (r *fakeRepository) PreparedVersion(name string, version uint64) (model.AlgorithmVersionInfo, bool) {
	info, ok := r.ActiveVersion(name)
	if !ok || info.Version != version || !info.IsPrepared() {
		return model.AlgorithmVersionInfo{}, false
	}
	return info, true
}

func (r *fakeRepository) SetActiveVersion(name string, info model.AlgorithmVersionInfo) {
	if r.versions == nil {
		r.versions = make(map[string]model.AlgorithmVersionInfo)
	}
	r.versions[name] = info
	r.active[name] = info.Base()
}

func (r *fakeRepository) DeleteActiveVersion(name string) {
	delete(r.versions, name)
	delete(r.active, name)
}

func (r *fakeRepository) DeletePreparedVersion(name string, version uint64) {
	if info, ok := r.versions[name]; ok && info.Version == version {
		delete(r.versions, name)
	}
}

func (r *fakeRepository) Save() error {
	if r.order != nil {
		*r.order = append(*r.order, "save")
	}
	r.saveCalls++
	return r.saveErr
}

func TestServiceActivateRunsStagesInOrder(t *testing.T) {
	var order []string
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sourcePath := filepath.Join(t.TempDir(), "Add.wasm")
	compiledPath := filepath.Join(t.TempDir(), "compiled", "Add")
	repository := &fakeRepository{
		order:      &order,
		active:     make(map[string]model.AlgorithmInfo),
		sourcePath: sourcePath,
		pluginPath: compiledPath,
	}
	info := model.AlgorithmInfo{Code: "encoded", Gas: 7, IType: "bytes", OType: "bytes"}
	service := NewService(
		CodecFunc(func(encoded, path string) error {
			order = append(order, "decode")
			if encoded != info.Code || path != repository.sourcePath {
				t.Fatalf("unexpected decode input: %q %q", encoded, path)
			}
			if err := os.WriteFile(path, wasm, 0o644); err != nil {
				t.Fatalf("write wasm bytes: %v", err)
			}
			return nil
		}),
		RuntimeFunc(func(_ context.Context, wasmPath, compiledPath string) error {
			order = append(order, "activate")
			if wasmPath != repository.sourcePath || compiledPath != repository.pluginPath {
				t.Fatalf("unexpected activate paths: %q %q", wasmPath, compiledPath)
			}
			return nil
		}),
		repository,
	)
	if err := service.Activate(context.Background(), "add", info); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if want := []string{"ensure", "decode", "save", "activate"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected stage order: want %v got %v", want, order)
	}
	got, ok := repository.ActiveVersion("Add")
	if !ok {
		t.Fatal("expected prepared version metadata")
	}
	if got.Code != info.Code || got.Gas != info.Gas || got.IType != info.IType || got.OType != info.OType {
		t.Fatalf("unexpected algorithm metadata: %#v", got)
	}
	if got.WasmHash != crypto.Keccak256Hash(wasm).Hex() {
		t.Fatalf("unexpected wasm hash: %s", got.WasmHash)
	}
	if got.RuntimeName == "" || got.RuntimeVersion == "" {
		t.Fatalf("runtime metadata not recorded: %#v", got)
	}
	base, ok := repository.Active("Add")
	if !ok || base.Code != info.Code || base.Gas != info.Gas || base.IType != info.IType || base.OType != info.OType {
		t.Fatalf("legacy active metadata not preserved: ok=%t info=%#v", ok, base)
	}
}

func TestServiceActivateVersionUsesVersionedPaths(t *testing.T) {
	var order []string
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sourcePath := filepath.Join(t.TempDir(), "Add-2.wasm")
	compiledPath := filepath.Join(t.TempDir(), "compiled", "Add-2")
	repository := &fakeRepository{
		order:      &order,
		active:     make(map[string]model.AlgorithmInfo),
		sourcePath: sourcePath,
		pluginPath: compiledPath,
	}
	info := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "encoded", Gas: 7, IType: "bytes", OType: "bytes"},
		Version:         2,
		ActivationBlock: 42,
	}
	service := NewService(
		CodecFunc(func(encoded, path string) error {
			if encoded != info.Code || path != repository.sourcePath {
				t.Fatalf("unexpected decode input: %q %q", encoded, path)
			}
			if err := os.WriteFile(path, wasm, 0o644); err != nil {
				t.Fatalf("write wasm bytes: %v", err)
			}
			return nil
		}),
		RuntimeFunc(func(_ context.Context, wasmPath, compiledPath string) error {
			if wasmPath != repository.sourcePath || compiledPath != repository.pluginPath {
				t.Fatalf("unexpected activate paths: %q %q", wasmPath, compiledPath)
			}
			return nil
		}),
		repository,
	)
	if err := service.ActivateVersion(context.Background(), "add", info); err != nil {
		t.Fatalf("ActivateVersion failed: %v", err)
	}
	got, ok := repository.ActiveVersion("Add")
	if !ok {
		t.Fatal("expected prepared version metadata")
	}
	if got.Version != info.Version || got.ActivationBlock != info.ActivationBlock {
		t.Fatalf("unexpected version metadata: %#v", got)
	}
	if got.WasmHash != crypto.Keccak256Hash(wasm).Hex() {
		t.Fatalf("unexpected wasm hash: %s", got.WasmHash)
	}
	if got.RuntimeName == "" || got.RuntimeVersion == "" {
		t.Fatalf("runtime metadata not recorded: %#v", got)
	}
}

func TestServiceActivateWritesTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "activation_trace.jsonl")
	t.Setenv(activationtrace.TraceFileEnvVar, tracePath)
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	sourcePath := filepath.Join(t.TempDir(), "Add.wasm")
	compiledPath := filepath.Join(t.TempDir(), "compiled", "Add")
	repository := &fakeRepository{
		active:     make(map[string]model.AlgorithmInfo),
		sourcePath: sourcePath,
		pluginPath: compiledPath,
	}
	info := model.AlgorithmInfo{Code: "encoded", Gas: 7, IType: "bytes", OType: "bytes"}
	service := NewService(
		CodecFunc(func(string, string) error {
			return os.WriteFile(sourcePath, wasm, 0o644)
		}),
		RuntimeFunc(func(context.Context, string, string) error {
			return nil
		}),
		repository,
	)
	if err := service.Activate(context.Background(), "Add", info); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected trace line count %d: %s", len(lines), raw)
	}
	var persisted, completed activationtrace.Event
	if err := json.Unmarshal([]byte(lines[0]), &persisted); err != nil {
		t.Fatalf("unmarshal persisted trace: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &completed); err != nil {
		t.Fatalf("unmarshal completed trace: %v", err)
	}
	if persisted.Stage != "wasm_persisted" || completed.Stage != "activation_completed" {
		t.Fatalf("unexpected trace stages: %#v %#v", persisted, completed)
	}
	if persisted.WasmHash != crypto.Keccak256Hash(wasm).Hex() || persisted.WasmPath != sourcePath || completed.Name != "Add" {
		t.Fatalf("unexpected trace metadata: %#v %#v", persisted, completed)
	}
}

func TestServiceActivateVersionSkipsAlreadyPreparedVersion(t *testing.T) {
	var order []string
	info := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "encoded", Gas: 7, WasmHash: "0x1", RuntimeName: "wazero", RuntimeVersion: "v1.12.0"},
		Version:         2,
		ActivationBlock: 42,
	}
	repository := &fakeRepository{
		order:      &order,
		active:     map[string]model.AlgorithmInfo{"Add": info.Base()},
		versions:   map[string]model.AlgorithmVersionInfo{"Add": info},
		sourcePath: filepath.Join(t.TempDir(), "Add-2.wasm"),
		pluginPath: filepath.Join(t.TempDir(), "compiled", "Add-2"),
	}
	service := NewService(
		CodecFunc(func(string, string) error {
			order = append(order, "decode")
			return nil
		}),
		RuntimeFunc(func(context.Context, string, string) error {
			order = append(order, "load")
			return nil
		}),
		repository,
	)
	if err := service.ActivateVersion(context.Background(), "Add", info); err != nil {
		t.Fatalf("ActivateVersion failed: %v", err)
	}
	if len(order) != 0 {
		t.Fatalf("already prepared version repeated activation stages: %v", order)
	}
}

func TestServiceActivatePropagatesStageFailures(t *testing.T) {
	stageErr := errors.New("stage failed")
	tests := []struct {
		name      string
		failStage string
		wantStage string
	}{
		{name: "decode", failStage: "decode", wantStage: "wasm"},
		{name: "persist", failStage: "save", wantStage: "persist"},
		{name: "activate", failStage: "activate", wantStage: "activate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var order []string
			oldWasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
			old := model.AlgorithmVersionInfo{
				AlgorithmInfo:   model.AlgorithmInfo{Code: "old", Gas: 1, WasmHash: crypto.Keccak256Hash(oldWasm).Hex(), RuntimeName: "wazero", RuntimeVersion: "v1.12.0"},
				Version:         1,
				ActivationBlock: 0,
			}
			repository := &fakeRepository{
				order:      &order,
				active:     map[string]model.AlgorithmInfo{"Add": old.Base()},
				versions:   map[string]model.AlgorithmVersionInfo{"Add": old},
				sourcePath: filepath.Join(t.TempDir(), "Add.wasm"),
				pluginPath: filepath.Join(t.TempDir(), "compiled", "Add"),
			}
			if test.failStage == "save" {
				repository.saveErr = stageErr
			}
			service := NewService(
				CodecFunc(func(_, path string) error {
					order = append(order, "decode")
					if err := os.WriteFile(path, oldWasm, 0o644); err != nil {
						t.Fatalf("write wasm bytes: %v", err)
					}
					if test.failStage == "decode" {
						return stageErr
					}
					return nil
				}),
				RuntimeFunc(func(context.Context, string, string) error {
					order = append(order, "activate")
					if test.failStage == "activate" {
						return stageErr
					}
					return nil
				}),
				repository,
			)
			err := service.Activate(context.Background(), "Add", model.AlgorithmInfo{Code: "new", Gas: 2})
			if err == nil || !errors.Is(err, stageErr) || !strings.Contains(err.Error(), test.wantStage) {
				t.Fatalf("unexpected stage error: %v", err)
			}
			if got, ok := repository.Active("Add"); !ok || !reflect.DeepEqual(got, old.Base()) {
				t.Fatalf("failed activation changed active metadata: ok=%t %#v", ok, got)
			}
			if got, ok := repository.ActiveVersion("Add"); !ok || !reflect.DeepEqual(got, old) {
				t.Fatalf("failed activation changed active version metadata: ok=%t %#v", ok, got)
			}
		})
	}
}

func TestServiceActivateFailureRemovesFirstActivationMetadata(t *testing.T) {
	var order []string
	repository := &fakeRepository{
		order:      &order,
		active:     make(map[string]model.AlgorithmInfo),
		sourcePath: filepath.Join(t.TempDir(), "Add.wasm"),
		pluginPath: filepath.Join(t.TempDir(), "compiled", "Add"),
	}
	service := NewService(
		CodecFunc(func(_, path string) error {
			return os.WriteFile(path, []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}, 0o644)
		}),
		RuntimeFunc(func(context.Context, string, string) error { return errors.New("invalid wasm runtime") }),
		repository,
	)
	if err := service.Activate(context.Background(), "Add", model.AlgorithmInfo{Code: "new"}); err == nil {
		t.Fatal("expected activate failure")
	}
	if got, ok := repository.Active("Add"); ok {
		t.Fatalf("activate failure left first activation metadata active: %#v", got)
	}
	if repository.saveCalls != 2 {
		t.Fatalf("expected activation save and rollback save, got %d", repository.saveCalls)
	}
}

func TestServiceActivateVersionLoadFailureRestoresPreviousVersion(t *testing.T) {
	previous := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "old", Gas: 1},
		Version:         1,
		ActivationBlock: 0,
	}
	next := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "new", Gas: 2},
		Version:         2,
		ActivationBlock: 42,
	}
	repository := &fakeRepository{
		order:      new([]string),
		active:     map[string]model.AlgorithmInfo{"Add": previous.Base()},
		versions:   map[string]model.AlgorithmVersionInfo{"Add": previous},
		sourcePath: filepath.Join(t.TempDir(), "Add-2.wasm"),
		pluginPath: filepath.Join(t.TempDir(), "compiled", "Add-2"),
	}
	service := NewService(
		CodecFunc(func(_, path string) error {
			return os.WriteFile(path, []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}, 0o644)
		}),
		RuntimeFunc(func(context.Context, string, string) error { return errors.New("invalid wasm runtime") }),
		repository,
	)
	if err := service.ActivateVersion(context.Background(), "Add", next); err == nil {
		t.Fatal("expected activate failure")
	}
	if got := repository.versions["Add"]; got != previous {
		t.Fatalf("load failure did not restore previous version: %#v", got)
	}
	if got := repository.active["Add"]; got != previous.Base() {
		t.Fatalf("load failure changed legacy active metadata: %#v", got)
	}
}
