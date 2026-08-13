package main

import "math/big"

func LabOneAdd(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}
