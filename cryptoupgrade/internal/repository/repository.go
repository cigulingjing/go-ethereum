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

	wasmSubdir        = "wasm"
	compiledSubdir    = "compiled"
	algorithmInfoFile = "algorithm_info.json"
	versionInfoFile   = "algorithm_versions.json"
	directoryMode     = 0755
	fileMode          = 0644
)

// Workspace 描述动态算法 WASM 制品、编译缓存和 metadata 的本地存储位置。
type Workspace struct {
	BaseDir           string
	SourceDir         string
	SharedObjectDir   string
	AlgorithmInfoPath string
	VersionInfoPath   string
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
		SourceDir:         filepath.Join(baseDir, wasmSubdir),
		SharedObjectDir:   filepath.Join(baseDir, compiledSubdir),
		AlgorithmInfoPath: filepath.Join(baseDir, algorithmInfoFile),
		VersionInfoPath:   filepath.Join(baseDir, versionInfoFile),
	}
}

// EnsureDirs 创建编译缓存和 WASM 制品所需目录。
func (w Workspace) EnsureDirs() error {
	for _, dir := range []string{w.SourceDir, w.SharedObjectDir} {
		if err := os.MkdirAll(dir, directoryMode); err != nil {
			return fmt.Errorf("create cryptoupgrade artifact directory %s: %w", dir, err)
		}
	}
	return nil
}

// SourcePath 返回指定算法的 WASM 路径。
func (w Workspace) SourcePath(name string) string {
	return filepath.Join(w.SourceDir, name+".wasm")
}

// VersionSourcePath 返回指定算法版本的 WASM 路径。
func (w Workspace) VersionSourcePath(name string, version uint64) string {
	return filepath.Join(w.SourceDir, fmt.Sprintf("%s-%d.wasm", name, version))
}

// PluginPath 返回指定算法的编译缓存路径。
func (w Workspace) PluginPath(name string) string {
	return filepath.Join(w.SharedObjectDir, name)
}

// VersionPluginPath 返回指定算法版本的编译缓存路径。
func (w Workspace) VersionPluginPath(name string, version uint64) string {
	return filepath.Join(w.SharedObjectDir, fmt.Sprintf("%s-%d", name, version))
}

// Repository 管理活动算法、待激活算法以及 metadata 持久化。
type Repository struct {
	workspace Workspace

	mu               sync.RWMutex
	active           map[string]model.AlgorithmInfo
	uploaded         map[string]model.AlgorithmInfo
	activeVersion    map[string]model.AlgorithmVersionInfo
	activeVersions   map[string]map[uint64]model.AlgorithmVersionInfo
	uploadedVersions map[string]map[uint64]model.AlgorithmVersionInfo
}

