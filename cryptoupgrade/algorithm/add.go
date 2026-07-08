package main

import "math/big"

func Add(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}
