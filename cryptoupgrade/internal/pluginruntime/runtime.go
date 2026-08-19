package pluginruntime

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

// Module 是已打开 plugin 所需的最小符号查询边界。
type Module interface {
	Lookup(name string) (any, error)
}

// Opener 打开指定路径的 Go plugin。
type Opener func(path string) (Module, error)

// Loader 缓存已打开 plugin，并提供安全的符号查询和反射调用。
type Loader struct {
	open Opener

	mu     sync.Mutex
	cache  map[string]Module
	active map[string]string
}

// NewLoader 创建使用指定 opener 的 plugin loader。
func NewLoader(open Opener) *Loader {
	return &Loader{
		open:   open,
		cache:  make(map[string]Module),
		active: make(map[string]string),
	}
}

// Lookup 查询 plugin 导出的符号。
func (l *Loader) Lookup(pluginPath, symbolName string) (any, error) {
	l.mu.Lock()
	module, ok := l.cache[pluginPath]
	if !ok {
		var err error
		module, err = l.open(pluginPath)
		if err != nil {
			l.mu.Unlock()
			return nil, fmt.Errorf("open plugin %s: %w", pluginPath, err)
		}
		l.cache[pluginPath] = module
	}
	l.mu.Unlock()

	symbol, err := module.Lookup(symbolName)
	if err != nil {
		return nil, fmt.Errorf("find symbol %s in plugin %s: %w", symbolName, pluginPath, err)
	}
	return symbol, nil
}

// Call 查询并调用 plugin 函数。
func (l *Loader) Call(pluginPath, symbolName string, args []any) ([]any, error) {
	fn, err := l.Lookup(pluginPath, symbolName)
	if err != nil {
		return nil, err
	}
	return CallFunction(fn, args)
}

// Activate 验证新制品包含目标符号，并将后续调用切换到该不可变制品。
func (l *Loader) Activate(pluginPath, symbolName string) error {
	snapshotPath, err := snapshotPlugin(pluginPath)
	if err != nil {
		return err
	}
	if _, err := l.Lookup(snapshotPath, symbolName); err != nil {
		return err
	}
	// plugin.Open 会按真实路径缓存已加载对象；使用内容寻址快照才能让同名算法升级后加载新二进制。
	l.mu.Lock()
	l.active[activeKey(pluginPath, symbolName)] = snapshotPath
	l.mu.Unlock()
	return nil
}

// LookupActive 查询激活阶段选定的不可变制品，进程重启后回退到 canonical 路径。
func (l *Loader) LookupActive(fallbackPath, symbolName string) (any, error) {
	l.mu.Lock()
	pluginPath := l.active[activeKey(fallbackPath, symbolName)]
	l.mu.Unlock()
	if pluginPath == "" {
		pluginPath = fallbackPath
	}
	return l.Lookup(pluginPath, symbolName)
}

func activeKey(pluginPath, symbolName string) string {
	return filepath.Clean(pluginPath) + "\x00" + symbolName
}

func snapshotPlugin(pluginPath string) (string, error) {
	artifact, err := os.ReadFile(pluginPath)
	if err != nil {
		return "", fmt.Errorf("read plugin artifact %s: %w", pluginPath, err)
	}
	sum := sha256.Sum256(artifact)
	snapshotDir := filepath.Join(filepath.Dir(pluginPath), ".runtime")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return "", fmt.Errorf("create plugin runtime directory: %w", err)
	}
	base := strings.TrimSuffix(filepath.Base(pluginPath), filepath.Ext(pluginPath))
	snapshotPath := filepath.Join(snapshotDir, fmt.Sprintf("%s-%x.so", base, sum))
	if _, err := os.Stat(snapshotPath); err == nil {
		return snapshotPath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect plugin snapshot: %w", err)
	}
	tmp, err := os.CreateTemp(snapshotDir, "."+base+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create plugin snapshot: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		return "", fmt.Errorf("set plugin snapshot mode: %w", err)
	}
	if _, err := tmp.Write(artifact); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write plugin snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close plugin snapshot: %w", err)
	}
	if err := os.Rename(tmpPath, snapshotPath); err != nil {
		if _, statErr := os.Stat(snapshotPath); statErr == nil {
			return snapshotPath, nil
		}
		return "", fmt.Errorf("publish plugin snapshot: %w", err)
	}
	return snapshotPath, nil
}

// CallFunction 安全调用签名在运行时确定的函数。
func CallFunction(fn any, args []any) (ret []any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			ret = nil
			err = fmt.Errorf("cryptoupgrade function panic: %v", recovered)
		}
	}()

	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}
	fnType := value.Type()
	if !fnType.IsVariadic() && len(args) != fnType.NumIn() {
		return nil, fmt.Errorf("function expects %d arguments, got %d", fnType.NumIn(), len(args))
	}
	if fnType.IsVariadic() && len(args) < fnType.NumIn()-1 {
		return nil, fmt.Errorf("function expects at least %d arguments, got %d", fnType.NumIn()-1, len(args))
	}

	inputs := make([]reflect.Value, len(args))
	for index, arg := range args {
		expected := argumentType(fnType, index)
		input, err := reflectArgument(arg, expected)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", index, err)
		}
		inputs[index] = input
	}
	results := value.Call(inputs)
	outputs := make([]any, len(results))
	for index, result := range results {
		outputs[index] = result.Interface()
	}
	return outputs, nil
}

func argumentType(fnType reflect.Type, index int) reflect.Type {
	if fnType.IsVariadic() && index >= fnType.NumIn()-1 {
		return fnType.In(fnType.NumIn() - 1).Elem()
	}
	return fnType.In(index)
}

func reflectArgument(arg any, expected reflect.Type) (reflect.Value, error) {
	if arg == nil {
		switch expected.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return reflect.Zero(expected), nil
		default:
			return reflect.Value{}, fmt.Errorf("nil is not assignable to %s", expected)
		}
	}
	value := reflect.ValueOf(arg)
	if !value.Type().AssignableTo(expected) {
		return reflect.Value{}, fmt.Errorf("%s is not assignable to %s", value.Type(), expected)
	}
	return value, nil
}
