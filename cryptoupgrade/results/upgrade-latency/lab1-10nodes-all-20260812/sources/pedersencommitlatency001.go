package main

import (
	"crypto/sha256"
	"math/big"
)

const pedersenPrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
	"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
	"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
	"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
	"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

// PedersenCommit returns g^message h^blinding mod p in the RFC 3526 2048-bit group.
func PedersenCommitLatency001(message, blinding []byte) []byte {
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
		sum := sha256.Sum256([]byte("cryptoupgrade-pedersen-h-" + string([]byte{counter})))
		h := new(big.Int).SetBytes(sum[:])
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
