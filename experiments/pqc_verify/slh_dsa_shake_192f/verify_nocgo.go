//go:build !cgo

package slhdsa

import "errors"

const Name = "slh_dsa_shake_192f_verify"

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.New("slh_dsa_shake_192f_verify requires CGO")
}
