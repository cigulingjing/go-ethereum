package pqcbench

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	wasmMemoryPages uint32        = 256
	wasmExecBudget  time.Duration = 30 * time.Second
	wasmInputOffset uint32        = 1 << 16
	wasmInitExport                = "_initialize"
)

// WASMRuntime 与 EvoCrypt 相同：execute(ptr,len)->ptr，长度前缀输出，仅 wasi_snapshot_preview1。
type WASMRuntime struct {
	mu      sync.Mutex
	modules map[string]*preparedWASM
}

func NewWASMRuntime() *WASMRuntime {
	return &WASMRuntime{modules: make(map[string]*preparedWASM)}
}

func (r *WASMRuntime) Execute(ctx context.Context, wasmPath, compiledPath string, input []byte) ([]byte, error) {
	module, err := r.prepare(ctx, wasmPath, compiledPath)
	if err != nil {
		return nil, err
	}
	return module.execute(ctx, input)
}

func (r *WASMRuntime) prepare(ctx context.Context, wasmPath, compiledPath string) (*preparedWASM, error) {
	wasmPath = filepath.Clean(wasmPath)
	compiledPath = filepath.Clean(compiledPath)
	r.mu.Lock()
	defer r.mu.Unlock()
	if module, ok := r.modules[wasmPath]; ok && module != nil {
		return module, nil
	}
	prepared, err := compileWASM(ctx, wasmPath, compiledPath)
	if err != nil {
		return nil, err
	}
	r.modules[wasmPath] = prepared
	return prepared, nil
}

type preparedWASM struct {
	runtime wazero.Runtime
	module  api.Module
	fn      api.Function
	memory  api.Memory
	mu      sync.Mutex
}

func compileWASM(ctx context.Context, wasmPath, compiledPath string) (*preparedWASM, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(compiledPath, 0o755); err != nil {
		return nil, err
	}
	cache, err := wazero.NewCompilationCacheWithDir(compiledPath)
	if err != nil {
		return nil, err
	}
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithMemoryLimitPages(wasmMemoryPages).
		WithCloseOnContextDone(true).
		WithCompilationCache(cache))
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("compile %s: %w", wasmPath, err)
	}
	for _, fn := range compiled.ImportedFunctions() {
		moduleName, name, _ := fn.Import()
		if moduleName != wasi_snapshot_preview1.ModuleName {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("forbidden import %s.%s", moduleName, name)
		}
	}
	instance, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithName(strings.TrimSuffix(filepath.Base(wasmPath), filepath.Ext(wasmPath))).
		WithStartFunctions())
	if err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("instantiate %s: %w", wasmPath, err)
	}
	execute := instance.ExportedFunction("execute")
	if execute == nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("execute export missing: %s", wasmPath)
	}
	memory := instance.Memory()
	if memory == nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("memory missing: %s", wasmPath)
	}
	if init := instance.ExportedFunction(wasmInitExport); init != nil {
		if _, err := init.Call(ctx); err != nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("initialize %s: %w", wasmPath, err)
		}
	}
	return &preparedWASM{runtime: rt, module: instance, fn: execute, memory: memory}, nil
}

func (m *preparedWASM) execute(ctx context.Context, input []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	execCtx, cancel := context.WithTimeout(ctx, wasmExecBudget)
	defer cancel()
	if len(input) > 0 {
		required := wasmInputOffset + uint32(len(input))
		if current := m.memory.Size(); current < required {
			pages := (required - current + 65535) / 65536
			if _, ok := m.memory.Grow(pages); !ok {
				return nil, fmt.Errorf("grow wasm memory by %d pages", pages)
			}
		}
		if !m.memory.Write(wasmInputOffset, input) {
			return nil, fmt.Errorf("write %d input bytes", len(input))
		}
	}
	results, err := m.fn.Call(execCtx, api.EncodeU32(wasmInputOffset), api.EncodeU32(uint32(len(input))))
	if err != nil {
		return nil, err
	}
	if len(results) != 1 {
		return nil, fmt.Errorf("execute returned %d values", len(results))
	}
	ptr := api.DecodeU32(results[0])
	length, ok := m.memory.ReadUint32Le(ptr)
	if !ok {
		return nil, fmt.Errorf("read output length at %d", ptr)
	}
	out, ok := m.memory.Read(ptr+4, length)
	if !ok {
		return nil, fmt.Errorf("read output at %d length %d", ptr+4, length)
	}
	return append([]byte(nil), out...), nil
}
