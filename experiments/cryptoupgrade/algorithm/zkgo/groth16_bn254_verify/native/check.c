#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

int groth16_bn254_verify(const uint8_t *proof, size_t plen,
                         const uint8_t *pub, size_t publen,
                         const uint8_t *vk, size_t vklen);

static uint8_t *read_all(const char *path, size_t *n) {
	FILE *f = fopen(path, "rb");
	if (f == NULL) {
		return NULL;
	}
	if (fseek(f, 0, SEEK_END) != 0) {
		fclose(f);
		return NULL;
	}
	long sz = ftell(f);
	if (sz < 0) {
		fclose(f);
		return NULL;
	}
	if (fseek(f, 0, SEEK_SET) != 0) {
		fclose(f);
		return NULL;
	}
	uint8_t *buf = (uint8_t *)malloc((size_t)sz);
	if (buf == NULL) {
		fclose(f);
		return NULL;
	}
	if (sz > 0 && fread(buf, 1, (size_t)sz, f) != (size_t)sz) {
		free(buf);
		fclose(f);
		return NULL;
	}
	fclose(f);
	*n = (size_t)sz;
	return buf;
}

int main(int argc, char **argv) {
	if (argc != 4) {
		fprintf(stderr, "usage: check vk.bin public.bin proof.bin\n");
		return 2;
	}
	size_t vklen = 0, publen = 0, plen = 0;
	uint8_t *vk = read_all(argv[1], &vklen);
	uint8_t *pub = read_all(argv[2], &publen);
	uint8_t *proof = read_all(argv[3], &plen);
	if (vk == NULL || pub == NULL || proof == NULL) {
		fprintf(stderr, "failed to read testdata\n");
		return 1;
	}
	int rc = groth16_bn254_verify(proof, plen, pub, publen, vk, vklen);
	free(vk);
	free(pub);
	free(proof);
	if (rc != 0) {
		fprintf(stderr, "verify failed rc=%d\n", rc);
		return 1;
	}
	printf("ok\n");
	return 0;
}
