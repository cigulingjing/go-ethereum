package main

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
)

const (
	shamirMagic      = "SS01"
	shamirFieldBytes = 66
	shamirHeaderSize = 12
)

// ShamirSplit encodes Shamir shares as SS01 || secretLen || threshold || count || fieldLen || shares.
func ShamirSplit(secret []byte, threshold, total *big.Int) []byte {
	t, ok := shamirSmallInt(threshold, 2, 255)
	if !ok {
		return nil
	}
	n, ok := shamirSmallInt(total, t, 255)
	if !ok || len(secret) > shamirFieldBytes || len(secret) > 0xffff {
		return nil
	}

	p := shamirPrime()
	secretInt := new(big.Int).SetBytes(secret)
	if secretInt.Cmp(p) >= 0 {
		return nil
	}
	coefficients := make([]*big.Int, t)
	coefficients[0] = secretInt
	for i := 1; i < t; i++ {
		c, err := rand.Int(rand.Reader, p)
		if err != nil {
			return nil
		}
		coefficients[i] = c
	}

	out := make([]byte, 0, shamirHeaderSize+n*(2+shamirFieldBytes))
	out = append(out, shamirMagic...)
	out = binary.BigEndian.AppendUint16(out, uint16(len(secret)))
	out = binary.BigEndian.AppendUint16(out, uint16(t))
	out = binary.BigEndian.AppendUint16(out, uint16(n))
	out = binary.BigEndian.AppendUint16(out, shamirFieldBytes)
	for x := 1; x <= n; x++ {
		y := shamirEval(coefficients, big.NewInt(int64(x)), p)
		out = binary.BigEndian.AppendUint16(out, uint16(x))
		out = append(out, shamirFixedBytes(y, shamirFieldBytes)...)
	}
	return out
}

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

func shamirSmallInt(v *big.Int, min, max int) (int, bool) {
	if v == nil || !v.IsInt64() {
		return 0, false
	}
	n := v.Int64()
	if n < int64(min) || n > int64(max) {
		return 0, false
	}
	return int(n), true
}

func shamirEval(coefficients []*big.Int, x, p *big.Int) *big.Int {
	result := new(big.Int)
	power := big.NewInt(1)
	for _, coefficient := range coefficients {
		term := new(big.Int).Mul(coefficient, power)
		result.Add(result, term)
		result.Mod(result, p)
		power.Mul(power, x)
		power.Mod(power, p)
	}
	return result
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
