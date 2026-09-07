// Package wasmruntime provides the WASM execution boundary used by
// cryptoupgrade. It compiles modules with wazero, instantiates them without
// host imports, and executes the exported execute entry point.
package wasmruntime

import (
	"context"
	"errors"
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
	RuntimeName    = "wazero"
	RuntimeVersion = "v1.12.0"
	wasmInitExport = "_initialize"

	DefaultMemoryLimitPages uint32        = 256
	DefaultExecutionBudget  time.Duration = 5 * time.Second
	inputMemoryOffset       uint32        = 1 << 16
	allowedImportModule                   = wasi_snapshot_preview1.ModuleName
)

var (
	ErrRequiredVersionMissing = errors.New("required-version-missing")
	ErrExecuteExportMissing   = errors.New("execute export missing")
	ErrMemoryMissing          = errors.New("module memory missing")
)

// Config controls how modules are compiled and executed.
type Config struct {
	MemoryLimitPages uint32
	ExecutionBudget  time.Duration
}

func (c Config) normalize() Config {
	if c.MemoryLimitPages == 0 {
		c.MemoryLimitPages = DefaultMemoryLimitPages
	}
	if c.ExecutionBudget <= 0 {
		c.ExecutionBudget = DefaultExecutionBudget
	}
	return c
}

// Loader caches prepared WASM modules by their source path.
type Loader struct {
	cfg Config

	mu      sync.RWMutex
	modules map[string]*preparedModule
}

// Default is the process-wide WASM loader used by cryptoupgrade.
var Default = NewLoader(Config{})

// NewLoader creates a new loader with the supplied configuration.
func NewLoader(cfg Config) *Loader {
	return &Loader{
		cfg:     cfg.normalize(),
		modules: make(map[string]*preparedModule),
	}
}

// Activate compiles and instantiates a module from wasmPath, caching the
// prepared module for later execution.
func (l *Loader) Activate(ctx context.Context, wasmPath, compiledPath string) error {
	_, err := l.prepare(ctx, wasmPath, compiledPath)
	return err
}

// Execute runs the prepared module identified by wasmPath. If the module is
// not already loaded, it is prepared lazily from the persisted WASM file.
func (l *Loader) Execute(ctx context.Context, wasmPath, compiledPath string, input []byte) ([]byte, error) {
	module, err := l.prepare(ctx, wasmPath, compiledPath)
	if err != nil {
		return nil, err
	}
	return module.Execute(ctx, l.cfg.ExecutionBudget, input)
}

func (l *Loader) prepare(ctx context.Context, wasmPath, compiledPath string) (*preparedModule, error) {
	wasmPath = filepath.Clean(wasmPath)
	compiledPath = filepath.Clean(compiledPath)
	if wasmPath == "" {
		return nil, errors.New("wasm path is empty")
	}
	if compiledPath == "" {
		return nil, errors.New("compiled path is empty")
	}

	l.mu.RLock()
	module, ok := l.modules[wasmPath]
	l.mu.RUnlock()
	if ok && module != nil && !module.IsClosed() {
		return module, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if module, ok = l.modules[wasmPath]; ok && module != nil && !module.IsClosed() {
		return module, nil
	}
	if module != nil {
		_ = module.Close(ctx)
	}

	prepared, err := l.compileAndInstantiate(ctx, wasmPath, compiledPath)
	if err != nil {
		return nil, err
	}
	l.modules[wasmPath] = prepared
	return prepared, nil
}

func (l *Loader) compileAndInstantiate(ctx context.Context, wasmPath, compiledPath string) (*preparedModule, error) {
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("read wasm module %s: %w", wasmPath, err)
	}
	if err := os.MkdirAll(compiledPath, 0o755); err != nil {
		return nil, fmt.Errorf("create compiled cache directory %s: %w", compiledPath, err)
	}
	cache, err := wazero.NewCompilationCacheWithDir(compiledPath)
	if err != nil {
		return nil, fmt.Errorf("create wazero compilation cache at %s: %w", compiledPath, err)
	}

	runtimeConfig := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(l.cfg.MemoryLimitPages).
		WithCloseOnContextDone(true).
		WithCompilationCache(cache)
	rt := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("instantiate wasi module for %s: %w", wasmPath, err)
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("compile wasm module %s: %w", wasmPath, err)
	}
	if err := validateImports(compiled, wasmPath); err != nil {
		_ = compiled.Close(ctx)
		_ = rt.Close(ctx)
		return nil, err
	}

	moduleName := strings.TrimSuffix(filepath.Base(wasmPath), filepath.Ext(wasmPath))
	instance, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(moduleName).WithStartFunctions())
	if err != nil {
		_ = compiled.Close(ctx)
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("instantiate wasm module %s: %w", wasmPath, err)
	}
	execute := instance.ExportedFunction("execute")
	if execute == nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("%w: %s", ErrExecuteExportMissing, wasmPath)
	}
	memory := instance.Memory()
	if memory == nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("%w: %s", ErrMemoryMissing, wasmPath)
	}
	if init := instance.ExportedFunction(wasmInitExport); init != nil {
		if _, err := init.Call(ctx); err != nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("initialize wasm module %s: %w", wasmPath, err)
		}
	}

	return &preparedModule{
		runtime:   rt,
		module:    instance,
		execute:   execute,
		memory:    memory,
		wasmPath:  wasmPath,
		cachePath: compiledPath,
	}, nil
}

