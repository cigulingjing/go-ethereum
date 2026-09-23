//go:build !cgo

package mldsa65

import "errors"

const Name = "ml_dsa_65_verify"

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.New("ml_dsa_65_verify requires CGO")
}
