package main

import "math/big"

type ed25519Point struct {
	x, y, z, t *big.Int
}

var (
	ed25519Prime    = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	ed25519Order, _ = new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	ed25519D        = ed25519Mod(new(big.Int).Mul(big.NewInt(-121665), new(big.Int).ModInverse(big.NewInt(121666), ed25519Prime)))
	ed25519SqrtM1   = new(big.Int).Exp(big.NewInt(2), new(big.Int).Rsh(new(big.Int).Sub(ed25519Prime, big.NewInt(1)), 2), ed25519Prime)
	ed25519Base     = ed25519Point{
		x: ed25519Decimal("15112221349535400772501151409588531511454012693041857206046113283949847762202"),
		y: ed25519Decimal("46316835694926478169428394003475163141307993866256225615783033603165251855960"),
		z: big.NewInt(1),
	}
)

func init() {
	ed25519Base.t = ed25519Mod(new(big.Int).Mul(ed25519Base.x, ed25519Base.y))
}

func ed25519Decimal(value string) *big.Int {
	result, _ := new(big.Int).SetString(value, 10)
	return result
}

func ed25519Mod(value *big.Int) *big.Int {
	return value.Mod(value, ed25519Prime)
}

func ed25519LittleEndian(data []byte) *big.Int {
	reversed := make([]byte, len(data))
	for index := range data {
		reversed[index] = data[len(data)-1-index]
	}
	return new(big.Int).SetBytes(reversed)
}

func ed25519Decode(encoded []byte) (ed25519Point, bool) {
	copyBytes := append([]byte(nil), encoded...)
	sign := copyBytes[31] >> 7
	copyBytes[31] &= 0x7f
	y := ed25519LittleEndian(copyBytes)
	if y.Cmp(ed25519Prime) >= 0 {
		return ed25519Point{}, false
	}
	ySquared := ed25519Mod(new(big.Int).Mul(y, y))
	numerator := ed25519Mod(new(big.Int).Sub(ySquared, big.NewInt(1)))
	denominator := ed25519Mod(new(big.Int).Add(ed25519Mod(new(big.Int).Mul(ed25519D, ySquared)), big.NewInt(1)))
	inverse := new(big.Int).ModInverse(denominator, ed25519Prime)
	if inverse == nil {
		return ed25519Point{}, false
	}
	xSquared := ed25519Mod(new(big.Int).Mul(numerator, inverse))
	x := new(big.Int).Exp(xSquared, new(big.Int).Rsh(new(big.Int).Add(ed25519Prime, big.NewInt(3)), 3), ed25519Prime)
	if ed25519Mod(new(big.Int).Mul(x, x)).Cmp(xSquared) != 0 {
		x = ed25519Mod(new(big.Int).Mul(x, ed25519SqrtM1))
	}
	if ed25519Mod(new(big.Int).Mul(x, x)).Cmp(xSquared) != 0 || x.Sign() == 0 && sign != 0 {
		return ed25519Point{}, false
	}
	if x.Bit(0) != uint(sign) {
		x.Sub(ed25519Prime, x)
	}
	return ed25519Point{x, y, big.NewInt(1), ed25519Mod(new(big.Int).Mul(x, y))}, true
}

func ed25519Add(left, right ed25519Point) ed25519Point {
	a := ed25519Mod(new(big.Int).Mul(ed25519Mod(new(big.Int).Sub(left.y, left.x)), ed25519Mod(new(big.Int).Sub(right.y, right.x))))
	b := ed25519Mod(new(big.Int).Mul(ed25519Mod(new(big.Int).Add(left.y, left.x)), ed25519Mod(new(big.Int).Add(right.y, right.x))))
	c := ed25519Mod(new(big.Int).Mul(ed25519Mod(new(big.Int).Mul(big.NewInt(2), ed25519D)), ed25519Mod(new(big.Int).Mul(left.t, right.t))))
	d := ed25519Mod(new(big.Int).Mul(big.NewInt(2), ed25519Mod(new(big.Int).Mul(left.z, right.z))))
	e := ed25519Mod(new(big.Int).Sub(b, a))
	f := ed25519Mod(new(big.Int).Sub(d, c))
	g := ed25519Mod(new(big.Int).Add(d, c))
	h := ed25519Mod(new(big.Int).Add(b, a))
	return ed25519Point{
		ed25519Mod(new(big.Int).Mul(e, f)), ed25519Mod(new(big.Int).Mul(g, h)),
		ed25519Mod(new(big.Int).Mul(f, g)), ed25519Mod(new(big.Int).Mul(e, h)),
	}
}

