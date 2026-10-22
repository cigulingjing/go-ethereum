package main

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"testing"
)

func TestSchnorrChallengeMatchesStandardLibrary(t *testing.T) {
	prime, order := schnorrGroup()
	public := new(big.Int).Exp(big.NewInt(4), big.NewInt(17), prime)
	commitment := new(big.Int).Exp(big.NewInt(4), big.NewInt(23), prime)
	for _, length := range []int{0, 1, 34, 35, 55, 56, 63, 64, 65, 128, 512, 4096} {
		message := bytes.Repeat([]byte{byte(length + 1)}, length)
		input := append([]byte("cryptoupgrade-schnorr"), schnorrFixedBytes(public)...)
		input = append(input, schnorrFixedBytes(commitment)...)
		input = append(input, message...)
		sum := sha256.Sum256(input)
		want := new(big.Int).Mod(new(big.Int).SetBytes(sum[:]), order)
		if got := schnorrChallenge(order, public, commitment, message); got.Cmp(want) != 0 {
			t.Fatalf("length %d: challenge %x, want %x", length, got, want)
		}
	}
}

func TestSchnorrVerifyStandardLibraryProof(t *testing.T) {
	prime, order := schnorrGroup()
	secret := big.NewInt(17)
	public := new(big.Int).Exp(big.NewInt(4), secret, prime)
	for _, length := range []int{0, 1, 56, 64, 65, 1024} {
		message := bytes.Repeat([]byte{0xa7}, length)
		nonce := big.NewInt(23)
		commitment := new(big.Int).Exp(big.NewInt(4), nonce, prime)
		input := append([]byte("cryptoupgrade-schnorr"), schnorrFixedBytes(public)...)
		input = append(input, schnorrFixedBytes(commitment)...)
		input = append(input, message...)
		sum := sha256.Sum256(input)
		challenge := new(big.Int).SetBytes(sum[:])
		response := new(big.Int).Mod(new(big.Int).Add(nonce, new(big.Int).Mul(challenge, secret)), order)
		proof := make([]byte, 0, 3*schnorrFieldBytes())
		proof = append(proof, schnorrFixedBytes(public)...)
		proof = append(proof, schnorrFixedBytes(commitment)...)
		proof = append(proof, schnorrFixedBytes(response)...)
		if !SchnorrVerify(message, proof) {
			t.Fatalf("length %d: valid proof rejected", length)
		}
		if SchnorrVerify(append(append([]byte(nil), message...), 1), proof) {
			t.Fatalf("length %d: modified message accepted", length)
		}
		invalid := append([]byte(nil), proof...)
		invalid[len(invalid)-1] ^= 1
		if SchnorrVerify(message, invalid) {
			t.Fatalf("length %d: modified response accepted", length)
		}
		if SchnorrVerify(message, proof[:len(proof)-1]) {
			t.Fatal("invalid proof length accepted")
		}
		copy(invalid[:schnorrFieldBytes()], make([]byte, schnorrFieldBytes()))
		if SchnorrVerify(message, invalid) {
			t.Fatal("invalid group element accepted")
		}
	}
}
