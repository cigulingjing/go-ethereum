package main

import "math/big"

const pedersenPrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
	"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
	"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
	"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
	"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

// PedersenCommit returns g^message h^blinding mod p in the RFC 3526 2048-bit group.
func PedersenCommit(message, blinding []byte) []byte {
	p, q := pedersenGroup()
	g := big.NewInt(4)
	h := pedersenH(p)
	m := pedersenScalar(message, q)
	r := pedersenScalar(blinding, q)

	gm := new(big.Int).Exp(g, m, p)
	hr := new(big.Int).Exp(h, r, p)
	commitment := new(big.Int).Mul(gm, hr)
	commitment.Mod(commitment, p)
	return pedersenFixedBytes(commitment)
}

func pedersenGroup() (*big.Int, *big.Int) {
	p, _ := new(big.Int).SetString(pedersenPrimeHex, 16)
	q := new(big.Int).Sub(p, big.NewInt(1))
	q.Rsh(q, 1)
	return p, q
}

func pedersenH(p *big.Int) *big.Int {
	for counter := byte(0); ; counter++ {
		sum := pedersenSha256([]byte("cryptoupgrade-pedersen-h-" + string([]byte{counter})))
		h := new(big.Int).SetBytes(sum)
		h.Mod(h, p)
		h.Exp(h, big.NewInt(2), p)
		if h.Cmp(big.NewInt(1)) > 0 {
			return h
		}
	}
}

func pedersenScalar(data []byte, q *big.Int) *big.Int {
	x := new(big.Int).SetBytes(data)
	x.Mod(x, q)
	return x
}

func pedersenFixedBytes(x *big.Int) []byte {
	size := (len(pedersenPrimeHex) + 1) / 2
	b := x.Bytes()
	if len(b) > size {
		return b[len(b)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

var pedersenSha256RoundConstants = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

func pedersenSha256(data []byte) []byte {
	state := [8]uint32{
		0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
		0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
	}
	bitLength := uint64(len(data)) * 8
	for len(data) >= 64 {
		pedersenSha256Block(&state, data[:64])
		data = data[64:]
	}
	var finalBlocks [128]byte
	copy(finalBlocks[:], data)
	finalBlocks[len(data)] = 0x80
	finalLength := 64
	if len(data) >= 56 {
		finalLength = 128
	}
	for offset := 0; offset < 8; offset++ {
		finalBlocks[finalLength-8+offset] = byte(bitLength >> (56 - 8*offset))
	}
	pedersenSha256Block(&state, finalBlocks[:64])
	if finalLength == 128 {
		pedersenSha256Block(&state, finalBlocks[64:])
	}
	result := make([]byte, 32)
	for index, word := range state {
		result[index*4] = byte(word >> 24)
		result[index*4+1] = byte(word >> 16)
		result[index*4+2] = byte(word >> 8)
		result[index*4+3] = byte(word)
	}
	return result
}

func pedersenSha256Block(state *[8]uint32, block []byte) {
	var schedule [64]uint32
	for index := 0; index < 16; index++ {
		offset := index * 4
		schedule[index] = uint32(block[offset])<<24 | uint32(block[offset+1])<<16 |
			uint32(block[offset+2])<<8 | uint32(block[offset+3])
	}
	for index := 16; index < 64; index++ {
		previous := schedule[index-15]
		first := pedersenSha256Rotate(previous, 7) ^ pedersenSha256Rotate(previous, 18) ^ (previous >> 3)
		previous = schedule[index-2]
		second := pedersenSha256Rotate(previous, 17) ^ pedersenSha256Rotate(previous, 19) ^ (previous >> 10)
		schedule[index] = schedule[index-16] + first + schedule[index-7] + second
	}
	working := *state
	for round := 0; round < 64; round++ {
		choice := (working[4] & working[5]) ^ (^working[4] & working[6])
		majority := (working[0] & working[1]) ^ (working[0] & working[2]) ^ (working[1] & working[2])
		first := working[7] + (pedersenSha256Rotate(working[4], 6) ^ pedersenSha256Rotate(working[4], 11) ^ pedersenSha256Rotate(working[4], 25)) +
			choice + pedersenSha256RoundConstants[round] + schedule[round]
		second := (pedersenSha256Rotate(working[0], 2) ^ pedersenSha256Rotate(working[0], 13) ^ pedersenSha256Rotate(working[0], 22)) + majority
		working[7], working[6], working[5], working[4] = working[6], working[5], working[4], working[3]+first
		working[3], working[2], working[1], working[0] = working[2], working[1], working[0], first+second
	}
	for index := range state {
		state[index] += working[index]
	}
}

func pedersenSha256Rotate(value uint32, shift uint) uint32 {
	return value>>shift | value<<(32-shift)
}
