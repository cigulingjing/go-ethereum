package main

import "crypto/sha256"

// Sha256 returns the SHA-256 digest of data.
func Sha256(data []byte) []byte {
	sum := sha256.Sum256(data)
	out := make([]byte, len(sum))
	copy(out, sum[:])
	return out
}
