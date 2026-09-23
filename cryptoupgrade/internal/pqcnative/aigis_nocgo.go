//go:build !cgo

package pqcnative

import "errors"

const (
	PublicKeyBytes = 1312
	SignatureBytes = 2445
)

func Enabled() bool {
	return false
}

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.New("AigisSig2Verify requires CGO")
}
