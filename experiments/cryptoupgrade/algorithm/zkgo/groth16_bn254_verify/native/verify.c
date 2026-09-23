#include <mcl/bn_c256.h>

#include <stdint.h>
#include <stddef.h>
#include <string.h>

/* 从 gnark Groth16 BN254 Verify 抽出的无 commitment pairing 方程：
 *   e(Ar, Bs) · e(Krs, -δ) · e(L_pub, -γ) · e(α, -β) = 1
 *
 * 曲线是 Ethereum alt_bn128，对应 mcl MCL_BN_SNARK1（不是 mcl BN254）。
 * testdata 为 gnark RawBytes：G1 是 X||Y；G2 是 X.A1|X.A0|Y.A1|Y.A0，
 * 写入 mcl Jacobian 时按 A0+A1·i 还原。
 */
#define G1_SIZE 64
#define G2_SIZE 128
#define FR_SIZE 32
#define PROOF_SIZE (G1_SIZE + G2_SIZE + G1_SIZE)
#define VK_HEADER (G1_SIZE + G2_SIZE + G2_SIZE + G2_SIZE + 4)
#define MAX_PUBLIC 16

static uint32_t read_u32_be(const uint8_t *p) {
	return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | (uint32_t)p[3];
}

static int ensure_init(void) {
	static int ready = 0;
	static int ok = 0;
	if (!ready) {
		ok = mclBn_init(MCL_BN_SNARK1, MCLBN_COMPILED_TIME_VAR) == 0;
		ready = 1;
	}
	return ok ? 0 : -1;
}

static int g1_from_raw(mclBnG1 *out, const uint8_t *raw) {
	if (mclBnFp_setBigEndianMod(&out->x, raw, 32) != 0) {
		return -1;
	}
	if (mclBnFp_setBigEndianMod(&out->y, raw + 32, 32) != 0) {
		return -1;
	}
	mclBnFp_setInt32(&out->z, 1);
	return 0;
}

static int g2_from_raw(mclBnG2 *out, const uint8_t *raw) {
	/* gnark RawBytes: X.A1|X.A0|Y.A1|Y.A0 ；mcl Fp2: d[0]+d[1] i = A0+A1 i */
	if (mclBnFp_setBigEndianMod(&out->x.d[0], raw + 32, 32) != 0) {
		return -1;
	}
	if (mclBnFp_setBigEndianMod(&out->x.d[1], raw, 32) != 0) {
		return -1;
	}
	if (mclBnFp_setBigEndianMod(&out->y.d[0], raw + 96, 32) != 0) {
		return -1;
	}
	if (mclBnFp_setBigEndianMod(&out->y.d[1], raw + 64, 32) != 0) {
		return -1;
	}
	mclBnFp_setInt32(&out->z.d[0], 1);
	mclBnFp_clear(&out->z.d[1]);
	return 0;
}

int groth16_bn254_verify(const uint8_t *proof, size_t plen,
                         const uint8_t *pub, size_t publen,
                         const uint8_t *vk, size_t vklen) {
	if (proof == NULL || pub == NULL || vk == NULL) {
		return -1;
	}
	if (ensure_init() != 0) {
		return -1;
	}
	if (plen != PROOF_SIZE || vklen < VK_HEADER || publen % FR_SIZE != 0) {
		return -1;
	}

	mclBnG1 ar, krs, alpha;
	mclBnG2 bs, beta, gamma, delta;
	if (g1_from_raw(&ar, proof) != 0) {
		return -1;
	}
	if (g2_from_raw(&bs, proof + G1_SIZE) != 0) {
		return -1;
	}
	if (g1_from_raw(&krs, proof + G1_SIZE + G2_SIZE) != 0) {
		return -1;
	}
	if (!mclBnG1_isValid(&ar) || !mclBnG1_isValid(&krs) || !mclBnG2_isValid(&bs)) {
		return -1;
	}

	const uint8_t *p = vk;
	if (g1_from_raw(&alpha, p) != 0) {
		return -1;
	}
	p += G1_SIZE;
	if (g2_from_raw(&beta, p) != 0) {
		return -1;
	}
	p += G2_SIZE;
	if (g2_from_raw(&gamma, p) != 0) {
		return -1;
	}
	p += G2_SIZE;
	if (g2_from_raw(&delta, p) != 0) {
		return -1;
	}
	p += G2_SIZE;
	uint32_t nk = read_u32_be(p);
	p += 4;
	if (nk < 1 || nk > MAX_PUBLIC + 1) {
		return -1;
	}
	if (publen != (size_t)(nk - 1) * FR_SIZE) {
		return -1;
	}
	if (vklen != VK_HEADER + (size_t)nk * G1_SIZE) {
		return -1;
	}

	mclBnG1 kpoints[MAX_PUBLIC + 1];
	for (uint32_t i = 0; i < nk; i++) {
		if (g1_from_raw(&kpoints[i], p) != 0) {
			return -1;
		}
		p += G1_SIZE;
	}

	mclBnG1 lpub;
	mclBnG1_clear(&lpub);
	mclBnG1_add(&lpub, &lpub, &kpoints[0]);
	for (uint32_t i = 1; i < nk; i++) {
		mclBnFr s;
		mclBnG1 scaled;
		if (mclBnFr_setBigEndianMod(&s, pub + (size_t)(i - 1) * FR_SIZE, FR_SIZE) != 0) {
			return -1;
		}
		mclBnG1_mul(&scaled, &kpoints[i], &s);
		mclBnG1_add(&lpub, &lpub, &scaled);
	}

	mclBnG2 delta_neg, gamma_neg, beta_neg;
	mclBnG2_neg(&delta_neg, &delta);
	mclBnG2_neg(&gamma_neg, &gamma);
	mclBnG2_neg(&beta_neg, &beta);

	mclBnG1 ps[4];
	mclBnG2 qs[4];
	ps[0] = ar;
	ps[1] = krs;
	ps[2] = lpub;
	ps[3] = alpha;
	qs[0] = bs;
	qs[1] = delta_neg;
	qs[2] = gamma_neg;
	qs[3] = beta_neg;

	mclBnGT f;
	mclBn_millerLoopVec(&f, ps, qs, 4);
	mclBn_finalExp(&f, &f);
	return mclBnGT_isOne(&f) ? 0 : -1;
}
