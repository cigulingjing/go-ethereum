package cryptoupgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// pluginDirEnvVar is read during package initialization to override the
	// default cryptoupgrade plugin artifact directory.
	pluginDirEnvVar = "GETH_CRYPTOUPGRADE_PLUGIN_DIR"

	defaultPluginBaseDir      = "./plugin"
	pluginSourceSubdir        = "src"
	pluginSharedObjectSubdir  = "so"
	pluginAlgorithmInfoFile   = "algorithm_info.json"
	pluginDirectoryPermission = 0755
)

type pluginPaths struct {
	baseDir           string
	sourceDir         string
	sharedObjectDir   string
	algorithmInfoPath string
}

var runtimePluginPaths, runtimePluginPathsErr = resolvePluginPaths(os.Getenv(pluginDirEnvVar), "")

func resolvePluginPaths(configuredDir, cwd string) (pluginPaths, error) {
	baseDir := strings.TrimSpace(configuredDir)
	if baseDir == "" {
		baseDir = defaultPluginBaseDir
	}
	if !filepath.IsAbs(baseDir) {
		if cwd == "" {
			var err error
			cwd, err = os.Getwd()
			if err != nil {
				return newPluginPaths(baseDir), fmt.Errorf("resolve cryptoupgrade plugin base directory: %w", err)
			}
		}
		baseDir = filepath.Join(cwd, baseDir)
	}
	return newPluginPaths(baseDir), nil
}

func newPluginPaths(baseDir string) pluginPaths {
	baseDir = filepath.Clean(baseDir)
	return pluginPaths{
		baseDir:           baseDir,
		sourceDir:         filepath.Join(baseDir, pluginSourceSubdir),
		sharedObjectDir:   filepath.Join(baseDir, pluginSharedObjectSubdir),
		algorithmInfoPath: filepath.Join(baseDir, pluginAlgorithmInfoFile),
	}
}

func (paths pluginPaths) ensureDirs() error {
	for _, dir := range []string{paths.sourceDir, paths.sharedObjectDir} {
		if err := os.MkdirAll(dir, pluginDirectoryPermission); err != nil {
			return fmt.Errorf("create cryptoupgrade plugin directory %s: %w", dir, err)
		}
	}
	return nil
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
	return filepath.Join(pluginSourceDir(), fileName+".go")
}

func sofilePath(fileName string) string {
	return filepath.Join(pluginSharedObjectDir(), fileName+".so")
}

func directoryInit() error {
	if runtimePluginPathsErr != nil {
		return runtimePluginPathsErr
	}
	return runtimePluginPaths.ensureDirs()
}
