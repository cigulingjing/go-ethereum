package cryptoupgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeWasmFileMatchesEncodeWasm(t *testing.T) {
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	path := filepath.Join(t.TempDir(), "add.wasm")
	if err := os.WriteFile(path, wasm, 0600); err != nil {
		t.Fatal(err)
	}
	fromBytes, err := EncodeWasm(wasm)
	if err != nil {
		t.Fatal(err)
	}
	fromFile, err := EncodeWasmFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fromFile != fromBytes {
		t.Fatal("file and byte wasm encoding differ")
	}
}
