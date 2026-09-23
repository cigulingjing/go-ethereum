#ifndef AIGIS_SIG2_VERIFY_H
#define AIGIS_SIG2_VERIFY_H

#include <stddef.h>
#include <stdint.h>

/* 与 WASM execute 相同：空 ctx 的公开 crypto_sign_verify，0 表示验签成功。 */
int aigis_sig2_verify(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk);

#endif
