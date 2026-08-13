package main

import "math/big"

// Add returns the sum of two integer values.
func AddLatency001(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}
