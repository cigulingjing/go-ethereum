#include <stdint.h>
#include <stddef.h>
#include <string.h>

#ifdef PQC_USE_API_H
#include "api.h"
#else
#include "sign.h"
#endif

#ifndef PQC_VERIFY_FN
#error "PQC_VERIFY_FN must be defined"
#endif

int PQC_VERIFY_FN(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk);

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

__attribute__((export_name("execute")))
uint32_t execute(uint32_t input_ptr, uint32_t input_len) {
	const uint8_t *input = (const uint8_t *)(uintptr_t)input_ptr;
	const uint8_t *pk = NULL;
	const uint8_t *message = NULL;
	const uint8_t *signature = NULL;
	uint32_t pk_len = 0;
	uint32_t message_len = 0;
	uint32_t signature_len = 0;

	if (input == NULL ||
	    read_bytes_arg(input, input_len, 0, &pk, &pk_len) != 0 ||
	    read_bytes_arg(input, input_len, 1, &message, &message_len) != 0 ||
	    read_bytes_arg(input, input_len, 2, &signature, &signature_len) != 0) {
		return write_bool(0);
	}
	if (pk_len != CRYPTO_PUBLICKEYBYTES) {
		return write_bool(0);
	}
	return write_bool(PQC_VERIFY_FN(signature, signature_len, message, message_len, pk) == 0);
}
