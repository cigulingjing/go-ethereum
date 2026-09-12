package main

import "math/big"

const dh2048PrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
	"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
	"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
	"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
	"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"

// Dh2048Secret computes a Diffie-Hellman shared secret.
func Dh2048Secret(privateKey, peerPublicKey []byte) []byte {
	p := dh2048Prime()
	x := new(big.Int).SetBytes(privateKey)
	if !dh2048ValidScalar(x, p) {
		return nil
	}
	y := new(big.Int).SetBytes(peerPublicKey)
	if y.Cmp(big.NewInt(1)) <= 0 || y.Cmp(new(big.Int).Sub(p, big.NewInt(1))) >= 0 {
		return nil
	}
	secret := new(big.Int).Exp(y, x, p)
	return dh2048FixedBytes(secret, dh2048FieldBytes())
}

func dh2048Prime() *big.Int {
	p, ok := new(big.Int).SetString(dh2048PrimeHex, 16)
	if !ok {
		return nil
	}
	return p
}

func dh2048ValidScalar(x, p *big.Int) bool {
	return x.Cmp(big.NewInt(1)) > 0 && x.Cmp(new(big.Int).Sub(p, big.NewInt(1))) < 0
}

func dh2048FieldBytes() int {
	return (len(dh2048PrimeHex) + 1) / 2
}

func dh2048FixedBytes(x *big.Int, size int) []byte {
	if x == nil {
		return nil
	}
	b := x.Bytes()
	if len(b) > size {
		return b[len(b)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}
