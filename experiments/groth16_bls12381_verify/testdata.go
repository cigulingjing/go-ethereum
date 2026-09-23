package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	curve "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

const (
	g1Size = curve.SizeOfG1AffineUncompressed
	g2Size = curve.SizeOfG2AffineUncompressed
	frSize = fr.Bytes
)

func loadArchived(dir string) (*Proof, *VerifyingKey, []fr.Element, error) {
	vkRaw, err := os.ReadFile(filepath.Join(dir, "vk.bin"))
	if err != nil {
		return nil, nil, nil, err
	}
	proofRaw, err := os.ReadFile(filepath.Join(dir, "proof.bin"))
	if err != nil {
		return nil, nil, nil, err
	}
	pubRaw, err := os.ReadFile(filepath.Join(dir, "public.bin"))
	if err != nil {
		return nil, nil, nil, err
	}
	proof, err := decodeProof(proofRaw)
	if err != nil {
		return nil, nil, nil, err
	}
	vk, err := decodeVK(vkRaw)
	if err != nil {
		return nil, nil, nil, err
	}
	pub, err := decodePublic(pubRaw)
	if err != nil {
		return nil, nil, nil, err
	}
	return proof, vk, pub, nil
}

func decodeProof(raw []byte) (*Proof, error) {
	if len(raw) != g1Size+g2Size+g1Size {
		return nil, fmt.Errorf("proof size %d", len(raw))
	}
	var p Proof
	if _, err := p.Ar.SetBytes(raw[:g1Size]); err != nil {
		return nil, err
	}
	if _, err := p.Bs.SetBytes(raw[g1Size : g1Size+g2Size]); err != nil {
		return nil, err
	}
	if _, err := p.Krs.SetBytes(raw[g1Size+g2Size:]); err != nil {
		return nil, err
	}
	return &p, nil
}

func decodeVK(raw []byte) (*VerifyingKey, error) {
	need := g1Size + 3*g2Size + 4
	if len(raw) < need {
		return nil, fmt.Errorf("vk too short: %d", len(raw))
	}
	var vk VerifyingKey
	off := 0
	if _, err := vk.Alpha.SetBytes(raw[off : off+g1Size]); err != nil {
		return nil, err
	}
	off += g1Size
	if _, err := vk.Beta.SetBytes(raw[off : off+g2Size]); err != nil {
		return nil, err
	}
	off += g2Size
	if _, err := vk.Gamma.SetBytes(raw[off : off+g2Size]); err != nil {
		return nil, err
	}
	off += g2Size
	if _, err := vk.Delta.SetBytes(raw[off : off+g2Size]); err != nil {
		return nil, err
	}
	off += g2Size
	nk := binary.BigEndian.Uint32(raw[off : off+4])
	off += 4
	if nk < 1 {
		return nil, fmt.Errorf("vk has no K points")
	}
	if len(raw) != need+int(nk)*g1Size {
		return nil, fmt.Errorf("vk size %d nk=%d", len(raw), nk)
	}
	vk.K = make([]curve.G1Affine, nk)
	for i := range vk.K {
		if _, err := vk.K[i].SetBytes(raw[off : off+g1Size]); err != nil {
			return nil, err
		}
		off += g1Size
	}
	return &vk, nil
}

func decodePublic(raw []byte) ([]fr.Element, error) {
	if len(raw)%frSize != 0 {
		return nil, fmt.Errorf("public size %d", len(raw))
	}
	n := len(raw) / frSize
	out := make([]fr.Element, n)
	for i := 0; i < n; i++ {
		out[i].SetBytes(raw[i*frSize : (i+1)*frSize])
	}
	return out, nil
}
