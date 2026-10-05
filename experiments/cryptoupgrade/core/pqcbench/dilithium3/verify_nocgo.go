//go:build !cgo

package dilithium3

import (
	"errors"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
)

const Name = "dilithium3_verify"

func Enabled() bool { return false }

func Verify(pk, message, signature []byte) (bool, error) {
	return false, errors.Join(pqcbench.ErrCGODisabled, errors.New("dilithium3_verify requires CGO"))
}
