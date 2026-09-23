//go:build cgo

package dilithium3

/*
#cgo CFLAGS: -O3 -std=gnu11 -DDILITHIUM_MODE=3 -DUSE_SHAKE -DPQC_VERIFY_FN=dilithium3_verify
#cgo CFLAGS: -DPQC_NAMESPACE=dilithium3_verify -Drandombytes=dilithium3_verify_randombytes
#cgo CFLAGS: -I${SRCDIR}/native -I${SRCDIR}/../common -I${SRCDIR}/../common/utils -I${SRCDIR}/../common/hash/keccak
#include <stdint.h>
#include <stddef.h>
int dilithium3_verify(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const Name = "dilithium3_verify"

func Verify(pk, message, signature []byte) (bool, error) {
	if len(pk) == 0 || len(message) == 0 || len(signature) == 0 {
		return false, errors.New("empty verify argument")
	}
	ret := C.dilithium3_verify(
		(*C.uint8_t)(unsafe.Pointer(&signature[0])),
		C.size_t(len(signature)),
		(*C.uint8_t)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
	)
	return ret == 0, nil
}
