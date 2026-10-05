//go:build !cgo

package groth16bn254

import (
	"errors"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
)

const Name = "groth16_bn254_verify"

func Enabled() bool { return false }

func Verify(vk, public, proof []byte) (bool, error) {
	return false, errors.Join(pqcbench.ErrCGODisabled, errors.New("groth16_bn254_verify requires CGO"))
}
