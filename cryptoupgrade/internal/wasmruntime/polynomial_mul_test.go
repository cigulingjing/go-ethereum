package wasmruntime

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestPolynomialMulWasmExecute(t *testing.T) {
	wasmPath := filepath.Join("..", "..", "..", "experiments", "cryptoupgrade", "algorithm", "go", "polynomial_mul.wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		t.Skipf("polynomial_mul.wasm fixture unavailable: %v", err)
	}
	cachePath := filepath.Join(t.TempDir(), "compiled")

	loader := NewLoader(Config{MemoryLimitPages: 256, ExecutionBudget: 10 * time.Second})
	ctx := context.Background()
	if err := loader.Activate(ctx, wasmPath, cachePath); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	left := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}
	right := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}
	mod := big.NewInt(12289)
	uint256Ty, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatalf("NewType uint256: %v", err)
	}
	uint256Arr, err := abi.NewType("uint256[]", "", nil)
	if err != nil {
		t.Fatalf("NewType uint256[]: %v", err)
	}
	input, err := abi.Arguments{{Type: uint256Arr}, {Type: uint256Arr}, {Type: uint256Ty}}.Pack(left, right, mod)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	output, err := loader.Execute(ctx, wasmPath, cachePath, input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("empty output")
	}
}

func TestPolynomialMulWasmExecuteP6Profile(t *testing.T) {
	wasmPath := filepath.Join("..", "..", "..", "experiments", "cryptoupgrade", "algorithm", "go", "polynomial_mul.wasm")
	if _, err := os.Stat(wasmPath); err != nil {
		t.Skipf("polynomial_mul.wasm fixture unavailable: %v", err)
	}
	cachePath := filepath.Join(t.TempDir(), "compiled")
	loader := NewLoader(Config{MemoryLimitPages: 512, ExecutionBudget: 30 * time.Second})
	ctx := context.Background()
	if err := loader.Activate(ctx, wasmPath, cachePath); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	left := make([]*big.Int, 10)
	right := make([]*big.Int, 10)
	for i := range left {
		left[i] = big.NewInt(int64(i + 1))
		right[i] = big.NewInt(int64(i + 1))
	}
	mod := big.NewInt(12289)
	uint256Ty, _ := abi.NewType("uint256", "", nil)
	uint256Arr, _ := abi.NewType("uint256[]", "", nil)
	input, err := abi.Arguments{{Type: uint256Arr}, {Type: uint256Arr}, {Type: uint256Ty}}.Pack(left, right, mod)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	output, err := loader.Execute(ctx, wasmPath, cachePath, input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(output) == 0 {
		t.Fatal("empty output")
	}
}
