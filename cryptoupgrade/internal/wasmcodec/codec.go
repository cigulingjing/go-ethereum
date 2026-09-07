// Package wasmcodec implements the transport encoding used for WASM upgrade
// artifacts uploaded through CodeStorage.
package wasmcodec

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const DefaultMaxDecodedSize int64 = 16 << 20

var wasmMagic = []byte{0x00, 0x61, 0x73, 0x6d}

// Encode compresses WASM bytecode with gzip and encodes it as standard base64.
func Encode(wasm []byte) (string, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(wasm); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("compress wasm bytecode: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finish wasm compression: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

// EncodeFile reads and encodes a WASM file.
func EncodeFile(path string) (string, error) {
	wasm, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read wasm file %s: %w", path, err)
	}
	return Encode(wasm)
}

// Decode decodes a gzip/base64 WASM artifact using the package default size limit.
func Decode(encoded string) ([]byte, error) {
	return DecodeLimit(encoded, DefaultMaxDecodedSize)
}

// DecodeLimit decodes a WASM artifact and rejects output larger than maxSize.
func DecodeLimit(encoded string, maxSize int64) ([]byte, error) {
	if maxSize <= 0 {
		maxSize = DefaultMaxDecodedSize
	}
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode wasm base64: %w", err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open wasm gzip stream: %w", err)
	}
	defer reader.Close()

	wasm, err := io.ReadAll(io.LimitReader(reader, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("decompress wasm bytecode: %w", err)
	}
	if int64(len(wasm)) > maxSize {
		return nil, fmt.Errorf("decoded wasm exceeds %d bytes", maxSize)
	}
	if err := validateWASM(wasm); err != nil {
		return nil, err
	}
	return wasm, nil
}

// DecodeToFile decodes a WASM artifact and writes it to path.
func DecodeToFile(encoded, path string) error {
	wasm, err := Decode(encoded)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create wasm directory %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, wasm, 0o644); err != nil {
		return fmt.Errorf("write wasm file %s: %w", path, err)
	}
	return nil
}

func validateWASM(wasm []byte) error {
	if len(wasm) < len(wasmMagic) {
		return fmt.Errorf("decoded wasm is shorter than magic header")
	}
	if !bytes.Equal(wasm[:len(wasmMagic)], wasmMagic) {
		return fmt.Errorf("decoded wasm does not start with \\0asm magic")
	}
	return nil
}
