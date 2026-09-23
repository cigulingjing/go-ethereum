package main

import (
	"errors"

	"github.com/consensys/gnark-crypto/ecc"
	curve "github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
)

// 从 experiments/gnark/backend/groth16/bn254/verify.go 抽出的无 commitment 验证核。
//
//	e(Ar, Bs) · e(Krs, -δ) · e(L_pub, -γ) = e(α, β)
//
// L_pub = K[0] + Σ public[i]·K[i+1]

type Proof struct {
	Ar, Krs curve.G1Affine
	Bs      curve.G2Affine
}

type VerifyingKey struct {
	Alpha curve.G1Affine
	Beta  curve.G2Affine
	Gamma curve.G2Affine
	Delta curve.G2Affine
	K     []curve.G1Affine
}

func (p *Proof) isValid() bool {
	return p.Ar.IsInSubGroup() && p.Krs.IsInSubGroup() && p.Bs.IsInSubGroup()
}

func Verify(proof *Proof, vk *VerifyingKey, public []fr.Element) error {
	if proof == nil || vk == nil {
		return errors.New("nil groth16 argument")
	}
	if !proof.isValid() {
		return errors.New("points in the proof are not in the correct subgroup")
	}
	if len(vk.K) < 1 || len(public) != len(vk.K)-1 {
		return errors.New("invalid public witness size")
	}

	var deltaNeg, gammaNeg curve.G2Affine
	deltaNeg.Neg(&vk.Delta)
	gammaNeg.Neg(&vk.Gamma)

	doubleML, err := curve.MillerLoop(
		[]curve.G1Affine{proof.Krs, proof.Ar},
		[]curve.G2Affine{deltaNeg, proof.Bs},
	)
	if err != nil {
		return err
	}

	var kSum curve.G1Jac
	if len(public) > 0 {
		if _, err := kSum.MultiExp(vk.K[1:], public, ecc.MultiExpConfig{}); err != nil {
			return err
		}
	}
	kSum.AddMixed(&vk.K[0])
	var lpub curve.G1Affine
	lpub.FromJacobian(&kSum)

	right, err := curve.MillerLoop([]curve.G1Affine{lpub}, []curve.G2Affine{gammaNeg})
	if err != nil {
		return err
	}
	right = curve.FinalExponentiation(&right, &doubleML)

	left, err := curve.Pair([]curve.G1Affine{vk.Alpha}, []curve.G2Affine{vk.Beta})
	if err != nil {
		return err
	}
	if !left.Equal(&right) {
		return errors.New("pairing doesn't match")
	}
	return nil
}
