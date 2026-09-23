#include "verify.h"

#include "sign.h"

int aigis_sig2_verify(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk) {
	if (sig == NULL || m == NULL || pk == NULL) {
		return -1;
	}
	return crypto_sign_verify(sig, slen, m, mlen, NULL, 0, pk);
}
