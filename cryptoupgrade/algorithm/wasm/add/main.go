package main

import (
	"encoding/binary"
	"unsafe"
)

const (
	addInputSize  = 64
	addOutputSize = 32
	addOffset     = 32
)

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := memoryBytes(inputPtr, inputLen)
	if len(input) < addInputSize {
		return 0
	}

	var left [addOutputSize]byte
	var right [addOutputSize]byte
	for i := 0; i < addOutputSize; i++ {
		left[i] = input[i]
		right[i] = input[addOutputSize+i]
	}

	var sum [addOutputSize]byte
	carry := uint16(0)
	for i := addOutputSize - 1; i >= 0; i-- {
		total := uint16(left[i]) + uint16(right[i]) + carry
		sum[i] = byte(total)
		carry = total >> 8
	}

	out := make([]byte, 4+addOutputSize)
	binary.LittleEndian.PutUint32(out[:4], addOutputSize)
	copy(out[4:], sum[:])
	return memoryPointer(&out[0])
}

func memoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func memoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func main() {}
