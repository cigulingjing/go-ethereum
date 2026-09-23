#include "blst.h"

#include <stdint.h>
#include <stddef.h>
#include <string.h>

/* 从 gnark Groth16 BLS12-381 Verify 抽出的无 commitment pairing 方程：
 *   e(Ar, Bs) · e(Krs, -δ) · e(L_pub, -γ) · e(α, -β) = 1
 * L_pub = K[0] + Σ public[i] · K[i+1]
 *
 * testdata 使用 ZCash uncompressed 点编码，与 blst deserialize 对齐。
 */
#define G1_SIZE 96
#define G2_SIZE 192
#define FR_SIZE 32
#define PROOF_SIZE (G1_SIZE + G2_SIZE + G1_SIZE)
#define VK_HEADER (G1_SIZE + G2_SIZE + G2_SIZE + G2_SIZE + 4)
#define MAX_PUBLIC 16

static uint32_t read_u32_be(const uint8_t *p) {
	return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | (uint32_t)p[3];
}

static void reverse32(uint8_t out[FR_SIZE], const uint8_t in[FR_SIZE]) {
	for (int i = 0; i < FR_SIZE; i++) {
		out[i] = in[FR_SIZE - 1 - i];
	}
}

static int g2_neg_affine(blst_p2_affine *out, const blst_p2_affine *in) {
	blst_p2 tmp;
	blst_p2_from_affine(&tmp, in);
	blst_p2_cneg(&tmp, 1);
	blst_p2_to_affine(out, &tmp);
	return 0;
}

/* proof || public || vk，返回 0 表示验证通过。参数顺序对齐 PQC harness。 */
int groth16_bls12381_verify(const uint8_t *proof, size_t plen,
                            const uint8_t *pub, size_t publen,
                            const uint8_t *vk, size_t vklen) {
	if (proof == NULL || pub == NULL || vk == NULL) {
		return -1;
	}
	if (plen != PROOF_SIZE) {
		return -1;
	}
	if (vklen < VK_HEADER) {
		return -1;
	}
	if (publen % FR_SIZE != 0) {
		return -1;
	}

	blst_p1_affine ar, krs, alpha;
	blst_p2_affine bs, beta, gamma, delta;
	if (blst_p1_deserialize(&ar, proof) != BLST_SUCCESS) {
		return -1;
	}
	if (blst_p2_deserialize(&bs, proof + G1_SIZE) != BLST_SUCCESS) {
		return -1;
	}
	if (blst_p1_deserialize(&krs, proof + G1_SIZE + G2_SIZE) != BLST_SUCCESS) {
		return -1;
	}
	if (!blst_p1_affine_in_g1(&ar) || !blst_p1_affine_in_g1(&krs) || !blst_p2_affine_in_g2(&bs)) {
		return -1;
	}

	const uint8_t *p = vk;
	if (blst_p1_deserialize(&alpha, p) != BLST_SUCCESS) {
		return -1;
	}
	p += G1_SIZE;
	if (blst_p2_deserialize(&beta, p) != BLST_SUCCESS) {
		return -1;
	}
	p += G2_SIZE;
	if (blst_p2_deserialize(&gamma, p) != BLST_SUCCESS) {
		return -1;
	}
	p += G2_SIZE;
	if (blst_p2_deserialize(&delta, p) != BLST_SUCCESS) {
		return -1;
	}
	p += G2_SIZE;
	uint32_t nk = read_u32_be(p);
	p += 4;
	if (nk < 1 || nk > MAX_PUBLIC + 1) {
		return -1;
	}
	size_t expected_pub = (size_t)(nk - 1) * FR_SIZE;
	if (publen != expected_pub) {
		return -1;
	}
	if (vklen != VK_HEADER + (size_t)nk * G1_SIZE) {
		return -1;
	}

	blst_p1_affine kpoints[MAX_PUBLIC + 1];
	for (uint32_t i = 0; i < nk; i++) {
		if (blst_p1_deserialize(&kpoints[i], p) != BLST_SUCCESS) {
			return -1;
		}
		p += G1_SIZE;
	}

	blst_p1 acc;
	blst_p1_from_affine(&acc, &kpoints[0]);
	for (uint32_t i = 1; i < nk; i++) {
		blst_p1 base, scaled;
		uint8_t scalar_le[FR_SIZE];
		blst_p1_from_affine(&base, &kpoints[i]);
		reverse32(scalar_le, pub + (size_t)(i - 1) * FR_SIZE);
		blst_p1_mult(&scaled, &base, scalar_le, 255);
		blst_p1_add(&acc, &acc, &scaled);
	}
	blst_p1_affine lpub;
	blst_p1_to_affine(&lpub, &acc);

	blst_p2_affine delta_neg, gamma_neg, beta_neg;
	g2_neg_affine(&delta_neg, &delta);
	g2_neg_affine(&gamma_neg, &gamma);
	g2_neg_affine(&beta_neg, &beta);

	const blst_p1_affine *ps[4] = {&ar, &krs, &lpub, &alpha};
	const blst_p2_affine *qs[4] = {&bs, &delta_neg, &gamma_neg, &beta_neg};
	blst_fp12 f;
	blst_miller_loop_n(&f, qs, ps, 4);
	blst_final_exp(&f, &f);
	return blst_fp12_is_one(&f) ? 0 : -1;
}
