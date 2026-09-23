//go:build !cgo

package aigis

import (
	"errors"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench"
)

const Name = "aigis_sig2_verify"

func Enabled() bool { return false }

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.Join(pqcbench.ErrCGODisabled, errors.New("aigis_sig2_verify requires CGO"))
}
