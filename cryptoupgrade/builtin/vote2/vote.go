package vote

import (
	"bytes"
	"crypto/rand"
	"errors"
	. "github.com/ethereum/go-ethereum/cryptoupgrade/builtin/bls12381"
	"math"
	"time"
)

type PauseTimer struct {
	startTime   time.Time
	pauseTime   time.Time
	totalPaused time.Duration
	paused      bool
}

func NewPauseTimer() *PauseTimer {
	return &PauseTimer{
		startTime: time.Now(),
	}
}

func (pt *PauseTimer) Pause() {
	if !pt.paused {
		pt.pauseTime = time.Now()
		pt.paused = true
	}
}

func (pt *PauseTimer) Resume() {
	if pt.paused {
		pt.totalPaused += time.Since(pt.pauseTime)
		pt.paused = false
	}
}

func (pt *PauseTimer) Elapsed() time.Duration {
	if pt.paused {
		return pt.pauseTime.Sub(pt.startTime) - pt.totalPaused
	}
	return time.Since(pt.startTime) - pt.totalPaused
}

type SingleElemCipher struct {
	C1 *PointG1
	C2 *PointG1
}

func (cipher *SingleElemCipher) Size() int {
	// group1 := NewG1()
	// return len(group1.ToBytes(cipher.C1)) * 2
	return 65 * 2
}

type Ballot struct {
	n     uint64 //实际长度
	len   uint64 //2的幂次长度
	index uint64
	e     []*Fr
	r     []*Fr
}

type BallotCipher struct {
	n      uint64              //实际长度
	cipher []*SingleElemCipher //2的幂次长度
}

func (ballotcipher BallotCipher) Size() int {
	s := 0
	// group1 := NewG1()
	// s += len(group1.ToBytes(ballotcipher.cipher[0].C1)) * 2 * int(ballotcipher.n)
	s += 65 * 2 * int(ballotcipher.n)
	return s
}

type BallotZKP struct {
	I []*PointG1
	B []*PointG1
	A []*PointG1
	D []*PointG1
	z []*Fr
	w []*Fr
	v []*Fr
	R *Fr
}

type CoefficientOfLine [2]*Fr

func NewBallotZKP() *BallotZKP {
	return &BallotZKP{
		I: make([]*PointG1, 1),
		B: make([]*PointG1, 1),
		A: make([]*PointG1, 1),
		D: make([]*PointG1, 0),
		z: make([]*Fr, 1),
		w: make([]*Fr, 1),
		v: make([]*Fr, 1),
		R: NewFr(),
	}
}

func (zkp *BallotZKP) Size() int {
	s := 0
	// group1 := NewG1()
	// s += len(group1.ToBytes(zkp.I[1]))*(len(zkp.I)-1)*3 + len(group1.ToBytes(zkp.D[0]))*len(zkp.D) + len(zkp.v[1].ToBytes())*((len(zkp.v)-1)*3+1)
	s += 65*(len(zkp.I)-1)*3 + 65*len(zkp.D) + 32*((len(zkp.v)-1)*3+1)
	return s
}

func encrypt_m_r(g *PointG1, pk *PointG1, m *Fr, r *Fr) (c1 *PointG1, c2 *PointG1) {
	group1 := NewG1()
	// C1 = g ^ r
	c1 = group1.MulScalar(group1.New(), g, r)
	// C2 = mPoint * y ^ r
	c2 = group1.Add(group1.New(), group1.MulScalar(group1.New(), g, m), group1.MulScalar(group1.New(), pk, r))
	return c1, c2
}

func GenerateBallot(n, len uint64, index uint64) (*Ballot, error) {
	ballot := &Ballot{}
	ballot.n = n
	ballot.len = len
	ballot.index = index
	if index >= n {
		return nil, errors.New("wrong index")
	}
	for i := uint64(0); i < ballot.len; i++ {
		if i == index {
			ballot.e = append(ballot.e, NewFr().One())
		} else {
			ballot.e = append(ballot.e, NewFr().Zero())
		}
		if i < n {
			r, _ := NewFr().Rand(rand.Reader)
			ballot.r = append(ballot.r, r)
		} else {
			ballot.r = append(ballot.r, NewFr().Zero())
		}

	}
	return ballot, nil
}

func EncryptBallot(ballot *Ballot, g *PointG1, pk *PointG1) BallotCipher {
	var res BallotCipher
	res.n = ballot.n
	len := ballot.len
	for i := uint64(0); i < len; i++ {
		cipher := &SingleElemCipher{}
		cipher.C1, cipher.C2 = encrypt_m_r(g, pk, ballot.e[i], ballot.r[i])
		res.cipher = append(res.cipher, cipher)
	}
	return res
}

