//go:build cgo

package slhdsa

/*
#cgo CFLAGS: -O3 -std=gnu11 -DUSE_SHAKE -DPQC_USE_API_H=1 -DPQC_VERIFY_FN=slh_dsa_shake_192f_verify -Drandombytes=slh_dsa_shake_192f_verify_randombytes
#cgo CFLAGS: -I${SRCDIR}/../../../algorithm/pqcgo/slh_dsa_shake_192f_verify/native
#cgo CFLAGS: -I${SRCDIR}/../../../algorithm/pqcgo/slh_dsa_shake_192f_verify/native/utils
#cgo CFLAGS: -I${SRCDIR}/../../../algorithm/pqcgo/slh_dsa_shake_192f_verify/native/hash/keccak
#include <stdint.h>
#include <stddef.h>
int slh_dsa_shake_192f_verify(const uint8_t *sig, size_t slen, const uint8_t *m, size_t mlen, const uint8_t *pk);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const Name = "slh_dsa_shake_192f_verify"

func Enabled() bool { return true }

func Verify(pk, message, signature []byte) (bool, error) {
	if len(pk) == 0 || len(message) == 0 || len(signature) == 0 {
		return false, errors.New("empty slh_dsa_shake_192f verify argument")
	}
	ret := C.slh_dsa_shake_192f_verify(
		(*C.uint8_t)(unsafe.Pointer(&signature[0])),
		C.size_t(len(signature)),
		(*C.uint8_t)(unsafe.Pointer(&message[0])),
		C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&pk[0])),
	)
	return ret == 0, nil
}
