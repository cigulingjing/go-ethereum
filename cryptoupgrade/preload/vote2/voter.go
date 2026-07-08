package vote

import (
	"crypto/rand"
	"math/bits"
	"time"

	. "github.com/ethereum/go-ethereum/cryptoupgrade/preload/bls12381"
)

type VoterBallotR1 struct {
	l        uint64
	h        uint64
	m        uint64
	index    []uint64
	scorevec [][]*Fr
	scorer   [][]*Fr
	experte  []*Fr
	expertr  []*Fr
}

type VoterBallotCipherR1 struct {
	scoreCipher  [][]*SingleElemCipher
	expertCipher []*SingleElemCipher
}

type VoterBallotZKPR1 struct {
	VectorZKP []*BallotZKP
}

func (ballot *VoterBallotCipherR1) Size() int {
	return (len(ballot.scoreCipher)*len(ballot.scoreCipher[0]) + len(ballot.expertCipher)) * ballot.scoreCipher[0][0].Size()
}

func (zkp *VoterBallotZKPR1) Size() int {
	return zkp.VectorZKP[0].Size() * len(zkp.VectorZKP)
}

// index为分数和专家向量合并后的索引，若委托，index所有项一样，若不委托，index两两不同
func GenerateVoterBallotR1(h, l, m uint64, index []uint64) (*VoterBallotR1, error) {
	ballot := &VoterBallotR1{}
	ballot.l = l
	ballot.h = h
	ballot.m = m

	for f := uint64(0); f < h; f++ {
		ballot.scorevec = append(ballot.scorevec, []*Fr{})
		ballot.scorer = append(ballot.scorer, []*Fr{})
		ballot.index = append(ballot.index, index[f])
	}

	for f := uint64(0); f < h; f++ {
		i := index[f]
		for lambda := uint64(0); lambda < l; lambda++ {
			if lambda == i {
				ballot.scorevec[f] = append(ballot.scorevec[f], NewFr().One())
			} else {
				ballot.scorevec[f] = append(ballot.scorevec[f], NewFr().Zero())
			}
			r, _ := NewFr().Rand(rand.Reader)
			ballot.scorer[f] = append(ballot.scorer[f], r)
		}
	}

	i := index[0]
	for j := uint64(0); j < m; j++ {
		if j+l == i {
			ballot.experte = append(ballot.experte, NewFr().One())
		} else {
			ballot.experte = append(ballot.experte, NewFr().Zero())
		}
		r, _ := NewFr().Rand(rand.Reader)
		ballot.expertr = append(ballot.expertr, r)
	}

	return ballot, nil
}

func EncryptVoterBallotR1(ballot *VoterBallotR1, g *PointG1, pk *PointG1) *VoterBallotCipherR1 {
	l := ballot.l
	h := ballot.h
	m := ballot.m

	res := &VoterBallotCipherR1{}
	for f := uint64(0); f < h; f++ {
		res.scoreCipher = append(res.scoreCipher, []*SingleElemCipher{})
	}

	for f := uint64(0); f < h; f++ {
		for lambda := uint64(0); lambda < l; lambda++ {
			cipher := &SingleElemCipher{}
			cipher.C1, cipher.C2 = encrypt_m_r(g, pk, ballot.scorevec[f][lambda], ballot.scorer[f][lambda])
			res.scoreCipher[f] = append(res.scoreCipher[f], cipher)
		}
	}
	for j := uint64(0); j < m; j++ {
		cipher := &SingleElemCipher{}
		cipher.C1, cipher.C2 = encrypt_m_r(g, pk, ballot.experte[j], ballot.expertr[j])
		res.expertCipher = append(res.expertCipher, cipher)
	}
	return res
}

func nextPowerOfTwo(n uint64) uint64 {
	return 1 << bits.Len(uint(n))
}

func vector_cat(ballotv1 *VoterBallotR1, f uint64) *Ballot {
	ballot := &Ballot{}
	ballot.n = ballotv1.l + ballotv1.m
	ballot.len = nextPowerOfTwo(ballot.n)
	ballot.index = ballotv1.index[f]
	ballot.e = append(ballotv1.scorevec[f], ballotv1.experte...)
	ballot.r = append(ballotv1.scorer[f], ballotv1.expertr...)
	for i := 0; i < int(ballot.len)-int(ballot.n); i++ {
		ballot.e = append(ballot.e, NewFr().Zero())
		ballot.r = append(ballot.r, NewFr().Zero())
	}
	return ballot
}

func GenerateVoterBallotVectorZKPR1(ballotv1 *VoterBallotR1, g *PointG1, pk *PointG1, ck *PointG1) (*VoterBallotZKPR1, time.Duration) {
	res := &VoterBallotZKPR1{}
	d := time.Duration(0)
	for f := uint64(0); f < ballotv1.h; f++ {
		ballot := vector_cat(ballotv1, f)
		zkp, tmp := GenerateBallotZKP(ballot, g, pk, ck)
		d += tmp
		res.VectorZKP = append(res.VectorZKP, zkp)
	}
	return res, d
}

func VerifyVoterBallotVectorZKPR1(cipherv1 *VoterBallotCipherR1, zkp *VoterBallotZKPR1, g *PointG1, pk *PointG1, ck *PointG1) bool {
	h := len(cipherv1.scoreCipher)
	l := len(cipherv1.scoreCipher[0])
	m := len(cipherv1.expertCipher)
	len := nextPowerOfTwo(uint64(l + m))
	c1, c2 := encrypt_m_r(g, pk, NewFr().Zero(), NewFr().Zero())
	normal := &SingleElemCipher{
		c1,
		c2,
	}
	for f := 0; f < h; f++ {
		single_ballot_cipher := &BallotCipher{}
		single_ballot_cipher.n = uint64(l + m)
		single_ballot_cipher.cipher = cipherv1.scoreCipher[f]
		single_ballot_cipher.cipher = append(single_ballot_cipher.cipher, cipherv1.expertCipher...)
		for k := uint64(0); k < len-single_ballot_cipher.n; k++ {
			single_ballot_cipher.cipher = append(single_ballot_cipher.cipher, normal)
		}
		if !VerifyBallotZKP(*single_ballot_cipher, zkp.VectorZKP[f], g, pk, ck) {
			return false
		}
	}
	return true
}
