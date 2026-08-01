package cryptoupgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePluginPathsDefault(t *testing.T) {
	cwd := t.TempDir()
	paths, err := resolvePluginPaths("", cwd)
	if err != nil {
		t.Fatalf("resolvePluginPaths failed: %v", err)
	}
	want := newPluginPaths(filepath.Join(cwd, "plugin"))
	if paths != want {
		t.Fatalf("unexpected default paths\nwant: %#v\n got: %#v", want, paths)
	}
}

func TestResolvePluginPathsOverride(t *testing.T) {
	absoluteBase := filepath.Join(t.TempDir(), "cryptoupgrade-plugin")
	paths, err := resolvePluginPaths(absoluteBase, t.TempDir())
	if err != nil {
		t.Fatalf("resolvePluginPaths absolute override failed: %v", err)
	}
	if paths != newPluginPaths(absoluteBase) {
		t.Fatalf("unexpected absolute override paths: %#v", paths)
	}

	cwd := t.TempDir()
	paths, err = resolvePluginPaths("node-a-plugin", cwd)
	if err != nil {
		t.Fatalf("resolvePluginPaths relative override failed: %v", err)
	}
	want := newPluginPaths(filepath.Join(cwd, "node-a-plugin"))
	if paths != want {
		t.Fatalf("unexpected relative override paths\nwant: %#v\n got: %#v", want, paths)
	}
}

func TestAlgorithmPathsUseRuntimeResolver(t *testing.T) {
	base := filepath.Join(t.TempDir(), "plugin")
	withRuntimePluginPaths(t, newPluginPaths(base), nil)

	if want := filepath.Join(base, "src", "Sha256.go"); gofilePath("Sha256") != want {
		t.Fatalf("unexpected source path: want %s got %s", want, gofilePath("Sha256"))
	}
	if want := filepath.Join(base, "so", "Sha256.so"); sofilePath("Sha256") != want {
		t.Fatalf("unexpected plugin path: want %s got %s", want, sofilePath("Sha256"))
	}
	if want := filepath.Join(base, "algorithm_info.json"); algorithmInfoPath() != want {
		t.Fatalf("unexpected metadata path: want %s got %s", want, algorithmInfoPath())
	}
}

func TestPluginPathsEnsureDirs(t *testing.T) {
	paths := newPluginPaths(filepath.Join(t.TempDir(), "plugin"))
	if err := paths.ensureDirs(); err != nil {
		t.Fatalf("ensureDirs failed: %v", err)
	}
	for _, dir := range []string{paths.sourceDir, paths.sharedObjectDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s failed: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}

func TestPluginPathsEnsureDirsError(t *testing.T) {
	base := filepath.Join(t.TempDir(), "plugin")
	if err := os.WriteFile(base, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	paths := newPluginPaths(base)
	err := paths.ensureDirs()
	if err == nil {
		t.Fatal("expected ensureDirs to fail")
	}
	if !strings.Contains(err.Error(), paths.sourceDir) {
		t.Fatalf("expected error to include source dir %q, got %v", paths.sourceDir, err)
	}
}

func TestStorePropagatesDirectoryInitError(t *testing.T) {
	base := filepath.Join(t.TempDir(), "plugin")
	if err := os.WriteFile(base, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	paths := newPluginPaths(base)
	withRuntimePluginPaths(t, paths, nil)

	err := Store()
	if err == nil {
		t.Fatal("expected Store to fail")
	}
	if !strings.Contains(err.Error(), paths.sourceDir) {
		t.Fatalf("expected Store error to include source dir %q, got %v", paths.sourceDir, err)
	}
}

func TestActivateAlgorithmPropagatesDirectoryInitError(t *testing.T) {
	base := filepath.Join(t.TempDir(), "plugin")
	if err := os.WriteFile(base, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	paths := newPluginPaths(base)
	withRuntimePluginPaths(t, paths, nil)

	err := ActivateAlgorithm("Sha256", algoInfo{})
	if err == nil {
		t.Fatal("expected ActivateAlgorithm to fail")
	}
	if !strings.Contains(err.Error(), paths.sourceDir) {
		t.Fatalf("expected ActivateAlgorithm error to include source dir %q, got %v", paths.sourceDir, err)
	}
}

func withRuntimePluginPaths(t *testing.T, paths pluginPaths, pathErr error) {
	t.Helper()

	oldPaths := runtimePluginPaths
	oldErr := runtimePluginPathsErr
	runtimePluginPaths = paths
	runtimePluginPathsErr = pathErr
	t.Cleanup(func() {
		runtimePluginPaths = oldPaths
		runtimePluginPathsErr = oldErr
	})
}
