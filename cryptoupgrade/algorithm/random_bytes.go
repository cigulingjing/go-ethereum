package main

import (
	"crypto/rand"
	"math/big"
)

const randomBytesMaxLen = 1 << 20

// RandomBytes returns cryptographically secure random bytes.
func RandomBytes(length *big.Int) []byte {
	n, ok := randomBytesLength(length)
	if !ok {
		return nil
	}
	out := make([]byte, n)
	if _, err := rand.Read(out); err != nil {
		return nil
	}
	return out
}

func randomBytesLength(length *big.Int) (int, bool) {
	if length == nil || length.Sign() < 0 || !length.IsInt64() {
		return 0, false
	}
	n := length.Int64()
	if n > randomBytesMaxLen {
		return 0, false
	}
	return int(n), true
}
