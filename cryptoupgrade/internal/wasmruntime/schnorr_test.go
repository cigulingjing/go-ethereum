package wasmruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestSchnorrWasmExecute(t *testing.T) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	inputs := abi.Arguments{{Type: bytesType}, {Type: bytesType}}
	outputs := abi.Arguments{{Type: boolType}}
	path := filepath.Join("..", "..", "..", "experiments", "cryptoupgrade", "algorithm", "go", "wasm", "schnorr_proof.wasm")
	loader := NewLoader(Config{})
	cache := filepath.Join(t.TempDir(), "compiled")

	for _, length := range []int{0, 1, 56, 192, 4096} {
		message := bytes.Repeat([]byte{0x73}, length)
		proof := make([]byte, 3*256)
		proof[255] = 4
		proof[511] = 4
		challengeInput := append([]byte("cryptoupgrade-schnorr"), proof[:512]...)
		challengeInput = append(challengeInput, message...)
		digest := sha256.Sum256(challengeInput)
		response := new(big.Int).Add(new(big.Int).SetBytes(digest[:]), big.NewInt(1)).Bytes()
		copy(proof[len(proof)-len(response):], response)
		verify := func(message, proof []byte) bool {
			t.Helper()
			input, err := inputs.Pack(message, proof)
			if err != nil {
				t.Fatal(err)
			}
			output, err := loader.Execute(context.Background(), path, cache, input)
			if err != nil {
				t.Fatalf("message length %d: %v", len(message), err)
			}
			values, err := outputs.Unpack(output)
			if err != nil {
				t.Fatal(err)
			}
			return values[0].(bool)
		}
		if !verify(message, proof) {
			t.Fatalf("message length %d: valid proof rejected", length)
		}
		if verify(append(append([]byte(nil), message...), 1), proof) {
			t.Fatalf("message length %d: altered message accepted", length)
		}
		if verify(message, proof[:len(proof)-1]) {
			t.Fatalf("message length %d: short proof accepted", length)
		}
	}
}
