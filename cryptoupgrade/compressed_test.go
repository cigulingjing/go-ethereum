package cryptoupgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeSourceFileMatchesEncodeSource(t *testing.T) {
	source := []byte("package main\n\nfunc Add(a, b uint64) uint64 { return a + b }\n")
	path := filepath.Join(t.TempDir(), "add.go")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	fromBytes, err := EncodeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	fromFile, err := EncodeSourceFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fromFile != fromBytes {
		t.Fatal("file and byte source encoding differ")
	}
}
