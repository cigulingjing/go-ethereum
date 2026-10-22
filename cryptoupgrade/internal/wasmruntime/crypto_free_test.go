package wasmruntime

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestExperimentCryptoWasmExecute(t *testing.T) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	uintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "..", "experiments", "cryptoupgrade", "algorithm", "go", "wasm")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	loader := NewLoader(Config{})
	cache := filepath.Join(t.TempDir(), "cache")
	invoke := func(name string, inputs abi.Arguments, outputs abi.Arguments, args ...interface{}) interface{} {
		t.Helper()
		path := filepath.Join(root, name+".wasm")
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
		input, err := inputs.Pack(args...)
		if err != nil {
			t.Fatal(err)
		}
		output, err := loader.Execute(ctx, path, cache, input)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		decoded, err := outputs.Unpack(output)
		if err != nil {
			t.Fatal(err)
		}
		return decoded[0]
	}
	byteArgs := abi.Arguments{{Type: bytesType}, {Type: bytesType}}
	byteResult := abi.Arguments{{Type: bytesType}}
	derived := invoke("pbkdf2_sha256", abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: uintType}, {Type: uintType}}, byteResult,
		[]byte("password"), []byte("salt"), big.NewInt(2), big.NewInt(64)).([]byte)
	if len(derived) != 64 || !bytes.Equal(derived[:32], []byte{0xae, 0x4d, 0x0c, 0x95, 0xaf, 0x6b, 0x46, 0xd3, 0x2d, 0x0a, 0xdf, 0xf9, 0x28, 0xf0, 0x6d, 0xd0, 0x2a, 0x30, 0x3f, 0x8e, 0xf3, 0xc2, 0x51, 0xdf, 0xd6, 0xe2, 0xd8, 0x5a, 0x95, 0x47, 0x4c, 0x43}) {
		t.Fatalf("PBKDF2 WASM output %x", derived)
	}
	derived = invoke("pbkdf2_sha256", abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: uintType}, {Type: uintType}}, byteResult,
		[]byte("password"), []byte("salt"), big.NewInt(2000), big.NewInt(32)).([]byte)
	if len(derived) != 32 {
		t.Fatalf("PBKDF2 2000-iteration WASM output length %d", len(derived))
	}
	commitment := invoke("pedersen_commit", byteArgs, byteResult, []byte{1}, []byte{2}).([]byte)
	if len(commitment) != 256 {
		t.Fatalf("Pedersen WASM output length %d", len(commitment))
	}
	seed := bytes.Repeat([]byte{7}, 32)
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	message := bytes.Repeat([]byte("a"), 120)
	signature := ed25519.Sign(private, message)
	verifyArgs := abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}
	verifyResult := abi.Arguments{{Type: boolType}}
	if !invoke("ed25519", verifyArgs, verifyResult, []byte(public), message, signature).(bool) {
		t.Fatal("valid Ed25519 WASM signature rejected")
	}
	message[0] ^= 1
	if invoke("ed25519", verifyArgs, verifyResult, []byte(public), message, signature).(bool) {
		t.Fatal("invalid Ed25519 WASM signature accepted")
	}
}
