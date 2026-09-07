package cryptoupgrade

import (
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/repository"
)

// pluginDirEnvVar is read during package initialization to override the
// default cryptoupgrade plugin artifact directory.
const pluginDirEnvVar = repository.PluginDirEnvVar

type pluginPaths struct {
	baseDir           string
	sourceDir         string
	sharedObjectDir   string
	algorithmInfoPath string
	versionInfoPath   string
}

var runtimePluginPaths, runtimePluginPathsErr = resolvePluginPaths(os.Getenv(pluginDirEnvVar), "")

func resolvePluginPaths(configuredDir, cwd string) (pluginPaths, error) {
	workspace, err := repository.ResolveWorkspace(configuredDir, cwd)
	return pluginPathsFromWorkspace(workspace), err
}

func newPluginPaths(baseDir string) pluginPaths {
	return pluginPathsFromWorkspace(repository.NewWorkspace(baseDir))
}

func pluginPathsFromWorkspace(workspace repository.Workspace) pluginPaths {
	return pluginPaths{
		baseDir:           workspace.BaseDir,
		sourceDir:         workspace.SourceDir,
		sharedObjectDir:   workspace.SharedObjectDir,
		algorithmInfoPath: workspace.AlgorithmInfoPath,
		versionInfoPath:   workspace.VersionInfoPath,
	}
}

func (paths pluginPaths) ensureDirs() error {
	return paths.workspace().EnsureDirs()
}

func (paths pluginPaths) workspace() repository.Workspace {
	return repository.Workspace{
		BaseDir:           paths.baseDir,
		SourceDir:         paths.sourceDir,
		SharedObjectDir:   paths.sharedObjectDir,
		AlgorithmInfoPath: paths.algorithmInfoPath,
		VersionInfoPath:   paths.versionInfoPath,
	}
}

func pluginBaseDir() string {
	return runtimePluginPaths.baseDir
}

func pluginSourceDir() string {
	return runtimePluginPaths.sourceDir
}

func pluginSharedObjectDir() string {
	return runtimePluginPaths.sharedObjectDir
}

func algorithmInfoPath() string {
	return runtimePluginPaths.algorithmInfoPath
}

func gofilePath(fileName string) string {
	return filepath.Join(pluginSourceDir(), fileName+".wasm")
}

func sofilePath(fileName string) string {
	return filepath.Join(pluginSharedObjectDir(), fileName)
}

func directoryInit() error {
	if runtimePluginPathsErr != nil {
		return runtimePluginPathsErr
	}
	return runtimePluginPaths.ensureDirs()
}
