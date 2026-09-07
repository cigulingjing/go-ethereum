package wasmcodec

import (
	"bytes"
	"testing"
)

var emptyWASM = []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	encoded, err := Encode(emptyWASM)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !bytes.Equal(decoded, emptyWASM) {
		t.Fatalf("decoded bytes mismatch: got %x want %x", decoded, emptyWASM)
	}
}

func TestDecodeRejectsNonWASM(t *testing.T) {
	encoded, err := Encode([]byte("package main"))
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if _, err := Decode(encoded); err == nil {
		t.Fatal("Decode accepted non-WASM payload")
	}
}

func TestDecodeRejectsCorruptedEncoding(t *testing.T) {
	if _, err := Decode("not-base64"); err == nil {
		t.Fatal("Decode accepted corrupted base64")
	}
}

func TestDecodeLimit(t *testing.T) {
	encoded, err := Encode(emptyWASM)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if _, err := DecodeLimit(encoded, 7); err == nil {
		t.Fatal("DecodeLimit accepted oversized wasm")
	}
	decoded, err := DecodeLimit(encoded, 8)
	if err != nil {
		t.Fatalf("DecodeLimit failed: %v", err)
	}
	if !bytes.Equal(decoded, emptyWASM) {
		t.Fatalf("decoded bytes mismatch: got %x want %x", decoded, emptyWASM)
	}
}
