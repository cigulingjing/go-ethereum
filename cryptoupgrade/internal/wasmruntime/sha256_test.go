package wasmruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestSha256WasmExecute(t *testing.T) {
	wasmPath := filepath.Join("..", "..", "..", "experiments", "cryptoupgrade", "algorithm", "go", "wasm", "sha256.wasm")
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(wasm) >= 16<<10 {
		t.Fatalf("SHA-256 WASM size = %d, want < 16384", len(wasm))
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	arguments := abi.Arguments{{Type: bytesType}}
	loader := NewLoader(Config{})
	ctx := context.Background()
	cachePath := filepath.Join(t.TempDir(), "compiled")
	for _, length := range []int{0, 1, 55, 56, 64, 65, 128, 192, 193, 256, 1024, 4096, 65536} {
		message := bytes.Repeat([]byte{0x61}, length)
		input, err := arguments.Pack(message)
		if err != nil {
			t.Fatal(err)
		}
		output, err := loader.Execute(ctx, wasmPath, cachePath, input)
		if err != nil {
			t.Fatalf("length %d: execute: %v", length, err)
		}
		decoded, err := arguments.Unpack(output)
		if err != nil {
			t.Fatalf("length %d: unpack: %v", length, err)
		}
		want := sha256.Sum256(message)
		if !bytes.Equal(decoded[0].([]byte), want[:]) {
			t.Errorf("length %d: digest %x, want %x", length, decoded[0], want)
		}
	}
}
