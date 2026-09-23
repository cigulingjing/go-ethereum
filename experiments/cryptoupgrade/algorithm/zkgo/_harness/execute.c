#include <stdint.h>
#include <stddef.h>
#include <string.h>

#ifndef GROTH16_VERIFY_FN
#define GROTH16_VERIFY_FN groth16_bls12381_verify
#endif

int GROTH16_VERIFY_FN(const uint8_t *proof, size_t plen,
                      const uint8_t *pub, size_t publen,
                      const uint8_t *vk, size_t vklen);

static uint8_t g_output[4 + 32];

static int read_bytes_arg(const uint8_t *input, uint32_t input_len, int index,
                          const uint8_t **data, uint32_t *data_len) {
	const uint32_t head = (uint32_t)index * 32u;
	if (head + 32u > input_len) {
		return -1;
	}
	for (uint32_t i = 0; i < 24u; i++) {
		if (input[head + i] != 0) {
			return -1;
		}
	}
	uint32_t offset = 0;
	for (uint32_t i = 24; i < 32u; i++) {
		offset = (offset << 8) | input[head + i];
	}
	if (offset + 32u > input_len) {
		return -1;
	}
	for (uint32_t i = 0; i < 24u; i++) {
		if (input[offset + i] != 0) {
			return -1;
		}
	}
	uint32_t length = 0;
	for (uint32_t i = 24; i < 32u; i++) {
		length = (length << 8) | input[offset + i];
	}
	uint32_t start = offset + 32u;
	if (start + length > input_len) {
		return -1;
	}
	*data = input + start;
	*data_len = length;
	return 0;
}

static uint32_t write_bool(int ok) {
	memset(g_output, 0, sizeof(g_output));
	g_output[0] = 32;
	if (ok) {
		g_output[4 + 31] = 1;
	}
	return (uint32_t)(uintptr_t)g_output;
}

/* ABI: (bytes vk, bytes public, bytes proof) → bool，与实验入口的三段 bytes 打包一致。 */
__attribute__((export_name("execute")))
uint32_t execute(uint32_t input_ptr, uint32_t input_len) {
	const uint8_t *input = (const uint8_t *)(uintptr_t)input_ptr;
	const uint8_t *vk = NULL;
	const uint8_t *pub = NULL;
	const uint8_t *proof = NULL;
	uint32_t vk_len = 0;
	uint32_t pub_len = 0;
	uint32_t proof_len = 0;

	if (input == NULL ||
	    read_bytes_arg(input, input_len, 0, &vk, &vk_len) != 0 ||
	    read_bytes_arg(input, input_len, 1, &pub, &pub_len) != 0 ||
	    read_bytes_arg(input, input_len, 2, &proof, &proof_len) != 0) {
		return write_bool(0);
	}
	return write_bool(GROTH16_VERIFY_FN(proof, proof_len, pub, pub_len, vk, vk_len) == 0);
}
