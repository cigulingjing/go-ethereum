// Package sourcecodec implements the transport encoding used for Go algorithm
// source artifacts uploaded through CodeStorage.
package sourcecodec

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Encode compresses source with gzip and encodes it as standard base64.
func Encode(source []byte) (string, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(source); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("compress source: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finish source compression: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

// EncodeFile reads and encodes a Go source file.
func EncodeFile(path string) (string, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read source file %s: %w", path, err)
	}
	return Encode(source)
}

// Decode decodes a gzip/base64 source artifact without imposing a new size
// limit, preserving the existing CodeStorage activation semantics.
func Decode(encoded string) ([]byte, error) {
	return DecodeLimit(encoded, 0)
}

// DecodeLimit decodes a source artifact and rejects output larger than maxSize.
// A non-positive maxSize disables the limit for compatibility with existing uploads.
func DecodeLimit(encoded string, maxSize int64) ([]byte, error) {
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode source base64: %w", err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open source gzip stream: %w", err)
	}
	defer reader.Close()

	var source []byte
	if maxSize > 0 {
		source, err = io.ReadAll(io.LimitReader(reader, maxSize+1))
		if err == nil && int64(len(source)) > maxSize {
			return nil, fmt.Errorf("decoded source exceeds %d bytes", maxSize)
		}
	} else {
		source, err = io.ReadAll(reader)
	}
	if err != nil {
		return nil, fmt.Errorf("decompress source: %w", err)
	}
	return source, nil
}

// DecodeToFile decodes a source artifact and writes it to path.
func DecodeToFile(encoded, path string) error {
	source, err := Decode(encoded)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create source directory %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, source, 0644); err != nil {
		return fmt.Errorf("write source file %s: %w", path, err)
	}
	return nil
}
