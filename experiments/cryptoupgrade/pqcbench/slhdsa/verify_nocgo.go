//go:build !cgo

package slhdsa

import (
	"errors"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench"
)

const Name = "slh_dsa_shake_192f_verify"

func Enabled() bool { return false }

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.Join(pqcbench.ErrCGODisabled, errors.New("slh_dsa_shake_192f_verify requires CGO"))
}
