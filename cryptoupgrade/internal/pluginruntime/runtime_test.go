package pluginruntime

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeModule struct {
	symbols map[string]any
}

func (m fakeModule) Lookup(name string) (any, error) {
	symbol, ok := m.symbols[name]
	if !ok {
		return nil, errors.New("symbol not found")
	}
	return symbol, nil
}

func TestLoaderCachesOpenedPlugin(t *testing.T) {
	var opens atomic.Int32
	loader := NewLoader(func(string) (Module, error) {
		opens.Add(1)
		return fakeModule{symbols: map[string]any{
			"Add": func(a, b int) int { return a + b },
		}}, nil
	})
	for range 2 {
		output, err := loader.Call("add.so", "Add", []any{1, 2})
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}
		if len(output) != 1 || output[0] != 3 {
			t.Fatalf("unexpected output: %#v", output)
		}
	}
	if got := opens.Load(); got != 1 {
		t.Fatalf("plugin opened %d times, want 1", got)
	}
}

func TestLoaderMissingSymbol(t *testing.T) {
	loader := NewLoader(func(string) (Module, error) {
		return fakeModule{symbols: map[string]any{}}, nil
	})
	if _, err := loader.Lookup("empty.so", "Missing"); err == nil {
		t.Fatal("expected missing symbol error")
	}
}

func TestLoaderInvalidPluginIsNotCached(t *testing.T) {
	var opens atomic.Int32
	loader := NewLoader(func(string) (Module, error) {
		opens.Add(1)
		return nil, errors.New("invalid plugin")
	})
	for range 2 {
		if _, err := loader.Lookup("invalid.so", "Add"); err == nil {
			t.Fatal("expected invalid plugin error")
		}
	}
	if got := opens.Load(); got != 2 {
		t.Fatalf("failed open was cached: got %d attempts, want 2", got)
	}
}

func TestLoaderConcurrentAccessOpensOnce(t *testing.T) {
	var opens atomic.Int32
	loader := NewLoader(func(string) (Module, error) {
		opens.Add(1)
		time.Sleep(time.Millisecond)
		return fakeModule{symbols: map[string]any{
			"Identity": func(value int) int { return value },
		}}, nil
	})

	const workers = 32
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for index := range workers {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			output, err := loader.Call("identity.so", "Identity", []any{value})
			if err == nil && (len(output) != 1 || output[0] != value) {
				err = errors.New("unexpected call output")
			}
			errs <- err
		}(index)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Call failed: %v", err)
		}
	}
	if got := opens.Load(); got != 1 {
		t.Fatalf("plugin opened %d times, want 1", got)
	}
}

func TestActivateUsesImmutablePathForSameCanonicalPlugin(t *testing.T) {
	canonicalPath := filepath.Join(t.TempDir(), "Add.so")
	var openedPaths []string
	loader := NewLoader(func(path string) (Module, error) {
		openedPaths = append(openedPaths, path)
		version, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return fakeModule{symbols: map[string]any{
			"Add": func() string { return string(version) },
		}}, nil
	})
	for _, version := range []string{"version-one", "version-two"} {
		if err := os.WriteFile(canonicalPath, []byte(version), 0755); err != nil {
			t.Fatalf("write canonical artifact: %v", err)
		}
		if err := loader.Activate(canonicalPath, "Add"); err != nil {
			t.Fatalf("Activate %s failed: %v", version, err)
		}
		symbol, err := loader.LookupActive(canonicalPath, "Add")
		if err != nil {
			t.Fatalf("LookupActive failed: %v", err)
		}
		output, err := CallFunction(symbol, nil)
		if err != nil || len(output) != 1 || output[0] != version {
			t.Fatalf("unexpected active version: output=%v err=%v", output, err)
		}
	}
	if len(openedPaths) != 2 || openedPaths[0] == openedPaths[1] {
		t.Fatalf("same canonical path reused by opener: %v", openedPaths)
	}
}

func TestCallFunctionRejectsInvalidArgumentsAndRecoversPanic(t *testing.T) {
	if _, err := CallFunction(func(int) {}, []any{"wrong"}); err == nil {
		t.Fatal("expected argument type error")
	}
	if _, err := CallFunction(func() { panic("boom") }, nil); err == nil {
		t.Fatal("expected recovered panic error")
	}
}
