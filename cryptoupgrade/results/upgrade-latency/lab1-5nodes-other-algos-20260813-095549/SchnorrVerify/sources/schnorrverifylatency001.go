package main

import (
	"bytes"
	"crypto/sha256"
	"math/big"
)

const schnorrPrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
	"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
	"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
	"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
	"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

// SchnorrVerify verifies an encoded Schnorr proof.
func SchnorrVerifyLatency001(message, proof []byte) bool {
	size := schnorrFieldBytes()
	if len(proof) != 3*size {
		return false
	}
	p, q := schnorrGroup()
	y := new(big.Int).SetBytes(proof[:size])
	t := new(big.Int).SetBytes(proof[size : 2*size])
	s := new(big.Int).SetBytes(proof[2*size:])
	if !schnorrGroupElement(y, p) || !schnorrGroupElement(t, p) || s.Cmp(q) >= 0 {
		return false
	}

	g := big.NewInt(4)
	c := schnorrChallenge(q, y, t, message)
	left := new(big.Int).Exp(g, s, p)
	yc := new(big.Int).Exp(y, c, p)
	right := new(big.Int).Mul(t, yc)
	right.Mod(right, p)
	return bytes.Equal(schnorrFixedBytes(left), schnorrFixedBytes(right))
}

func schnorrGroup() (*big.Int, *big.Int) {
	p, _ := new(big.Int).SetString(schnorrPrimeHex, 16)
	q := new(big.Int).Sub(p, big.NewInt(1))
	q.Rsh(q, 1)
	return p, q
}

func schnorrChallenge(q, y, t *big.Int, message []byte) *big.Int {
	h := sha256.New()
	h.Write([]byte("cryptoupgrade-schnorr"))
	h.Write(schnorrFixedBytes(y))
	h.Write(schnorrFixedBytes(t))
	h.Write(message)
	c := new(big.Int).SetBytes(h.Sum(nil))
	c.Mod(c, q)
	return c
}

func schnorrGroupElement(x, p *big.Int) bool {
	return x.Cmp(big.NewInt(1)) > 0 && x.Cmp(new(big.Int).Sub(p, big.NewInt(1))) < 0
}

func schnorrFieldBytes() int {
	return (len(schnorrPrimeHex) + 1) / 2
}

func schnorrFixedBytes(x *big.Int) []byte {
	size := schnorrFieldBytes()
	b := x.Bytes()
	if len(b) > size {
		return b[len(b)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}
