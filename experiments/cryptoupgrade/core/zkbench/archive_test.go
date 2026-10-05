package zkbench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureSizesCountsOnlyAuthoredC(t *testing.T) {
	root := t.TempDir()
	algo := "groth16_dummy_verify"
	native := filepath.Join(root, algo, "native")
	if err := os.MkdirAll(filepath.Join(native, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(native, "include"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "verify.c"), []byte("int verify(void){return 0;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "check.c"), []byte("int main(){return 0;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "verify.c.bak"), []byte("backup should not count\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "src", "fp.cpp"), []byte("library should not count\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "include", "mcl.h"), []byte("header should not count\n"), 0o644); err != nil {
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
