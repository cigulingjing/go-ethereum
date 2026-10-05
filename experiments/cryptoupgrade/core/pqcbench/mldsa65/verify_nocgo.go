//go:build !cgo

package mldsa65

import (
	"errors"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
)

const Name = "ml_dsa_65_verify"

func Enabled() bool { return false }

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.Join(pqcbench.ErrCGODisabled, errors.New("ml_dsa_65_verify requires CGO"))
}
