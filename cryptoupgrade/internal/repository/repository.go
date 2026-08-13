package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

const (
	PluginDirEnvVar = "GETH_CRYPTOUPGRADE_PLUGIN_DIR"
	DefaultBaseDir  = "./plugin"

	sourceSubdir       = "src"
	sharedObjectSubdir = "so"
	algorithmInfoFile  = "algorithm_info.json"
	directoryMode      = 0755
	fileMode           = 0644
)

// Workspace 描述动态算法源码、plugin 制品和 metadata 的本地存储位置。
type Workspace struct {
	BaseDir           string
	SourceDir         string
	SharedObjectDir   string
	AlgorithmInfoPath string
}

// ResolveWorkspace 按既有环境变量语义解析 plugin workspace。
func ResolveWorkspace(configuredDir, cwd string) (Workspace, error) {
	baseDir := strings.TrimSpace(configuredDir)
	if baseDir == "" {
		baseDir = DefaultBaseDir
	}
	if !filepath.IsAbs(baseDir) {
		if cwd == "" {
			var err error
			cwd, err = os.Getwd()
			if err != nil {
				return NewWorkspace(baseDir), fmt.Errorf("resolve cryptoupgrade plugin base directory: %w", err)
			}
		}
		baseDir = filepath.Join(cwd, baseDir)
	}
	return NewWorkspace(baseDir), nil
}

// ResolveWorkspaceFromEnvironment 在进程启动配置中解析 plugin workspace。
func ResolveWorkspaceFromEnvironment() (Workspace, error) {
	return ResolveWorkspace(os.Getenv(PluginDirEnvVar), "")
}

// NewWorkspace 根据明确的根目录创建 workspace。
func NewWorkspace(baseDir string) Workspace {
	baseDir = filepath.Clean(baseDir)
	return Workspace{
		BaseDir:           baseDir,
		SourceDir:         filepath.Join(baseDir, sourceSubdir),
		SharedObjectDir:   filepath.Join(baseDir, sharedObjectSubdir),
		AlgorithmInfoPath: filepath.Join(baseDir, algorithmInfoFile),
	}
}

// EnsureDirs 创建编译源码和 plugin 制品所需目录。
func (w Workspace) EnsureDirs() error {
	for _, dir := range []string{w.SourceDir, w.SharedObjectDir} {
		if err := os.MkdirAll(dir, directoryMode); err != nil {
			return fmt.Errorf("create cryptoupgrade plugin directory %s: %w", dir, err)
		}
	}
	return nil
}

// SourcePath 返回指定算法的 Go 源码路径。
func (w Workspace) SourcePath(name string) string {
	return filepath.Join(w.SourceDir, name+".go")
}

// PluginPath 返回指定算法的 plugin 制品路径。
func (w Workspace) PluginPath(name string) string {
	return filepath.Join(w.SharedObjectDir, name+".so")
}

// Repository 管理活动算法、待激活算法以及 metadata 持久化。
type Repository struct {
	workspace Workspace

	mu       sync.RWMutex
	active   map[string]model.AlgorithmInfo
	uploaded map[string]model.AlgorithmInfo
}

// New 创建使用指定 workspace 的算法 metadata repository。
func New(workspace Workspace) *Repository {
	return &Repository{
		workspace: workspace,
		active:    make(map[string]model.AlgorithmInfo),
		uploaded:  make(map[string]model.AlgorithmInfo),
	}
}

// Workspace 返回 repository 使用的本地 workspace。
func (r *Repository) Workspace() Workspace {
	return r.workspace
}

// EnsureDirs 创建 repository 编译和持久化所需目录。
func (r *Repository) EnsureDirs() error {
	return r.workspace.EnsureDirs()
}

// SourcePath 返回指定算法的源码路径。
func (r *Repository) SourcePath(name string) string {
	return r.workspace.SourcePath(name)
}

// PluginPath 返回指定算法的 canonical plugin 路径。
func (r *Repository) PluginPath(name string) string {
	return r.workspace.PluginPath(name)
}

// Active 返回已激活算法的 metadata。
func (r *Repository) Active(name string) (model.AlgorithmInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.active[name]
	return info, ok
}

// Uploaded 返回待激活 metadata；不存在时回退到已激活版本。
func (r *Repository) Uploaded(name string) (model.AlgorithmInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.uploaded[name]; ok {
		return info, true
	}
	info, ok := r.active[name]
	return info, ok
}

// SetActive 更新进程内已激活算法 metadata。
func (r *Repository) SetActive(name string, info model.AlgorithmInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[name] = info
}

// DeleteActive 删除进程内已激活算法 metadata。
func (r *Repository) DeleteActive(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, name)
}

// SetUploaded 更新进程内待激活算法 metadata。
func (r *Repository) SetUploaded(name string, info model.AlgorithmInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploaded[name] = info
}

// UpdateGas 同步更新待激活及活动版本的 gas。
func (r *Repository) UpdateGas(name string, gas uint64) (found, activeUpdated bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.uploaded[name]; ok {
		info.Gas = gas
		r.uploaded[name] = info
		found = true
	}
	if info, ok := r.active[name]; ok {
		info.Gas = gas
		r.active[name] = info
		found = true
		activeUpdated = true
	}
	return found, activeUpdated
}

// Save 将活动算法 metadata 原子持久化到默认 algorithm_info.json。
func (r *Repository) Save() error {
	if err := r.workspace.EnsureDirs(); err != nil {
		return err
	}
	return r.SaveTo(r.workspace.AlgorithmInfoPath)
}

// SaveTo 将活动算法 metadata 原子持久化到指定文件。
func (r *Repository) SaveTo(filename string) error {
	r.mu.RLock()
	data, err := json.MarshalIndent(r.active, "", "  ")
	r.mu.RUnlock()
	if err != nil {
		return err
	}

	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(filename)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(fileMode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// 同目录 rename 保证读取方不会观察到部分写入的 JSON。
	return os.Rename(tmpName, filename)
}

// Load 从默认 algorithm_info.json 恢复活动算法 metadata。
func (r *Repository) Load() error {
	return r.LoadFrom(r.workspace.AlgorithmInfoPath)
}

// LoadFrom 从指定文件恢复活动算法 metadata。
func (r *Repository) LoadFrom(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	loaded := make(map[string]model.AlgorithmInfo)
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	r.mu.Lock()
	r.active = loaded
	r.mu.Unlock()
	return nil
}