func ed25519Multiply(point ed25519Point, scalar *big.Int) ed25519Point {
	result := ed25519Point{big.NewInt(0), big.NewInt(1), big.NewInt(1), big.NewInt(0)}
	for bit := scalar.BitLen() - 1; bit >= 0; bit-- {
		result = ed25519Add(result, result)
		if scalar.Bit(bit) != 0 {
			result = ed25519Add(result, point)
		}
	}
	return result
}

func ed25519Equal(left, right ed25519Point) bool {
	return ed25519Mod(new(big.Int).Mul(left.x, right.z)).Cmp(ed25519Mod(new(big.Int).Mul(right.x, left.z))) == 0 &&
		ed25519Mod(new(big.Int).Mul(left.y, right.z)).Cmp(ed25519Mod(new(big.Int).Mul(right.y, left.z))) == 0
}

func Ed25519Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != 32 || len(signature) != 64 || signature[63]&0xe0 != 0 {
		return false
	}
	scalar := ed25519LittleEndian(signature[32:])
	if scalar.Cmp(ed25519Order) >= 0 {
		return false
	}
	publicPoint, ok := ed25519Decode(publicKey)
	if !ok {
		return false
	}
	rPoint, ok := ed25519Decode(signature[:32])
	if !ok {
		return false
	}
	input := make([]byte, 0, 64+len(message))
	input = append(input, signature[:32]...)
	input = append(input, publicKey...)
	input = append(input, message...)
	challenge := ed25519LittleEndian(ed25519Sha512(input))
	challenge.Mod(challenge, ed25519Order)
	return ed25519Equal(ed25519Multiply(ed25519Base, scalar), ed25519Add(rPoint, ed25519Multiply(publicPoint, challenge)))
}

func ed25519Sha512(data []byte) []byte {
	state := [8]uint64{
		0x6a09e667f3bcc908, 0xbb67ae8584caa73b, 0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
		0x510e527fade682d1, 0x9b05688c2b3e6c1f, 0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
	}
	length := uint64(len(data)) * 8
	for len(data) >= 128 {
		ed25519Sha512Block(&state, data[:128])
		data = data[128:]
	}
	var tail [256]byte
	copy(tail[:], data)
	tail[len(data)] = 0x80
	finalLength := 128
	if len(data) >= 112 {
		finalLength = 256
	}
	for index := 0; index < 8; index++ {
		tail[finalLength-8+index] = byte(length >> (56 - 8*index))
	}
	ed25519Sha512Block(&state, tail[:128])
	if finalLength == 256 {
		ed25519Sha512Block(&state, tail[128:])
	}
	result := make([]byte, 64)
	for index, word := range state {
		for offset := 0; offset < 8; offset++ {
			result[index*8+offset] = byte(word >> (56 - 8*offset))
		}
	}
	return result
}

func ed25519Sha512Rotate(value uint64, shift uint) uint64 {
	return value>>shift | value<<(64-shift)
}

