//go:build cgo

package pqcnative

/*
#cgo CFLAGS: -O3 -std=gnu11 -DAIGIS_SIG_MODE=2 -DUSE_SHAKE
#cgo CFLAGS: -I${SRCDIR}/../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native
#cgo CFLAGS: -I${SRCDIR}/../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/include
#cgo CFLAGS: -I${SRCDIR}/../../../experiments/cryptoupgrade/algorithm/pqcgo/aigis_sig2_verify/native/utils
#include "verify.h"
*/
import "C"

import "unsafe"

const (
	PublicKeyBytes = 1312
	SignatureBytes = 2445
)

func Enabled() bool {
	return true
}

func Verify(pk, message, signature []byte) (bool, error) {
	if len(pk) != PublicKeyBytes {
		return false, nil
	}
	if len(message) == 0 || len(signature) == 0 {
		return false, nil
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
