//go:build linux || darwin || freebsd

package pluginruntime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultLoaderRejectsInvalidPluginFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.so")
	if err := os.WriteFile(path, []byte("not a go plugin"), 0644); err != nil {
		t.Fatalf("write invalid plugin: %v", err)
	}
	loader := NewLoader(openPlugin)
	if _, err := loader.Lookup(path, "Add"); err == nil {
		t.Fatal("expected invalid plugin file error")
	}
}
