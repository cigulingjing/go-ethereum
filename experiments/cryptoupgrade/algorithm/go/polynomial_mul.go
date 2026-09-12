package main

import "math/big"

const polynomialMulMaxLength = 256

// PolynomialMul 是三路径性能实验共享的朴素多项式乘法内核。
func PolynomialMul(left []*big.Int, right []*big.Int, modulus *big.Int) []*big.Int {
	if len(left) == 0 || len(right) == 0 || modulus == nil || modulus.Sign() <= 0 {
		return nil
	}
	if len(left) > polynomialMulMaxLength || len(right) > polynomialMulMaxLength {
		return nil
	}
	result := make([]*big.Int, len(left)+len(right)-1)
	for i := range result {
		result[i] = new(big.Int)
	}
	leftCoeff := new(big.Int)
	rightCoeff := new(big.Int)
	term := new(big.Int)
	for i, l := range left {
		if l == nil || l.Sign() < 0 {
			return nil
		}
		leftCoeff.Mod(l, modulus)
		for j, r := range right {
			if r == nil || r.Sign() < 0 {
				return nil
			}
			rightCoeff.Mod(r, modulus)
			term.Mul(leftCoeff, rightCoeff)
			term.Mod(term, modulus)
			result[i+j].Add(result[i+j], term)
			result[i+j].Mod(result[i+j], modulus)
		}
	}
	return result
}