type preparedModule struct {
	runtime   wazero.Runtime
	module    api.Module
	execute   api.Function
	memory    api.Memory
	wasmPath  string
	cachePath string

	mu sync.Mutex
}

func (m *preparedModule) IsClosed() bool {
	return m == nil || m.module == nil || m.module.IsClosed()
}

func (m *preparedModule) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if m.runtime == nil {
		return nil
	}
	return m.runtime.Close(ctx)
}

func (m *preparedModule) Execute(ctx context.Context, timeout time.Duration, input []byte) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("%w: <nil>", ErrRequiredVersionMissing)
	}
	if m.IsClosed() {
		return nil, fmt.Errorf("%w: %s", ErrRequiredVersionMissing, m.wasmPath)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsClosed() {
		return nil, fmt.Errorf("%w: %s", ErrRequiredVersionMissing, m.wasmPath)
	}
	execCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	if err := m.ensureInputCapacity(inputMemoryOffset, uint32(len(input))); err != nil {
		return nil, err
	}
	if !m.memory.Write(inputMemoryOffset, input) {
		return nil, fmt.Errorf("write %d input bytes to wasm memory", len(input))
	}
	results, err := m.execute.Call(execCtx, api.EncodeU32(inputMemoryOffset), api.EncodeU32(uint32(len(input))))
	if err != nil {
		return nil, fmt.Errorf("execute wasm module %s: %w", m.wasmPath, err)
	}
	if len(results) != 1 {
		return nil, fmt.Errorf("execute export on %s returned %d values, want 1", m.wasmPath, len(results))
	}
	outputPtr := api.DecodeU32(results[0])
	return m.readLengthPrefixed(outputPtr)
}

func validateImports(compiled wazero.CompiledModule, wasmPath string) error {
	for _, fn := range compiled.ImportedFunctions() {
		moduleName, name, _ := fn.Import()
		if moduleName != allowedImportModule {
			return fmt.Errorf("wasm module %s declares forbidden import %s.%s", wasmPath, moduleName, name)
		}
	}
	if len(compiled.ImportedMemories()) != 0 {
		return fmt.Errorf("wasm module %s declares forbidden imported memory", wasmPath)
	}
	return nil
}

func (m *preparedModule) ensureInputCapacity(offset, inputLen uint32) error {
	if inputLen == 0 {
		return nil
	}
	required := offset + inputLen
	if current := m.memory.Size(); current < required {
		missing := required - current
		pages := (missing + 65535) / 65536
		if _, ok := m.memory.Grow(pages); !ok {
			return fmt.Errorf("grow wasm memory for %s by %d pages", m.wasmPath, pages)
		}
	}
	return nil
}

func (m *preparedModule) readLengthPrefixed(ptr uint32) ([]byte, error) {
	if length, ok := m.memory.ReadUint32Le(ptr); ok {
		start := ptr + 4
		output, ok := m.memory.Read(start, length)
		if !ok {
			return nil, fmt.Errorf("read wasm output at %d length %d from %s", start, length, m.wasmPath)
		}
		return append([]byte(nil), output...), nil
	}
	return nil, fmt.Errorf("read wasm output length at %d from %s", ptr, m.wasmPath)
}
