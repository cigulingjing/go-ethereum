//go:build !cgo

package dilithium3

import "errors"

const Name = "dilithium3_verify"

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.New("dilithium3_verify requires CGO")
}
