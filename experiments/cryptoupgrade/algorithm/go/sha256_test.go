package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"testing"
)

func TestSha256KnownVectors(t *testing.T) {
	for _, test := range []struct {
		name string
		data []byte
		want string
	}{
		{"empty", nil, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"abc", []byte("abc"), "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"multiple-blocks", bytes.Repeat([]byte("a"), 1000000), "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := hex.EncodeToString(Sha256(test.data)); got != test.want {
				t.Fatalf("digest = %s, want %s", got, test.want)
			}
		})
	}
}

func TestSha256MatchesStandardLibrary(t *testing.T) {
	for _, length := range []int{128, 192, 193, 200, 224, 255, 256, 511, 512, 1024} {
		input := bytes.Repeat([]byte{0x61}, length)
		want := sha256.Sum256(input)
		if got := Sha256(input); !bytes.Equal(got, want[:]) {
			t.Fatalf("repeated length=%d: digest=%x, want=%x", length, got, want)
		}
	}
	random := rand.New(rand.NewSource(42))
	for _, length := range []int{0, 1, 3, 55, 56, 57, 63, 64, 65, 119, 120, 127, 128, 129, 255, 256, 1024, 4096} {
		input := make([]byte, length)
		if _, err := random.Read(input); err != nil {
			t.Fatal(err)
		}
		original := append([]byte(nil), input...)
		want := sha256.Sum256(input)
		if got := Sha256(input); !bytes.Equal(got, want[:]) {
			t.Fatalf("length=%d: digest=%x, want=%x", length, got, want)
		}
		if !bytes.Equal(input, original) {
			t.Fatalf("length=%d: input modified", length)
		}
	}
}
