//go:build !cgo

package groth16bls12381

import (
	"errors"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench"
)

const Name = "groth16_bls12381_verify"

func Enabled() bool { return false }

func Verify(vk, public, proof []byte) (bool, error) {
	return false, errors.Join(pqcbench.ErrCGODisabled, errors.New("groth16_bls12381_verify requires CGO"))
}
