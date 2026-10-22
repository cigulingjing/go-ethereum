package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"math/big"
	"math/rand"
	"testing"
)

func TestExperimentSha512MatchesStandardLibrary(t *testing.T) {
	for _, length := range []int{0, 1, 111, 112, 127, 128, 129, 255, 1024} {
		data := bytes.Repeat([]byte("x"), length)
		want := sha512.Sum512(data)
		if got := ed25519Sha512(data); !bytes.Equal(got, want[:]) {
			t.Fatalf("SHA512 length %d: %x != %x", length, got, want)
		}
	}
}

func TestExperimentEd25519MatchesStandardLibrary(t *testing.T) {
	public, err := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	if err != nil {
		t.Fatal(err)
	}
	signature, err := hex.DecodeString("e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b")
	if err != nil {
		t.Fatal(err)
	}
	if !Ed25519Verify(public, nil, signature) {
		t.Fatal("RFC 8032 empty-message signature rejected")
	}
	invalidPublic := bytes.Repeat([]byte{0xff}, 32)
	if Ed25519Verify(invalidPublic, nil, signature) {
		t.Fatal("non-canonical public key accepted")
	}
	invalidSignature := append([]byte(nil), signature...)
	copy(invalidSignature[:32], invalidPublic)
	if Ed25519Verify(public, nil, invalidSignature) {
		t.Fatal("non-canonical R accepted")
	}
	for _, length := range []int{0, 1, 56, 112, 128, 1024} {
		seed := bytes.Repeat([]byte{byte(length)}, ed25519.SeedSize)
		private := ed25519.NewKeyFromSeed(seed)
		public := private.Public().(ed25519.PublicKey)
		message := bytes.Repeat([]byte{0xb4}, length)
		signature := ed25519.Sign(private, message)
		if !Ed25519Verify(public, message, signature) {
			t.Fatalf("valid signature length %d rejected", length)
		}
		message = append(message, 1)
		if Ed25519Verify(public, message, signature) {
			t.Fatalf("modified message length %d accepted", length)
		}
		signature[63] = 0xff
		if Ed25519Verify(public, message[:length], signature) {
			t.Fatalf("invalid scalar length %d accepted", length)
		}
	}
	if Ed25519Verify(make([]byte, 31), nil, make([]byte, 64)) || Ed25519Verify(make([]byte, 32), nil, make([]byte, 63)) {
		t.Fatal("invalid sizes accepted")
	}
	random := rand.New(rand.NewSource(42))
	for index := 0; index < 10; index++ {
		seed := make([]byte, ed25519.SeedSize)
		if _, err := random.Read(seed); err != nil {
			t.Fatal(err)
		}
		private := ed25519.NewKeyFromSeed(seed)
		public := private.Public().(ed25519.PublicKey)
		message := bytes.Repeat([]byte{byte(index)}, index*37)
		if !Ed25519Verify(public, message, ed25519.Sign(private, message)) {
			t.Fatalf("random signature %d rejected", index)
		}
		badSignature := ed25519.Sign(private, message)
		badSignature[0] ^= 1
		if got, want := Ed25519Verify(public, message, badSignature), ed25519.Verify(public, message, badSignature); got != want {
			t.Fatalf("mutated signature %d: %t != %t", index, got, want)
		}
	}
}

func TestExperimentPbkdf2MatchesStandardLibrary(t *testing.T) {
	for _, test := range []struct {
		password, salt        []byte
		iterations, keyLength int64
	}{
		{nil, nil, 1, 0},
		{[]byte("password"), []byte("salt"), 1, 32},
		{[]byte("password"), []byte("salt"), 2, 64},
		{bytes.Repeat([]byte{0xab}, 100), bytes.Repeat([]byte{0xcd}, 130), 17, 65},
	} {
		got := Pbkdf2Sha256(test.password, test.salt, big.NewInt(test.iterations), big.NewInt(test.keyLength))
		want := make([]byte, test.keyLength)
		for block := int64(1); block <= (test.keyLength+31)/32; block++ {
			input := append(append([]byte(nil), test.salt...), byte(block>>24), byte(block>>16), byte(block>>8), byte(block))
			mac := hmac.New(sha256.New, test.password)
			mac.Write(input)
			value := mac.Sum(nil)
			result := append([]byte(nil), value...)
			for count := int64(1); count < test.iterations; count++ {
				mac = hmac.New(sha256.New, test.password)
				mac.Write(value)
				value = mac.Sum(nil)
				for index := range result {
					result[index] ^= value[index]
				}
			}
			copy(want[(block-1)*32:], result)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("PBKDF2 mismatch for %+v: %x != %x", test, got, want)
		}
	}
	if Pbkdf2Sha256(nil, nil, nil, big.NewInt(32)) != nil || Pbkdf2Sha256(nil, nil, big.NewInt(1), big.NewInt(-1)) != nil || Pbkdf2Sha256(nil, nil, big.NewInt(1000001), big.NewInt(1)) != nil {
		t.Fatal("out-of-range PBKDF2 parameters accepted")
	}
}

func TestExperimentPedersenMatchesStandardLibrary(t *testing.T) {
	prime, order := pedersenGroup()
	for _, count := range []int{0, 1, 31, 32, 256, 513} {
		message := bytes.Repeat([]byte{byte(count)}, count)
		blinding := bytes.Repeat([]byte{byte(count + 1)}, count+1)
		var generator *big.Int
		for counter := byte(0); ; counter++ {
			input := append([]byte("cryptoupgrade-pedersen-h-"), counter)
			digest := sha256.Sum256(input)
			generator = new(big.Int).SetBytes(digest[:])
			generator.Exp(generator, big.NewInt(2), prime)
			if generator.Cmp(big.NewInt(1)) > 0 {
				break
			}
		}
		left := new(big.Int).Exp(big.NewInt(4), new(big.Int).Mod(new(big.Int).SetBytes(message), order), prime)
		right := new(big.Int).Exp(generator, new(big.Int).Mod(new(big.Int).SetBytes(blinding), order), prime)
		want := pedersenFixedBytes(new(big.Int).Mod(new(big.Int).Mul(left, right), prime))
		if got := PedersenCommit(message, blinding); !bytes.Equal(got, want) {
			t.Fatalf("Pedersen length %d: %x != %x", count, got, want)
		}
	}
}
