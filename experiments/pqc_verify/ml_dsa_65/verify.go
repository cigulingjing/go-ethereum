//go:build cgo

package mldsa65

/*
#cgo CFLAGS: -O3 -std=gnu11 -DML_DSA_MODE=65 -DUSE_SHAKE -DPQC_HAS_CTX=1 -DPQC_VERIFY_FN=ml_dsa_65_verify
#cgo CFLAGS: -DPQC_NAMESPACE=ml_dsa_65_verify -Drandombytes=ml_dsa_65_verify_randombytes
#cgo CFLAGS: -I${SRCDIR}/native -I${SRCDIR}/../common -I${SRCDIR}/../common/utils -I${SRCDIR}/../common/hash/keccak
#include <stdint.h>
#include <stddef.h>
int ml_dsa_65_verify(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const Name = "ml_dsa_65_verify"

func Verify(pk, message, signature []byte) (bool, error) {
	if len(pk) == 0 || len(message) == 0 || len(signature) == 0 {
		return false, errors.New("empty verify argument")
	}
	ret := C.ml_dsa_65_verify(
		(*C.uint8_t)(unsafe.Pointer(&signature[0])),
		C.size_t(len(signature)),
		(*C.uint8_t)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
	)
	return ret == 0, nil
}
