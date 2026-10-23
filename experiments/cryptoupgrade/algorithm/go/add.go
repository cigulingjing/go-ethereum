package main

// Int256 是 ABI 中的 int256：32 字节、大端、二进制补码。
// 与 uint256 的加法在比特模式上相同，因此两条签名共用这一表示。
type Int256 [32]byte

// Add 计算两个 int256 的和。
// 按 256 位字从低字节进位，不使用 math/big，避免 TinyGo 链入通用大整数运行时。
// 和按 2^256 回绕，与补码加法的比特结果一致。
func Add(a, b Int256) Int256 {
	var sum Int256
	carry := uint16(0)
	for i := len(sum) - 1; i >= 0; i-- {
		total := uint16(a[i]) + uint16(b[i]) + carry
		sum[i] = byte(total)
		carry = total >> 8
	}
	return sum
}