// New 创建使用指定 workspace 的算法 metadata repository。
func New(workspace Workspace) *Repository {
	return &Repository{
		workspace:        workspace,
		active:           make(map[string]model.AlgorithmInfo),
		uploaded:         make(map[string]model.AlgorithmInfo),
		activeVersion:    make(map[string]model.AlgorithmVersionInfo),
		activeVersions:   make(map[string]map[uint64]model.AlgorithmVersionInfo),
		uploadedVersions: make(map[string]map[uint64]model.AlgorithmVersionInfo),
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

// SourcePath 返回指定算法的 WASM bytecode 路径。
func (r *Repository) SourcePath(name string) string {
	return r.workspace.SourcePath(name)
}

// VersionSourcePath 返回指定算法版本的 WASM bytecode 路径。
func (r *Repository) VersionSourcePath(name string, version uint64) string {
	return r.workspace.VersionSourcePath(name, version)
}

// PluginPath 返回指定算法的 WASM 编译缓存路径。
func (r *Repository) PluginPath(name string) string {
	return r.workspace.PluginPath(name)
}

// VersionPluginPath 返回指定算法版本的 WASM 编译缓存路径。
func (r *Repository) VersionPluginPath(name string, version uint64) string {
	return r.workspace.VersionPluginPath(name, version)
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
	versionInfo := model.LegacyVersion(info)
	r.activeVersion[name] = versionInfo
	r.setActiveVersionLocked(name, versionInfo)
}

// DeleteActive 删除进程内已激活算法 metadata。
func (r *Repository) DeleteActive(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, name)
	delete(r.activeVersion, name)
	delete(r.activeVersions, name)
}

// SetUploaded 更新进程内待激活算法 metadata。
func (r *Repository) SetUploaded(name string, info model.AlgorithmInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploaded[name] = info
	r.setUploadedVersionLocked(name, model.LegacyVersion(info))
}

// UploadedVersion 返回指定算法版本的待激活 metadata。
func (r *Repository) UploadedVersion(name string, version uint64) (model.AlgorithmVersionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	versions := r.uploadedVersions[name]
	if versions != nil {
		if info, ok := versions[version]; ok {
			return info, true
		}
	}
	if version == 1 {
		if info, ok := r.uploaded[name]; ok {
			return model.LegacyVersion(info), true
		}
		if info, ok := r.active[name]; ok {
			return model.LegacyVersion(info), true
		}
	}
	return model.AlgorithmVersionInfo{}, false
}

// SetUploadedVersion 更新指定算法版本的待激活 metadata。
func (r *Repository) SetUploadedVersion(name string, info model.AlgorithmVersionInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setUploadedVersionLocked(name, info)
	if info.Version == 1 && info.ActivationBlock == 0 {
		r.uploaded[name] = info.Base()
	}
}

// ActiveVersion 返回当前本地已激活版本 metadata。
func (r *Repository) ActiveVersion(name string) (model.AlgorithmVersionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if info, ok := r.activeVersion[name]; ok {
		return info, true
	}
	if versions := r.activeVersions[name]; versions != nil {
		var selected model.AlgorithmVersionInfo
		var ok bool
		for _, info := range versions {
			if !ok || info.Version > selected.Version {
				selected = info
				ok = true
			}
		}
		if ok {
			return selected, true
		}
	}
	if info, ok := r.active[name]; ok {
		return model.LegacyVersion(info), true
	}
	return model.AlgorithmVersionInfo{}, false
}

// PreparedVersion 返回本地已经编译并加载过的指定算法版本。
func (r *Repository) PreparedVersion(name string, version uint64) (model.AlgorithmVersionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	versions := r.activeVersions[name]
	if versions != nil {
		if info, ok := versions[version]; ok && info.IsPrepared() {
			return info, true
		}
	}
	if version == 1 {
		if info, ok := r.active[name]; ok {
			if info := model.LegacyVersion(info); info.IsPrepared() {
				return info, true
			}
		}
	}
	return model.AlgorithmVersionInfo{}, false
}

// DeletePreparedVersion 删除指定版本的本地 prepared metadata。
func (r *Repository) DeletePreparedVersion(name string, version uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if versions := r.activeVersions[name]; versions != nil {
		delete(versions, version)
		if len(versions) == 0 {
			delete(r.activeVersions, name)
		}
	}
	current, ok := r.activeVersion[name]
	if !ok || current.Version != version {
		return
	}
	delete(r.activeVersion, name)
	delete(r.active, name)
	if selected, ok := r.latestActiveVersionLocked(name); ok {
		r.activeVersion[name] = selected
		r.active[name] = selected.Base()
	}
}

// SetActiveVersion 更新当前本地已激活版本 metadata。
func (r *Repository) SetActiveVersion(name string, info model.AlgorithmVersionInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.setActiveVersionLocked(name, info)
	r.activeVersion[name] = info
	r.active[name] = info.Base()
}

// DeleteActiveVersion 删除当前本地已激活版本 metadata。
func (r *Repository) DeleteActiveVersion(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.activeVersion, name)
	delete(r.activeVersions, name)
	delete(r.active, name)
}

// ActiveVersionAt 返回指定区块高度下本地已准备且应生效的版本。
func (r *Repository) ActiveVersionAt(name string, blockNumber uint64) (model.AlgorithmVersionInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeVersionAtLocked(name, blockNumber)
}

func (r *Repository) setUploadedVersionLocked(name string, info model.AlgorithmVersionInfo) {
	if r.uploadedVersions[name] == nil {
		r.uploadedVersions[name] = make(map[uint64]model.AlgorithmVersionInfo)
	}
	r.uploadedVersions[name][info.Version] = info
}

func (r *Repository) setActiveVersionLocked(name string, info model.AlgorithmVersionInfo) {
	if r.activeVersions[name] == nil {
		r.activeVersions[name] = make(map[uint64]model.AlgorithmVersionInfo)
	}
	r.activeVersions[name][info.Version] = info
}

