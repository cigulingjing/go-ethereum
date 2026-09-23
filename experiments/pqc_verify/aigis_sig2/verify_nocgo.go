//go:build !cgo

package aigis

import "errors"

const Name = "aigis_sig2_verify"

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.New("aigis_sig2_verify requires CGO")
}
