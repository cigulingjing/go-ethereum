//go:build cgo

package groth16bn254

/*
#cgo CFLAGS: -O3 -std=gnu11
#cgo CFLAGS: -I${SRCDIR}/../../../algorithm/zkgo/groth16_bn254_verify/native/include
#cgo CFLAGS: -I${SRCDIR}/../../../algorithm/zkgo/groth16_bn254_verify/native
#cgo CXXFLAGS: -O3 -std=c++14 -DNDEBUG
#cgo CXXFLAGS: -DMCL_DONT_USE_XBYAK -DMCL_BINT_ASM=0 -DMCL_MSM=0
#cgo CXXFLAGS: -DMCL_FP_BIT=256 -DMCL_FR_BIT=256
#cgo CXXFLAGS: -I${SRCDIR}/../../../algorithm/zkgo/groth16_bn254_verify/native/include
#cgo CXXFLAGS: -I${SRCDIR}/../../../algorithm/zkgo/groth16_bn254_verify/native/src
#cgo LDFLAGS: -lstdc++
#include <stdint.h>
#include <stddef.h>
int groth16_bn254_verify(const uint8_t *proof, size_t plen, const uint8_t *pub, size_t publen, const uint8_t *vk, size_t vklen);
*/
import "C"

import (
	"errors"
	"unsafe"
)

const Name = "groth16_bn254_verify"

func Enabled() bool { return true }

func Verify(vk, public, proof []byte) (bool, error) {
	if len(vk) == 0 || len(public) == 0 || len(proof) == 0 {
		return false, errors.New("empty groth16 verify argument")
	}
	ret := C.groth16_bn254_verify(
		(*C.uint8_t)(unsafe.Pointer(&proof[0])),
		C.size_t(len(proof)),
		(*C.uint8_t)(unsafe.Pointer(&public[0])),
		C.size_t(len(public)),
		(*C.uint8_t)(unsafe.Pointer(&vk[0])),
		C.size_t(len(vk)),
	)
	return ret == 0, nil
}
