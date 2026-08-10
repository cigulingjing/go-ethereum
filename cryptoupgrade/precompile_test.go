package cryptoupgrade

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestPrecompileRegistry(t *testing.T) {
	entries := Precompiles()
	if len(entries) == 0 {
		t.Fatal("expected precompile entries")
	}
	addresses := make(map[common.Address]string)
	names := make(map[string]common.Address)
	systemAddresses := map[common.Address]string{
		common.CodeStorageAddress:   "CodeStorage",
		common.MutiVoucherAddress:   "MutiVoucher",
		common.CoinBaseAddress:      "CoinBase",
		common.Blake2bSum256Address: "Blake2bSum256",
	}
	for _, entry := range entries {
		if other, ok := addresses[entry.Address()]; ok {
			t.Fatalf("duplicate precompile address %s for %s and %s", entry.Address(), other, entry.Name())
		}
		addresses[entry.Address()] = entry.Name()
		if other, ok := names[entry.Name()]; ok {
			t.Fatalf("duplicate precompile name %s for %s and %s", entry.Name(), other, entry.Address())
		}
		names[entry.Name()] = entry.Address()
		if systemName, ok := systemAddresses[entry.Address()]; ok {
			t.Fatalf("precompile %s collides with system address %s", entry.Name(), systemName)
		}
		mustMakeABIArguments(t, entry.InputTypes())
		mustMakeABIArguments(t, entry.OutputTypes())
		if entry.RequiredGas(nil) == 0 {
			t.Fatalf("precompile %s has zero gas for empty input", entry.Name())
		}
	}
	for _, excluded := range []string{"RandomBytes", "SchnorrProve", "ShamirSplit"} {
		if _, ok := PrecompileByName(excluded); ok {
			t.Fatalf("random or stateful entry %s must not be registered", excluded)
		}
	}
}

func TestPrecompileRejectsRandomFallbackInputs(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
	}{
		{name: "Dh2048Private", args: []interface{}{[]byte{}}},
		{name: "Ed25519Keygen", args: []interface{}{[]byte("short")}},
	}
	for _, test := range tests {
		entry, ok := PrecompileByName(test.name)
		if !ok {
			t.Fatalf("missing precompile %s", test.name)
		}
		input := mustPackABI(t, entry.InputTypes(), test.args...)
		if entry.RequiredGas(input) == 0 {
			t.Fatalf("%s returned zero gas for rejected fallback input", test.name)
		}
		if _, err := entry.Run(input); err == nil {
			t.Fatalf("%s accepted random fallback input", test.name)
		}
	}
}

