//go:build cgo

package aigis

/*
#cgo CFLAGS: -O3 -std=gnu11 -DAIGIS_SIG_MODE=2 -DUSE_SHAKE -DPQC_HAS_CTX=1 -DPQC_VERIFY_FN=aigis_sig2_verify -Drandombytes=aigis_sig2_verify_randombytes
#cgo CFLAGS: -I${SRCDIR}/../../algorithm/pqcgo/aigis_sig2_verify/native
#cgo CFLAGS: -I${SRCDIR}/../../algorithm/pqcgo/aigis_sig2_verify/native/utils
#cgo CFLAGS: -I${SRCDIR}/../../algorithm/pqcgo/aigis_sig2_verify/native/hash/keccak
#include <stdint.h>
#include <stddef.h>
int aigis_sig2_verify(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const Name = "aigis_sig2_verify"

func Enabled() bool { return true }

func Verify(pk, message, signature []byte) (bool, error) {
	if len(pk) == 0 || len(message) == 0 || len(signature) == 0 {
		return false, errors.New("empty aigis verify argument")
	}
	ret := C.aigis_sig2_verify(
		(*C.uint8_t)(unsafe.Pointer(&signature[0])),
		C.size_t(len(signature)),
		(*C.uint8_t)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
	)
	return ret == 0, nil
}
