package pqcbench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureSizesCountsOnlyVerifyC(t *testing.T) {
	root := t.TempDir()
	algo := "aigis_sig2_verify"
	native := filepath.Join(root, algo, "native")
	if err := os.MkdirAll(filepath.Join(native, "include"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "verify.c"), []byte("int verify(void){return 0;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "poly.c"), []byte("library should not count\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "verify.c.bak"), []byte("backup should not count\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "include", "api.h"), []byte("header should not count\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wasm := []byte("dummy-wasm")
	if err := os.WriteFile(filepath.Join(root, algo, algo+".wasm"), wasm, 0o644); err != nil {
		t.Fatal(err)
	}

	sizes, err := MeasureSizes(root, algo)
	if err != nil {
		t.Fatal(err)
	}
	wantNative := int64(len("int verify(void){return 0;}\n"))
	if sizes.NativeBytes != wantNative {
		t.Fatalf("nativeBytes=%d want %d", sizes.NativeBytes, wantNative)
	}
	if sizes.WASMBytes != int64(len(wasm)) {
		t.Fatalf("wasmBytes=%d want %d", sizes.WASMBytes, len(wasm))
	}
}