func TestPrecompileRunFixtures(t *testing.T) {
	addOut := runPrecompile(t, "Add", big.NewInt(1), big.NewInt(2))
	if addOut[0].(*big.Int).Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("Add output mismatch: %v", addOut[0])
	}

	shaOut := runPrecompile(t, "Sha256", []byte("abc"))
	shaExpected := sha256.Sum256([]byte("abc"))
	if !bytes.Equal(shaOut[0].([]byte), shaExpected[:]) {
		t.Fatalf("Sha256 output mismatch")
	}

	key := []byte("0123456789abcdef")
	iv := []byte("abcdef9876543210")
	plaintext := []byte("hello cryptoupgrade")
	encrypted := runPrecompile(t, "AesCBCEncrypt", key, iv, plaintext)[0].([]byte)
	decrypted := runPrecompile(t, "AesCBCDecrypt", key, iv, encrypted)[0].([]byte)
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("AES-CBC round trip mismatch: %x", decrypted)
	}

	pbkdfOut := runPrecompile(t, "Pbkdf2Sha256", []byte("password"), []byte("salt"), big.NewInt(1), big.NewInt(32))[0].([]byte)
	if common.Bytes2Hex(pbkdfOut) != "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b" {
		t.Fatalf("PBKDF2 output mismatch: %x", pbkdfOut)
	}

	alicePrivate := runPrecompile(t, "Dh2048Private", []byte("alice"))[0].([]byte)
	bobPrivate := runPrecompile(t, "Dh2048Private", []byte("bob"))[0].([]byte)
	alicePublic := runPrecompile(t, "Dh2048Public", alicePrivate)[0].([]byte)
	bobPublic := runPrecompile(t, "Dh2048Public", bobPrivate)[0].([]byte)
	aliceSecret := runPrecompile(t, "Dh2048Secret", alicePrivate, bobPublic)[0].([]byte)
	bobSecret := runPrecompile(t, "Dh2048Secret", bobPrivate, alicePublic)[0].([]byte)
	if !bytes.Equal(aliceSecret, bobSecret) {
		t.Fatalf("DH2048 shared secrets differ")
	}

	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	privateKey := runPrecompile(t, "Ed25519Keygen", seed)[0].([]byte)
	publicKey := runPrecompile(t, "Ed25519PublicKey", privateKey)[0].([]byte)
	message := []byte("message")
	signature := runPrecompile(t, "Ed25519Sign", privateKey, message)[0].([]byte)
	verified := runPrecompile(t, "Ed25519Verify", publicKey, message, signature)[0].(bool)
	if !verified {
		t.Fatalf("Ed25519Verify returned false for valid signature")
	}

	commitment := runPrecompile(t, "PedersenCommit", []byte("msg"), []byte("blind"))[0].([]byte)
	commitmentOK := runPrecompile(t, "PedersenVerify", []byte("msg"), []byte("blind"), commitment)[0].(bool)
	if !commitmentOK {
		t.Fatalf("PedersenVerify returned false for valid commitment")
	}

	schnorrSecret := []byte("secret")
	schnorrMessage := []byte("schnorr-message")
	schnorrPublic := runPrecompile(t, "SchnorrPublicKey", schnorrSecret)[0].([]byte)
	if len(schnorrPublic) != precompileRFC3526FieldBytes() {
		t.Fatalf("unexpected Schnorr public key size %d", len(schnorrPublic))
	}
	proof := deterministicSchnorrProof(schnorrSecret, schnorrMessage, 7)
	schnorrOK := runPrecompile(t, "SchnorrVerify", schnorrMessage, proof)[0].(bool)
	if !schnorrOK {
		t.Fatalf("SchnorrVerify returned false for valid proof")
	}
}

func runPrecompile(t *testing.T, name string, args ...interface{}) []interface{} {
	t.Helper()
	entry, ok := PrecompileByName(name)
	if !ok {
		t.Fatalf("missing precompile %s", name)
	}
	input := mustPackABI(t, entry.InputTypes(), args...)
	output, err := entry.Run(input)
	if err != nil {
		t.Fatalf("%s failed: %v", name, err)
	}
	return mustUnpackABI(t, entry.OutputTypes(), output)
}

func mustMakeABIArguments(t *testing.T, types []string) abi.Arguments {
	t.Helper()
	args := make(abi.Arguments, 0, len(types))
	for _, typ := range types {
		abiType, err := abi.NewType(typ, "", nil)
		if err != nil {
			t.Fatalf("invalid ABI type %s: %v", typ, err)
		}
		args = append(args, abi.Argument{Type: abiType})
	}
	return args
}

func mustPackABI(t *testing.T, types []string, values ...interface{}) []byte {
	t.Helper()
	args := mustMakeABIArguments(t, types)
	out, err := args.Pack(values...)
	if err != nil {
		t.Fatalf("ABI pack failed: %v", err)
	}
	return out
}

func mustUnpackABI(t *testing.T, types []string, data []byte) []interface{} {
	t.Helper()
	args := mustMakeABIArguments(t, types)
	out, err := args.Unpack(data)
	if err != nil {
		t.Fatalf("ABI unpack failed: %v", err)
	}
	return out
}

func deterministicSchnorrProof(secret, message []byte, nonce int64) []byte {
	p, q := precompileRFC3526Subgroup()
	x := precompileRFC3526SubgroupScalar(secret, q)
	r := big.NewInt(nonce)
	r.Mod(r, q)
	if r.Sign() == 0 {
		r.SetInt64(1)
	}
	g := big.NewInt(4)
	y := new(big.Int).Exp(g, x, p)
	t := new(big.Int).Exp(g, r, p)
	c := schnorrChallenge(q, y, t, message)
	s := new(big.Int).Mul(c, x)
	s.Add(s, r)
	s.Mod(s, q)

	size := precompileRFC3526FieldBytes()
	out := make([]byte, 0, 3*size)
	out = append(out, precompileFixedBytes(y, size)...)
	out = append(out, precompileFixedBytes(t, size)...)
	out = append(out, precompileFixedBytes(s, size)...)
	return out
}
