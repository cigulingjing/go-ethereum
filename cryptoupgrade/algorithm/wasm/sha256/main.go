package main

import (
	"crypto/sha256"
	"encoding/binary"
	"unsafe"
)

const (
	sha256InputHeadSize  = 64
	sha256OutputBytesLen = 32
	sha256OutputABISize  = 96
)

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := memoryBytes(inputPtr, inputLen)
	if len(input) < sha256InputHeadSize {
		return 0
	}

	offset := uint32(binary.BigEndian.Uint64(input[24:32]))
	if int(offset)+32 > len(input) {
		return 0
	}
	length := uint32(binary.BigEndian.Uint64(input[int(offset)+24 : int(offset)+32]))
	start := int(offset) + 32
	end := start + int(length)
	if end > len(input) {
		return 0
	}

	digest := sha256.Sum256(input[start:end])
	out := make([]byte, 4+sha256OutputABISize)
	binary.LittleEndian.PutUint32(out[:4], sha256OutputABISize)
	binary.BigEndian.PutUint64(out[4+24:4+32], sha256OutputBytesLen)
	binary.BigEndian.PutUint64(out[36+24:36+32], sha256OutputBytesLen)
	copy(out[68:], digest[:])
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
