//go:build cgo

package groth16bls12381

/*
#cgo CFLAGS: -O3 -std=gnu11 -D__BLST_NO_ASM__ -D__BLST_NO_CPUID__ -D__BLST_CGO__
#cgo CFLAGS: -I${SRCDIR}/../../algorithm/zkgo/groth16_bls12381_verify/native/include
#cgo CFLAGS: -I${SRCDIR}/../../algorithm/zkgo/groth16_bls12381_verify/native/src
#cgo CFLAGS: -I${SRCDIR}/../../algorithm/zkgo/groth16_bls12381_verify/native
#include <stdint.h>
#include <stddef.h>
int groth16_bls12381_verify(const uint8_t *proof, size_t plen, const uint8_t *pub, size_t publen, const uint8_t *vk, size_t vklen);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const Name = "groth16_bls12381_verify"

func Enabled() bool { return true }

func Verify(vk, public, proof []byte) (bool, error) {
	if len(vk) == 0 || len(public) == 0 || len(proof) == 0 {
		return false, errors.New("empty groth16 verify argument")
	}
	ret := C.groth16_bls12381_verify(
		(*C.uint8_t)(unsafe.Pointer(&proof[0])),
		C.size_t(len(proof)),
		(*C.uint8_t)(unsafe.Pointer(&public[0])),
		C.size_t(len(public)),
		(*C.uint8_t)(unsafe.Pointer(&vk[0])),
		C.size_t(len(vk)),
	)
	return ret == 0, nil
}
