package sourcecodec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	source := []byte("package algorithm\n\nfunc Add(a, b int) int { return a + b }\n")
	encoded, err := Encode(source)
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if string(decoded) != string(source) {
		t.Fatalf("decoded source mismatch:\nwant %q\ngot  %q", source, decoded)
	}
}

func TestDecodeRejectsMalformedInput(t *testing.T) {
	for _, encoded := range []string{"not-base64", "bm90LWd6aXA="} {
		if _, err := Decode(encoded); err == nil {
			t.Fatalf("Decode(%q) succeeded, want error", encoded)
		}
	}
}

func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.go")
	outputPath := filepath.Join(dir, "output.go")
	source := []byte("package algorithm\n")
	if err := os.WriteFile(inputPath, source, 0644); err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeFile(inputPath)
	if err != nil {
		t.Fatalf("EncodeFile returned error: %v", err)
	}
	if err := DecodeToFile(encoded, outputPath); err != nil {
		t.Fatalf("DecodeToFile returned error: %v", err)
	}
	decoded, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(source) {
		t.Fatalf("decoded file mismatch: want %q, got %q", source, decoded)
	}
}

func TestDecodeLimit(t *testing.T) {
	encoded, err := Encode([]byte(strings.Repeat("x", 33)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeLimit(encoded, 32); err == nil {
		t.Fatal("DecodeLimit succeeded for oversized source")
	}
	decoded, err := DecodeLimit(encoded, 33)
	if err != nil {
		t.Fatalf("DecodeLimit returned error at boundary: %v", err)
	}
	if len(decoded) != 33 {
		t.Fatalf("decoded length %d, want 33", len(decoded))
	}
}
