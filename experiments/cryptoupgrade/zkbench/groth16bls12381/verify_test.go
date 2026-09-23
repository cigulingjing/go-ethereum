package groth16bls12381

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/zkbench"
)

func TestVerifyArchivedVector(t *testing.T) {
	if !Enabled() {
		t.Skip("cgo disabled")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	zkgoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "algorithm", "zkgo"))
	vectors, err := zkbench.LoadVectors(zkgoRoot, Name)
	if err != nil {
		t.Fatal(err)
	}
	okVerify, err := Verify(vectors.VK, vectors.Public, vectors.Proof)
	if err != nil {
		t.Fatal(err)
	}
	if !okVerify {
		t.Fatal("expected archived groth16 proof to verify")
	}
}
