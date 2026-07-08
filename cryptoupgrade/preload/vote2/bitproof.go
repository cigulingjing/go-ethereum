package vote

import (
	. "github.com/ethereum/go-ethereum/cryptoupgrade/preload/bls12381"
	"bytes"
	"crypto/rand"
)

type BitProof struct {
	A1, B1 *PointG1
	A2, B2 *PointG1
	D1, D2 *Fr
	R1, R2 *Fr
}

func (zkp *BitProof) Size() int {
	group1 := NewG1()
	return len(group1.ToBytes(zkp.A1))*4 + len(zkp.D1.ToBytes())*4
}

func GenerateBitProof(m, r *Fr, g, pk *PointG1) *BitProof {
	group := NewG1()

	// 模拟一个假的挑战 + 真正的挑战之和 = c
	w, _ := NewFr().Rand(rand.Reader)
	v, _ := NewFr().Rand(rand.Reader)
	d, _ := NewFr().Rand(rand.Reader)

	var x, y, a1, b1, a2, b2 *PointG1
	var d1, v1 *Fr

	if m.IsOne() {
		// 真：v = 1，对 x = g^xj, y = h^xj * g
		x = group.MulScalar(group.New(), g, r)
		y = group.Add(group.New(), group.MulScalar(group.New(), pk, r), g)
		a1 = group.Add(group.New(), group.MulScalar(group.New(), g, v), group.MulScalar(group.New(), x, d))
		b1 = group.Add(group.New(), group.MulScalar(group.New(), pk, v), group.MulScalar(group.New(), y, d))

		// 模拟另一边（v=0）
		a2 = group.MulScalar(group.New(), g, w)
		b2 = group.MulScalar(group.New(), pk, w)

	} else {
		x = group.MulScalar(group.New(), g, r)
		y = group.MulScalar(group.New(), pk, r)
		// 真：v = 0，对 x = g^xj, y = h^xj
		a1 = group.MulScalar(group.New(), g, w)
		b1 = group.MulScalar(group.New(), pk, w)
		a2 = group.Add(group.New(), group.MulScalar(group.New(), g, v), group.MulScalar(group.New(), x, d))
		tmp := group.Sub(group.New(), y, g)
		b2 = group.Add(group.New(), group.MulScalar(group.New(), pk, v), group.MulScalar(group.New(), tmp, d))
	}

	// Fiat-Shamir challenge
	var buf bytes.Buffer
	buf.Write(group.ToBytes(x))
	buf.Write(group.ToBytes(y))
	buf.Write(group.ToBytes(a1))
	buf.Write(group.ToBytes(b1))
	buf.Write(group.ToBytes(a2))
	buf.Write(group.ToBytes(b2))
	c := HashToFr(buf.Bytes()) // c = d1 + d2

	if m.IsOne() {
		d1 = NewFr().Sub(c, d)
		v1 = NewFr().Sub(w, NewFr().Mul(r, d1))
	} else {
		d1 = d
		v1 = v
		d = NewFr().Sub(c, d1)
		v = NewFr().Sub(w, NewFr().Mul(r, d))
	}

	return &BitProof{
		A1: a1,
		B1: b1,
		A2: a2,
		B2: b2,
		D1: d,
		D2: d1,
		R1: v,
		R2: v1,
	}
}

func VerifyBitProof(cipher *SingleElemCipher, proof *BitProof, g *PointG1, pk *PointG1) bool {
	group := NewG1()

	var buf bytes.Buffer
	buf.Write(group.ToBytes(cipher.C1))
	buf.Write(group.ToBytes(cipher.C2))
	buf.Write(group.ToBytes(proof.A1))
	buf.Write(group.ToBytes(proof.B1))
	buf.Write(group.ToBytes(proof.A2))
	buf.Write(group.ToBytes(proof.B2))
	c := HashToFr(buf.Bytes()) // c = d1 + d2

	if !c.Equal(NewFr().Add(proof.D1, proof.D2)) {
		return false
	}

	if !group.Equal(proof.A1, group.Add(group.New(), group.MulScalar(group.New(), g, proof.R1), group.MulScalar(group.New(), cipher.C1, proof.D1))) {
		return false
	}
	if !group.Equal(proof.B1, group.Add(group.New(), group.MulScalar(group.New(), pk, proof.R1), group.MulScalar(group.New(), cipher.C2, proof.D1))) {
		return false
	}
	if !group.Equal(proof.A2, group.Add(group.New(), group.MulScalar(group.New(), g, proof.R2), group.MulScalar(group.New(), cipher.C1, proof.D2))) {
		return false
	}
	tmp := group.Sub(group.New(), cipher.C2, g)
	return group.Equal(proof.B2, group.Add(group.New(), group.MulScalar(group.New(), pk, proof.R2), group.MulScalar(group.New(), tmp, proof.D2)))
}
