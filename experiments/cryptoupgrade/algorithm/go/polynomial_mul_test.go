package main

import (
	"math/big"
	"testing"
)

func TestPolynomialMulMatchesBigIntReference(t *testing.T) {
	cases := []struct {
		left  []int64
		right []int64
		mod   int64
	}{
		{[]int64{1, 2}, []int64{3, 4}, 97},
		{[]int64{18, 2, 3}, []int64{4, 22}, 17},
		{makeRange(64, 100), makeRange(64, 100), 65537},
		{makeRange(32, 16), makeRange(32, 16), 12289},
	}
	for _, tc := range cases {
		left := intsToBigSlice(tc.left)
		right := intsToBigSlice(tc.right)
		mod := big.NewInt(tc.mod)
		got := PolynomialMul(left, right, mod)
		want := polynomialMulReference(left, right, mod)
		if !sameBigIntSlice(got, want) {
			t.Fatalf("mod=%d left=%d right=%d: got %v want %v", tc.mod, len(tc.left), len(tc.right), got, want)
		}
	}
}

func makeRange(n int, max int64) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = int64(i%int(max) + 1)
	}
	return out
}

func intsToBigSlice(values []int64) []*big.Int {
	out := make([]*big.Int, len(values))
	for i, value := range values {
		out[i] = big.NewInt(value)
	}
	return out
}

func polynomialMulReference(left, right []*big.Int, modulus *big.Int) []*big.Int {
	result := make([]*big.Int, len(left)+len(right)-1)
	for i := range result {
		result[i] = new(big.Int)
	}
	leftCoeff := new(big.Int)
	rightCoeff := new(big.Int)
	term := new(big.Int)
	for i, l := range left {
		leftCoeff.Mod(l, modulus)
		for j, r := range right {
			rightCoeff.Mod(r, modulus)
			term.Mul(leftCoeff, rightCoeff)
			term.Mod(term, modulus)
			result[i+j].Add(result[i+j], term)
			result[i+j].Mod(result[i+j], modulus)
		}
	}
	return result
}

func sameBigIntSlice(a, b []*big.Int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Cmp(b[i]) != 0 {
			return false
		}
	}
	return true
}
