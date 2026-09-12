package main

import (
	"encoding/binary"
	"math/big"
)

const (
	shamirMagic      = "SS01"
	shamirFieldBytes = 66
	shamirHeaderSize = 12
)

// ShamirRecover decodes shares produced by ShamirSplit and recovers the secret.
func ShamirRecover(encodedShares []byte) []byte {
	if len(encodedShares) < shamirHeaderSize || string(encodedShares[:4]) != shamirMagic {
		return nil
	}
	secretLen := int(binary.BigEndian.Uint16(encodedShares[4:6]))
	threshold := int(binary.BigEndian.Uint16(encodedShares[6:8]))
	count := int(binary.BigEndian.Uint16(encodedShares[8:10]))
	fieldLen := int(binary.BigEndian.Uint16(encodedShares[10:12]))
	if threshold < 2 || count < threshold || fieldLen != shamirFieldBytes || secretLen > fieldLen {
		return nil
	}
	shareSize := 2 + fieldLen
	if len(encodedShares) != shamirHeaderSize+count*shareSize {
		return nil
	}

	p := shamirPrime()
	xs := make([]*big.Int, threshold)
	ys := make([]*big.Int, threshold)
	offset := shamirHeaderSize
	seen := make(map[uint16]bool, threshold)
	for i := 0; i < threshold; i++ {
		x := binary.BigEndian.Uint16(encodedShares[offset : offset+2])
		offset += 2
		if x == 0 || seen[x] {
			return nil
		}
		seen[x] = true
		xs[i] = new(big.Int).SetUint64(uint64(x))
		ys[i] = new(big.Int).SetBytes(encodedShares[offset : offset+fieldLen])
		if ys[i].Cmp(p) >= 0 {
			return nil
		}
		offset += fieldLen
	}

	secret := shamirInterpolateZero(xs, ys, p)
	if secret == nil {
		return nil
	}
	return shamirFixedBytes(secret, secretLen)
}

func shamirPrime() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 521), big.NewInt(1))
}

func shamirInterpolateZero(xs, ys []*big.Int, p *big.Int) *big.Int {
	secret := new(big.Int)
	for j := range xs {
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)
		for m := range xs {
			if m == j {
				continue
			}
			negX := new(big.Int).Neg(xs[m])
			numerator.Mul(numerator, negX)
			numerator.Mod(numerator, p)

			diff := new(big.Int).Sub(xs[j], xs[m])
			denominator.Mul(denominator, diff)
			denominator.Mod(denominator, p)
		}
		inverse := new(big.Int).ModInverse(denominator, p)
		if inverse == nil {
			return nil
		}
		term := new(big.Int).Mul(ys[j], numerator)
		term.Mod(term, p)
		term.Mul(term, inverse)
		term.Mod(term, p)
		secret.Add(secret, term)
		secret.Mod(secret, p)
	}
	return secret
}

func shamirFixedBytes(x *big.Int, size int) []byte {
	if size == 0 {
		return []byte{}
	}
	b := x.Bytes()
	if len(b) > size {
		return nil
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}
