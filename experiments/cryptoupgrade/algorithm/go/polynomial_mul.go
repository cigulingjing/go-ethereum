package main

const polynomialMulMaxLength = 256

// PolynomialMul 是三路径性能实验共享的朴素多项式乘法内核。
// 使用 uint64 固定宽度运算，避免 math/big 在 TinyGo WASM 中的高开销。
func PolynomialMul(left []uint64, right []uint64, modulus uint64) []uint64 {
	if len(left) == 0 || len(right) == 0 || modulus == 0 {
		return nil
	}
	if len(left) > polynomialMulMaxLength || len(right) > polynomialMulMaxLength {
		return nil
	}
	result := make([]uint64, len(left)+len(right)-1)
	leftCoeff := uint64(0)
	rightCoeff := uint64(0)
	for i, l := range left {
		leftCoeff = l % modulus
		for j, r := range right {
			rightCoeff = r % modulus
			term := (leftCoeff * rightCoeff) % modulus
			result[i+j] = (result[i+j] + term) % modulus
		}
	}
	return result
}