func (r *Repository) activeVersionAtLocked(name string, blockNumber uint64) (model.AlgorithmVersionInfo, bool) {
	var selected model.AlgorithmVersionInfo
	var ok bool
	if versions := r.uploadedVersions[name]; versions != nil {
		for _, info := range versions {
			if info.ActivationBlock > blockNumber {
				continue
			}
			if !ok || info.Version > selected.Version || (info.Version == selected.Version && info.ActivationBlock > selected.ActivationBlock) {
				selected = info
				ok = true
			}
		}
	}
	if versions := r.activeVersions[name]; versions != nil {
		for _, info := range versions {
			if info.ActivationBlock > blockNumber {
				continue
			}
			if !ok || info.Version > selected.Version || (info.Version == selected.Version && info.ActivationBlock > selected.ActivationBlock) {
				selected = info
				ok = true
			}
		}
	}
	if !ok {
		if info, exists := r.active[name]; exists && blockNumber == 0 {
			return model.LegacyVersion(info), true
		}
	}
	return selected, ok
}

func (r *Repository) latestActiveVersionLocked(name string) (model.AlgorithmVersionInfo, bool) {
	var selected model.AlgorithmVersionInfo
	var ok bool
	for _, info := range r.activeVersions[name] {
		if !ok || info.Version > selected.Version {
			selected = info
			ok = true
		}
	}
	return selected, ok
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
	if info, ok := r.activeVersion[name]; ok {
		info.Gas = gas
		r.activeVersion[name] = info
	}
	if versions := r.activeVersions[name]; versions != nil {
		for version, info := range versions {
			info.Gas = gas
			versions[version] = info
		}
	}
	if versions := r.uploadedVersions[name]; versions != nil {
		for version, info := range versions {
			info.Gas = gas
			versions[version] = info
		}
	}
	return found, activeUpdated
}

// Save 将活动算法 metadata 原子持久化到默认 algorithm_info.json。
func (r *Repository) Save() error {
	if err := r.workspace.EnsureDirs(); err != nil {
		return err
	}
	if err := r.SaveTo(r.workspace.AlgorithmInfoPath); err != nil {
		return err
	}
	return r.SaveVersionsTo(r.workspace.VersionInfoPath)
}

// SaveTo 将活动算法 metadata 原子持久化到指定文件。
func (r *Repository) SaveTo(filename string) error {
	r.mu.RLock()
	data, err := json.MarshalIndent(r.active, "", "  ")
	r.mu.RUnlock()
	if err != nil {
		return err
	}
	return writeFileAtomic(filename, data)
}

// SaveVersionsTo 将版本化 metadata 原子持久化到指定文件。
func (r *Repository) SaveVersionsTo(filename string) error {
	r.mu.RLock()
	data, err := json.MarshalIndent(r.activeVersions, "", "  ")
	r.mu.RUnlock()
	if err != nil {
		return err
	}
	return writeFileAtomic(filename, data)
}

// Load 从默认 algorithm_info.json 恢复活动算法 metadata。
func (r *Repository) Load() error {
	if err := r.LoadFrom(r.workspace.AlgorithmInfoPath); err != nil {
		return err
	}
	return r.LoadVersionsFrom(r.workspace.VersionInfoPath)
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
	for name, info := range loaded {
		if _, ok := r.activeVersion[name]; !ok {
			r.activeVersion[name] = model.LegacyVersion(info)
		}
	}
	r.mu.Unlock()
	return nil
}

// LoadVersionsFrom 从指定文件恢复版本化活动 metadata。
func (r *Repository) LoadVersionsFrom(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	loaded := make(map[string]map[uint64]model.AlgorithmVersionInfo)
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	r.mu.Lock()
	r.activeVersions = loaded
	r.activeVersion = make(map[string]model.AlgorithmVersionInfo, len(loaded))
	for name, versions := range loaded {
		var selected model.AlgorithmVersionInfo
		var ok bool
		for _, info := range versions {
			if !ok || info.Version > selected.Version {
				selected = info
				ok = true
			}
		}
		if ok {
			r.activeVersion[name] = selected
			r.active[name] = selected.Base()
		}
	}
	r.mu.Unlock()
	return nil
}

func writeFileAtomic(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return err
	}
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
