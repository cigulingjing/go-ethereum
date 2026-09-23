#include <stddef.h>
#include <stdint.h>

#ifdef PQC_USE_API_H
#include "api.h"
#else
#include "sign.h"
#endif

#ifndef PQC_VERIFY_FN
#error "PQC_VERIFY_FN must be defined"
#endif

int PQC_VERIFY_FN(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk) {
	if (sig == NULL || m == NULL || pk == NULL) {
		return -1;
	}
#ifdef PQC_HAS_CTX
	return crypto_sign_verify(sig, slen, m, mlen, NULL, 0, pk);
#else
	return crypto_sign_verify(sig, slen, m, mlen, pk);
#endif
}
