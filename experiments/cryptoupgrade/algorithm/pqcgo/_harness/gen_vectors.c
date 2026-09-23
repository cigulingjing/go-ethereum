#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#ifdef PQC_USE_API_H
#include "api.h"
#else
#include "sign.h"
#endif
#include "randombytes.h"

#ifndef PQC_ALG_NAME
#define PQC_ALG_NAME pqc
#endif
#define PQC_ALG_STR_HELPER(s) #s
#define PQC_ALG_STR(s) PQC_ALG_STR_HELPER(s)

static int write_all(const char *path, const uint8_t *data, size_t len) {
	FILE *fp = fopen(path, "wb");
	if (fp == NULL) {
		perror(path);
		return -1;
	}
	if (fwrite(data, 1, len, fp) != len) {
		fclose(fp);
		return -1;
	}
	fclose(fp);
	return 0;
}

int main(int argc, char **argv) {
	const char *outdir = argc > 1 ? argv[1] : ".";
	uint8_t pk[CRYPTO_PUBLICKEYBYTES];
	uint8_t sk[CRYPTO_SECRETKEYBYTES];
	uint8_t sig[CRYPTO_BYTES];
	uint8_t message[32];
	size_t siglen = 0;
	char path[512];

	randombytes(message, sizeof(message));
	if (crypto_sign_keypair(pk, sk) != 0) {
		fprintf(stderr, "%s keypair failed\n", PQC_ALG_STR(PQC_ALG_NAME));
		return 1;
	}
#ifdef PQC_HAS_CTX
	if (crypto_sign_signature(sig, &siglen, message, sizeof(message), NULL, 0, sk) != 0) {
		fprintf(stderr, "%s sign failed\n", PQC_ALG_STR(PQC_ALG_NAME));
		return 1;
	}
	if (crypto_sign_verify(sig, siglen, message, sizeof(message), NULL, 0, pk) != 0) {
		fprintf(stderr, "%s self-verify failed\n", PQC_ALG_STR(PQC_ALG_NAME));
		return 1;
	}
#else
	if (crypto_sign_signature(sig, &siglen, message, sizeof(message), sk) != 0) {
		fprintf(stderr, "%s sign failed\n", PQC_ALG_STR(PQC_ALG_NAME));
		return 1;
	}
	if (crypto_sign_verify(sig, siglen, message, sizeof(message), pk) != 0) {
		fprintf(stderr, "%s self-verify failed\n", PQC_ALG_STR(PQC_ALG_NAME));
		return 1;
	}
#endif

	snprintf(path, sizeof(path), "%s/pk.bin", outdir);
	if (write_all(path, pk, sizeof(pk)) != 0) {
		return 1;
	}
	snprintf(path, sizeof(path), "%s/message.bin", outdir);
	if (write_all(path, message, sizeof(message)) != 0) {
		return 1;
	}
	snprintf(path, sizeof(path), "%s/signature.bin", outdir);
	if (write_all(path, sig, siglen) != 0) {
		return 1;
	}
	printf("wrote %s vectors pk=%zu sig=%zu msg=%zu\n", PQC_ALG_STR(PQC_ALG_NAME), sizeof(pk), siglen, sizeof(message));
	return 0;
}