func ed25519Sha512Block(state *[8]uint64, data []byte) {
	var schedule [80]uint64
	for index := 0; index < 16; index++ {
		for offset := 0; offset < 8; offset++ {
			schedule[index] = schedule[index]<<8 | uint64(data[index*8+offset])
		}
	}
	for index := 16; index < 80; index++ {
		previous := schedule[index-15]
		first := ed25519Sha512Rotate(previous, 1) ^ ed25519Sha512Rotate(previous, 8) ^ previous>>7
		previous = schedule[index-2]
		second := ed25519Sha512Rotate(previous, 19) ^ ed25519Sha512Rotate(previous, 61) ^ previous>>6
		schedule[index] = schedule[index-16] + first + schedule[index-7] + second
	}
	working := *state
	for index := 0; index < 80; index++ {
		choice := (working[4] & working[5]) ^ (^working[4] & working[6])
		majority := (working[0] & working[1]) ^ (working[0] & working[2]) ^ (working[1] & working[2])
		first := working[7] + (ed25519Sha512Rotate(working[4], 14) ^ ed25519Sha512Rotate(working[4], 18) ^ ed25519Sha512Rotate(working[4], 41)) + choice + ed25519Sha512RoundConstants[index] + schedule[index]
		second := (ed25519Sha512Rotate(working[0], 28) ^ ed25519Sha512Rotate(working[0], 34) ^ ed25519Sha512Rotate(working[0], 39)) + majority
		working[7], working[6], working[5], working[4] = working[6], working[5], working[4], working[3]+first
		working[3], working[2], working[1], working[0] = working[2], working[1], working[0], first+second
	}
	for index := range state {
		state[index] += working[index]
	}
}

var ed25519Sha512RoundConstants = [80]uint64{
	0x428a2f98d728ae22,
	0x7137449123ef65cd,
	0xb5c0fbcfec4d3b2f,
	0xe9b5dba58189dbbc,
	0x3956c25bf348b538,
	0x59f111f1b605d019,
	0x923f82a4af194f9b,
	0xab1c5ed5da6d8118,
	0xd807aa98a3030242,
	0x12835b0145706fbe,
	0x243185be4ee4b28c,
	0x550c7dc3d5ffb4e2,
	0x72be5d74f27b896f,
	0x80deb1fe3b1696b1,
	0x9bdc06a725c71235,
	0xc19bf174cf692694,
	0xe49b69c19ef14ad2,
	0xefbe4786384f25e3,
	0x0fc19dc68b8cd5b5,
	0x240ca1cc77ac9c65,
	0x2de92c6f592b0275,
	0x4a7484aa6ea6e483,
	0x5cb0a9dcbd41fbd4,
	0x76f988da831153b5,
	0x983e5152ee66dfab,
	0xa831c66d2db43210,
	0xb00327c898fb213f,
	0xbf597fc7beef0ee4,
	0xc6e00bf33da88fc2,
	0xd5a79147930aa725,
	0x06ca6351e003826f,
	0x142929670a0e6e70,
	0x27b70a8546d22ffc,
	0x2e1b21385c26c926,
	0x4d2c6dfc5ac42aed,
	0x53380d139d95b3df,
	0x650a73548baf63de,
	0x766a0abb3c77b2a8,
	0x81c2c92e47edaee6,
	0x92722c851482353b,
	0xa2bfe8a14cf10364,
	0xa81a664bbc423001,
	0xc24b8b70d0f89791,
	0xc76c51a30654be30,
	0xd192e819d6ef5218,
	0xd69906245565a910,
	0xf40e35855771202a,
	0x106aa07032bbd1b8,
	0x19a4c116b8d2d0c8,
	0x1e376c085141ab53,
	0x2748774cdf8eeb99,
	0x34b0bcb5e19b48a8,
	0x391c0cb3c5c95a63,
	0x4ed8aa4ae3418acb,
	0x5b9cca4f7763e373,
	0x682e6ff3d6b2b8a3,
	0x748f82ee5defb2fc,
	0x78a5636f43172f60,
	0x84c87814a1f0ab72,
	0x8cc702081a6439ec,
	0x90befffa23631e28,
	0xa4506cebde82bde9,
	0xbef9a3f7b2c67915,
	0xc67178f2e372532b,
	0xca273eceea26619c,
	0xd186b8c721c0c207,
	0xeada7dd6cde0eb1e,
	0xf57d4f7fee6ed178,
	0x06f067aa72176fba,
	0x0a637dc5a2c898a6,
	0x113f9804bef90dae,
	0x1b710b35131c471b,
	0x28db77f523047d84,
	0x32caab7b40c72493,
	0x3c9ebe0a15c9bebc,
	0x431d67c49c100d4c,
	0x4cc5d4becb3e42b6,
	0x597f299cfc657e2a,
	0x5fcb6fab3ad6faec,
	0x6c44198c4a475817,
}
