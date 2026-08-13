package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

const (
	pbkdf2MaxIterations = 1000000
	pbkdf2MaxKeyLength  = 1 << 20
)

// Pbkdf2Sha256 derives a key using PBKDF2-HMAC-SHA256.
func Pbkdf2Sha256Latency001(password, salt []byte, iterations, keyLength *big.Int) []byte {
	iter, ok := pbkdf2BoundedInt(iterations, 1, pbkdf2MaxIterations)
	if !ok {
		return nil
	}
	keyLen, ok := pbkdf2BoundedInt(keyLength, 0, pbkdf2MaxKeyLength)
	if !ok {
		return nil
	}
	if keyLen == 0 {
		return []byte{}
	}

	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	blocks := (keyLen + hashLen - 1) / hashLen
	var blockIndex [4]byte
	u := make([]byte, 0, hashLen)
	t := make([]byte, hashLen)
	derived := make([]byte, 0, blocks*hashLen)

	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(blockIndex[:], uint32(block))
		prf.Write(blockIndex[:])
		u = prf.Sum(u[:0])
		copy(t, u)

		for round := 2; round <= iter; round++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for i := range t {
				t[i] ^= u[i]
			}
		}
		derived = append(derived, t...)
	}
	return derived[:keyLen]
}

func pbkdf2BoundedInt(v *big.Int, min, max int) (int, bool) {
	if v == nil || !v.IsInt64() {
		return 0, false
	}
	n := v.Int64()
	if n < int64(min) || n > int64(max) {
		return 0, false
	}
	return int(n), true
}