func coefficient_of_mul_polynomials(polynomials []CoefficientOfLine) []*Fr {
	n := len(polynomials)
	f := make([][]*Fr, n+1)
	for i := range f {
		f[i] = make([]*Fr, n+1)
	}
	// 边界条件
	f[0][0] = NewFr().One()

	// 递推计算
	for i := 1; i <= n; i++ {

		a, b := polynomials[i-1][0], polynomials[i-1][1] // 第 i 个多项式的系数
		for j := 0; j <= i; j++ {
			if f[i-1][j] == nil {
				f[i-1][j] = NewFr().Zero()
			}
			f[i][j] = NewFr().Mul(f[i-1][j], b) // 不选 x 项
			if j > 0 {
				f[i][j] = NewFr().Add(f[i][j], NewFr().Mul(f[i-1][j-1], a)) // 选 x 项
			}
		}
	}
	return f[n]
}

func uint64_bin_representation(k, len uint64) []*Fr {
	bin_representation := make([]*Fr, len+1)
	for i := uint64(1); i <= len; i++ {
		bin_representation[i] = FrFromInt(int((k >> (len - i)) & 1))
	}
	return bin_representation
}

func coefficient_of_pj_polynomials(j, index, len uint64, beta_vec []*Fr) []*Fr {
	index_bin_representation := uint64_bin_representation(index, len)
	j_bin_representation := uint64_bin_representation(j, len)
	polynomials := make([]CoefficientOfLine, len)
	for l := uint64(1); l <= len; l++ {
		if j_bin_representation[l].Equal(NewFr().One()) {
			polynomials[l-1][0] = index_bin_representation[l]
			polynomials[l-1][1] = beta_vec[l]
		} else {
			polynomials[l-1][0] = NewFr().Sub(NewFr().One(), index_bin_representation[l])
			polynomials[l-1][1] = NewFr().Sub(NewFr().Zero(), beta_vec[l])
		}
	}

	return coefficient_of_mul_polynomials(polynomials)
}

func GenerateBallotZKP(ballot *Ballot, g *PointG1, pk *PointG1, ck *PointG1) (*BallotZKP, time.Duration) {
	timer := NewPauseTimer()
	group1 := NewG1()
	len := ballot.len
	index := ballot.index
	log_len := uint64(math.Log2(float64(len)))
	zkp := NewBallotZKP()
	var res_bytes bytes.Buffer
	alpha_vec := make([]*Fr, 1)
	beta_vec := make([]*Fr, 1)
	gamma_vec := make([]*Fr, 1)
	delta_vec := make([]*Fr, 1)
	index_bin_representation := uint64_bin_representation(index, log_len)
	for l := uint64(1); l <= log_len; l++ {
		alpha, _ := NewFr().Rand(rand.Reader)
		beta, _ := NewFr().Rand(rand.Reader)
		gamma, _ := NewFr().Rand(rand.Reader)
		delta, _ := NewFr().Rand(rand.Reader)
		alpha_vec = append(alpha_vec, alpha)
		beta_vec = append(beta_vec, beta)
		gamma_vec = append(gamma_vec, gamma)
		delta_vec = append(delta_vec, delta)
		index_at_l := index_bin_representation[l]
		I := group1.Add(group1.New(), group1.MulScalar(group1.New(), g, index_at_l), group1.MulScalar(group1.New(), ck, alpha))
		B := group1.Add(group1.New(), group1.MulScalar(group1.New(), g, beta), group1.MulScalar(group1.New(), ck, gamma))
		A := group1.Add(group1.New(), group1.MulScalar(group1.New(), g, NewFr().Mul(index_at_l, beta)), group1.MulScalar(group1.New(), ck, delta))
		zkp.I = append(zkp.I, I)
		zkp.B = append(zkp.B, B)
		zkp.A = append(zkp.A, A)
		res_bytes.Write(group1.ToBytes(I))
		res_bytes.Write(group1.ToBytes(B))
		res_bytes.Write(group1.ToBytes(A))
	}
	y := HashToFr(res_bytes.Bytes())

	res_bytes.Reset()
	Rl_vec := make([]*Fr, 0)
	for l := uint64(0); l < log_len; l++ {
		Rl, _ := NewFr().Rand(rand.Reader)
		Rl_vec = append(Rl_vec, Rl)

		temp_sum := NewFr().Zero()
		for j := uint64(0); j < len; j++ {
			timer.Pause()
			coefficients := coefficient_of_pj_polynomials(j, index, log_len, beta_vec)
			timer.Resume()
			temp_sum = NewFr().Add(temp_sum, NewFr().Mul(coefficients[l], fr_pow(y, j)))
		}

		_, D := encrypt_m_r(g, pk, temp_sum, Rl)
		zkp.D = append(zkp.D, D)
		res_bytes.Write(group1.ToBytes(D))
	}
	res_bytes.Write(y.ToBytes())
	x := HashToFr(res_bytes.Bytes())

	R := NewFr().Zero()
	for j := uint64(0); j < len; j++ {
		R = NewFr().Add(R, NewFr().Mul(NewFr().Mul(ballot.r[j], fr_pow(x, log_len)), fr_pow(y, j)))
	}
	for l := uint64(0); l < log_len; l++ {
		R = NewFr().Add(R, NewFr().Mul(Rl_vec[l], fr_pow(x, l)))
	}
	zkp.R = R

	for l := uint64(1); l <= log_len; l++ {
		z := NewFr().Add(NewFr().Mul(index_bin_representation[l], x), beta_vec[l])
		w := NewFr().Add(NewFr().Mul(alpha_vec[l], x), gamma_vec[l])
		v := NewFr().Add(NewFr().Mul(alpha_vec[l], NewFr().Sub(x, z)), delta_vec[l])
		zkp.z = append(zkp.z, z)
		zkp.w = append(zkp.w, w)
		zkp.v = append(zkp.v, v)
	}
	return zkp, timer.Elapsed()
}

