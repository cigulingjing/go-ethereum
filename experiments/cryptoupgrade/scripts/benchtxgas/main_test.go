package main

import (
	"bytes"
	"math/big"
	"testing"
)

func TestReferenceInputs(t *testing.T) {
	fixtures, err := buildFixtures(config{referenceInputs: true, a: "100", b: "100", polyLeftLen: 4, polyRightLen: 4, polyModulus: "12289"})
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 8 {
		t.Fatalf("expected eight fixtures, got %d", len(fixtures))
	}
	if fixtures[0].Values[0].(*big.Int).BitLen() != 63 {
		t.Fatalf("unexpected Add operand: %v", fixtures[0].Values[0])
	}
	if !bytes.Equal(fixtures[1].Values[0].([]byte), bytes.Repeat([]byte{0x73}, 256)) || len(fixtures[2].Values[0].([]byte)) != 32 {
		t.Fatal("hash inputs differ from calibrated levels")
	}
	if fixtures[3].Values[2].(*big.Int).Uint64() != 500 || fixtures[4].Values[0].([]byte)[0] != 0x80 || len(fixtures[4].Values[1].([]byte)) != 256 {
		t.Fatal("PBKDF2 or DH reference level mismatch")
	}
	if len(fixtures[6].Values[0].([]byte)) != 256 || len(fixtures[6].Values[1].([]byte)) != 768 {
		t.Fatal("Schnorr reference level mismatch")
	}
	if len(fixtures[7].Values[0].([]*big.Int)) != 4 {
		t.Fatal("Polynomial reference level mismatch")
	}
}
