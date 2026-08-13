package callcodec

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func codeStorageABI(t *testing.T) abi.ABI {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		t.Fatalf("parse CodeStorage ABI: %v", err)
	}
	return parsed
}

func TestParseCall(t *testing.T) {
	contractABI := codeStorageABI(t)
	payload := []byte{1, 2, 3}
	data, err := contractABI.Pack("callFunc", "Add", payload)
	if err != nil {
		t.Fatal(err)
	}
	name, input, err := ParseCall(contractABI, data)
	if err != nil {
		t.Fatalf("ParseCall returned error: %v", err)
	}
	if name != "Add" || string(input) != string(payload) {
		t.Fatalf("ParseCall returned (%q, %x), want (%q, %x)", name, input, "Add", payload)
	}
}

func TestParseCallRejectsInvalidInput(t *testing.T) {
	contractABI := codeStorageABI(t)
	if _, _, err := ParseCall(contractABI, []byte{1, 2, 3}); err == nil {
		t.Fatal("ParseCall accepted short input")
	}
	data, err := contractABI.Pack("getGas", "Add")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseCall(contractABI, data); err == nil {
		t.Fatal("ParseCall accepted a different selector")
	}
}

func TestInputOutputCompatibility(t *testing.T) {
	values := []interface{}{big.NewInt(7), []byte("payload")}
	types := []string{"uint256", "bytes"}
	encoded, err := PackOutput(values, types)
	if err != nil {
		t.Fatalf("PackOutput returned error: %v", err)
	}
	decoded, err := UnpackInput(encoded, types)
	if err != nil {
		t.Fatalf("UnpackInput returned error: %v", err)
	}
	if decoded[0].(*big.Int).Cmp(values[0].(*big.Int)) != 0 {
		t.Fatalf("decoded integer %v, want %v", decoded[0], values[0])
	}
	if string(decoded[1].([]byte)) != "payload" {
		t.Fatalf("decoded bytes %q, want payload", decoded[1])
	}
}

func TestInvalidABIType(t *testing.T) {
	if _, err := UnpackInput(nil, []string{"not-a-type"}); err == nil {
		t.Fatal("UnpackInput accepted invalid ABI type")
	}
	if _, err := PackOutput(nil, []string{"not-a-type"}); err == nil {
		t.Fatal("PackOutput accepted invalid ABI type")
	}
}