func VerifyBallotZKP(ballot_cipher BallotCipher, zkp *BallotZKP, g *PointG1, pk *PointG1, ck *PointG1) bool {
	group1 := NewG1()

	len := uint64(len(ballot_cipher.cipher))
	log_len := uint64(math.Log2(float64(len)))
	var res_bytes bytes.Buffer
	for i := uint64(1); i <= log_len; i++ {
		res_bytes.Write(group1.ToBytes(zkp.I[i]))
		res_bytes.Write(group1.ToBytes(zkp.B[i]))
		res_bytes.Write(group1.ToBytes(zkp.A[i]))
	}
	y := HashToFr(res_bytes.Bytes())

	res_bytes.Reset()
	for i := uint64(0); i < log_len; i++ {
		res_bytes.Write(group1.ToBytes(zkp.D[i]))
	}
	res_bytes.Write(y.ToBytes())
	x := HashToFr(res_bytes.Bytes())

	x_powers := make([]*Fr, log_len+1)
	x_powers[0] = NewFr().One()
	for i := uint64(1); i <= log_len; i++ {
		x_powers[i] = fr_pow(x, i) // 或 x_powers[i] = NewFr().Mul(x_powers[i-1], x)
	}

	// 缓存 y 的幂次（最大为 len-1）
	y_powers := make([]*Fr, len)
	y_powers[0] = NewFr().One()
	for j := uint64(1); j < len; j++ {
		y_powers[j] = NewFr().Mul(y_powers[j-1], y)
	}

	for i := uint64(1); i <= log_len; i++ {
		if !group1.Equal(group1.Add(group1.New(), group1.MulScalar(group1.New(), zkp.I[i], x), zkp.B[i]), group1.Add(group1.New(), group1.MulScalar(group1.New(), g, zkp.z[i]), group1.MulScalar(group1.New(), ck, zkp.w[i]))) {
			return false
		}
		if !group1.Equal(group1.Add(group1.New(), group1.MulScalar(group1.New(), zkp.I[i], NewFr().Sub(x, zkp.z[i])), zkp.A[i]), group1.Add(group1.New(), group1.MulScalar(group1.New(), g, NewFr().Zero()), group1.MulScalar(group1.New(), ck, zkp.v[i]))) {
			return false
		}
	}
	x_log_len := fr_pow(x, log_len)
	left_sum1 := group1.New().Zero()
	for j := uint64(0); j < len; j++ {
		j_bin_representation := uint64_bin_representation(j, log_len)
		mul_temp := NewFr().One()
		for l := uint64(1); l <= log_len; l++ {
			if j_bin_representation[l].Equal(NewFr().Zero()) {
				mul_temp = NewFr().Mul(mul_temp, NewFr().Sub(x, zkp.z[l]))
			} else {
				mul_temp = NewFr().Mul(mul_temp, zkp.z[l])
			}
		}
		mul_temp = NewFr().Sub(NewFr().Zero(), mul_temp)

		_, enc_temp := encrypt_m_r(g, pk, mul_temp, NewFr().Zero())

		temp := group1.Add(group1.New(), group1.MulScalar(group1.New(), ballot_cipher.cipher[j].C2, x_log_len), enc_temp)

		left_sum1 = group1.Add(group1.New(), left_sum1, group1.MulScalar(group1.New(), temp, y_powers[j]))
	}

	left_sum2 := group1.New().Zero()
	for l := uint64(0); l < log_len; l++ {
		left_sum2 = group1.Add(group1.New(), left_sum2, group1.MulScalar(group1.New(), zkp.D[l], x_powers[l]))
	}

	_, right_c2 := encrypt_m_r(g, pk, NewFr().Zero(), zkp.R)
	return group1.Equal(group1.Add(group1.New(), left_sum1, left_sum2), right_c2)
}

func fr_pow(x *Fr, k uint64) *Fr {
	res := NewFr().One()   // res = 1
	base := NewFr().Set(x) // base = x（避免修改原始 x）

	for k > 0 {
		if k&1 == 1 {
			res = NewFr().Mul(res, base)
		}
		base = NewFr().Mul(base, base)
		k >>= 1
	}
	return res
}
