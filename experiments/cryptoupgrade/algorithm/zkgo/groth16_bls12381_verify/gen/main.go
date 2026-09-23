package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/consensys/gnark-crypto/ecc"
	curve "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/consensys/gnark/backend/groth16"
	groth16bls12381 "github.com/consensys/gnark/backend/groth16/bls12-381"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// 小型公开关系 y = x²，只为产出可复现的 Groth16 BLS12-381 验证向量。
type squareCircuit struct {
	X frontend.Variable
	Y frontend.Variable `gnark:",public"`
}

func (c *squareCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.X, c.X), c.Y)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen groth16 testdata: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	outDir := "."
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	ccs, err := frontend.Compile(ecc.BLS12_381.ScalarField(), r1cs.NewBuilder, &squareCircuit{})
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		return fmt.Errorf("setup: %w", err)
	}
	assignment := &squareCircuit{X: 3, Y: 9}
	witness, err := frontend.NewWitness(assignment, ecc.BLS12_381.ScalarField())
	if err != nil {
		return fmt.Errorf("witness: %w", err)
	}
	proof, err := groth16.Prove(ccs, pk, witness)
	if err != nil {
		return fmt.Errorf("prove: %w", err)
	}
	pubWitness, err := witness.Public()
	if err != nil {
		return fmt.Errorf("public witness: %w", err)
	}
	if err := groth16.Verify(proof, vk, pubWitness); err != nil {
		return fmt.Errorf("gnark verify: %w", err)
	}

	typedVK, ok := vk.(*groth16bls12381.VerifyingKey)
	if !ok {
		return fmt.Errorf("unexpected vk type %T", vk)
	}
	typedProof, ok := proof.(*groth16bls12381.Proof)
	if !ok {
		return fmt.Errorf("unexpected proof type %T", proof)
	}
	vec, ok := pubWitness.Vector().(fr.Vector)
	if !ok {
		return fmt.Errorf("unexpected public vector %T", pubWitness.Vector())
	}

	vkRaw, err := encodeVK(typedVK)
	if err != nil {
		return err
	}
	proofRaw := encodeProof(typedProof)
	pubRaw := encodePublic(vec)

	if err := os.WriteFile(filepath.Join(outDir, "vk.bin"), vkRaw, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "proof.bin"), proofRaw, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "public.bin"), pubRaw, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s vk=%d proof=%d public=%d\n", outDir, len(vkRaw), len(proofRaw), len(pubRaw))
	return nil
}

func encodeProof(proof *groth16bls12381.Proof) []byte {
	ar := proof.Ar.RawBytes()
	bs := proof.Bs.RawBytes()
	krs := proof.Krs.RawBytes()
	out := make([]byte, 0, len(ar)+len(bs)+len(krs))
	out = append(out, ar[:]...)
	out = append(out, bs[:]...)
	out = append(out, krs[:]...)
	return out
}

func encodeVK(vk *groth16bls12381.VerifyingKey) ([]byte, error) {
	if len(vk.G1.K) < 1 {
		return nil, fmt.Errorf("vk has no K points")
	}
	alpha := vk.G1.Alpha.RawBytes()
	beta := vk.G2.Beta.RawBytes()
	gamma := vk.G2.Gamma.RawBytes()
	delta := vk.G2.Delta.RawBytes()
	out := make([]byte, 0, len(alpha)+len(beta)+len(gamma)+len(delta)+4+len(vk.G1.K)*curve.SizeOfG1AffineUncompressed)
	out = append(out, alpha[:]...)
	out = append(out, beta[:]...)
	out = append(out, gamma[:]...)
	out = append(out, delta[:]...)
	var nk [4]byte
	binary.BigEndian.PutUint32(nk[:], uint32(len(vk.G1.K)))
	out = append(out, nk[:]...)
	for i := range vk.G1.K {
		raw := vk.G1.K[i].RawBytes()
		out = append(out, raw[:]...)
	}
	return out, nil
}

func encodePublic(vec fr.Vector) []byte {
	out := make([]byte, 0, len(vec)*fr.Bytes)
	for i := range vec {
		b := vec[i].Bytes()
		out = append(out, b[:]...)
	}
	return out
}
