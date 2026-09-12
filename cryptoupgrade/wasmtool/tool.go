package wasmtool

import (
	"context"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
)

// Spec 描述将 Go 算法源码包装成 TinyGo WASM 模块所需的 ABI 形状。
type Spec struct {
	Function    string
	InputTypes  []string
	OutputTypes []string
}

// EncodePath 读取 wasm 或 Go 源码文件，必要时调用 TinyGo 编译后再做 upload 编码。
func EncodePath(ctx context.Context, path string, spec Spec) (string, error) {
	_, encoded, err := BuildEncodedPath(ctx, path, spec)
	return encoded, err
}

// BuildEncodedPath returns the raw WASM module and its upload encoding.
func BuildEncodedPath(ctx context.Context, path string, spec Spec) ([]byte, string, error) {
	wasm, err := BuildPath(ctx, path, spec)
	if err != nil {
		return nil, "", err
	}
	encoded, err := cryptoupgrade.EncodeWasm(wasm)
	if err != nil {
		return nil, "", err
	}
	return wasm, encoded, nil
}

// EncodeSource 将内联 Go 源码编译为 WASM 并返回 upload 编码。
func EncodeSource(ctx context.Context, source []byte, spec Spec) (string, error) {
	_, encoded, err := BuildEncodedSource(ctx, source, spec)
	return encoded, err
}

// BuildEncodedSource compiles Go source to WASM and returns the upload encoding.
func BuildEncodedSource(ctx context.Context, source []byte, spec Spec) ([]byte, string, error) {
	wasm, err := Build(ctx, source, spec)
	if err != nil {
		return nil, "", err
	}
	encoded, err := cryptoupgrade.EncodeWasm(wasm)
	if err != nil {
		return nil, "", err
	}
	return wasm, encoded, nil
}

// WasmHash returns the module identity recorded by cryptoupgrade metadata.
func WasmHash(wasm []byte) string {
	return crypto.Keccak256Hash(wasm).Hex()
}

// BuildPath 读取路径指向的模块；.wasm 直接返回，.go 则经 TinyGo 编译为 wasm。
func BuildPath(ctx context.Context, path string, spec Spec) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("source path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source %s: %w", path, err)
	}
	if strings.EqualFold(filepath.Ext(path), ".wasm") {
		return append([]byte(nil), data...), nil
	}
	return Build(ctx, data, spec)
}

// Build 将 Go 源码和 execute wrapper 组合后经 TinyGo 编译为 wasm。
func Build(ctx context.Context, source []byte, spec Spec) ([]byte, error) {
	if len(source) == 0 {
		return nil, fmt.Errorf("source bytes are empty")
	}
	if strings.TrimSpace(spec.Function) == "" {
		return nil, fmt.Errorf("module function name is empty")
	}

	wrapper, err := wrapperSource(spec)
	if err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp("", "cryptoupgrade-wasm-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary wasm build dir: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "module.go"), source, 0o644); err != nil {
		return nil, fmt.Errorf("write Go source: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), wrapper, 0o644); err != nil {
		return nil, fmt.Errorf("write wasm wrapper: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module cryptoupgrade-wasmtool\n\ngo 1.25\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write wasm build module file: %w", err)
	}

	outputPath := filepath.Join(dir, "module.wasm")
	tinygoBin := os.Getenv("TINYGO")
	if tinygoBin == "" {
		tinygoBin = "tinygo"
	}
	cmd := exec.CommandContext(ctx, tinygoBin, "build",
		"-target", "wasi",
		"-buildmode=c-shared",
		"-scheduler=none",
		"-gc=leaking",
		"-no-debug",
		"-o", outputPath,
		".",
	)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tinygo build %s: %w\n%s", spec.Function, err, string(output))
	}
	wasm, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read built wasm: %w", err)
	}
	if len(wasm) == 0 {
		return nil, fmt.Errorf("built wasm artifact is empty")
	}
	return wasm, nil
}

func wrapperSource(spec Spec) ([]byte, error) {
	inputTypes := normalizeTypes(spec.InputTypes)
	outputTypes := normalizeTypes(spec.OutputTypes)
	switch signatureKey(inputTypes, outputTypes) {
	case signatureKey([]string{"int256", "int256"}, []string{"int256"}):
		return formatWrapper(fmt.Sprintf(addWrapperTemplate, spec.Function))
	case signatureKey([]string{"uint256", "uint256"}, []string{"uint256"}):
		return formatWrapper(fmt.Sprintf(addWrapperTemplate, spec.Function))
	case signatureKey([]string{"bytes"}, []string{"bytes"}):
		return formatWrapper(fmt.Sprintf(bytesWrapperTemplate, spec.Function))
	case signatureKey([]string{"bytes"}, []string{"bytes32"}):
		return formatWrapper(fmt.Sprintf(bytes32WrapperTemplate, spec.Function))
	case signatureKey([]string{"bytes", "bytes"}, []string{"bytes"}):
		return formatWrapper(fmt.Sprintf(bytesBytesWrapperTemplate, spec.Function))
	case signatureKey([]string{"bytes", "bytes"}, []string{"bool"}):
		return formatWrapper(fmt.Sprintf(bytesBoolWrapperTemplate, spec.Function))
	case signatureKey([]string{"bytes", "bytes", "uint256", "uint256"}, []string{"bytes"}):
		return formatWrapper(fmt.Sprintf(bytesBytesUintWrapperTemplate, spec.Function))
	case signatureKey([]string{"uint256[]", "uint256[]", "uint256"}, []string{"uint256[]"}):
		return formatWrapper(uint256ArrayPolynomialWrapperTemplate)
	default:
		return nil, fmt.Errorf("unsupported wasm signature %q -> %q", spec.InputTypes, spec.OutputTypes)
	}
}

func formatWrapper(source string) ([]byte, error) {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return nil, fmt.Errorf("format wrapper source: %w", err)
	}
	return formatted, nil
}

func normalizeTypes(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, typ := range raw {
		typ = strings.TrimSpace(strings.ToLower(typ))
		if typ != "" {
			out = append(out, typ)
		}
	}
	return out
}

func signatureKey(inputTypes, outputTypes []string) string {
	return strings.Join(inputTypes, ",") + "->" + strings.Join(outputTypes, ",")
}

const addWrapperTemplate = `package main

import (
	"encoding/binary"
	"math/big"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	if len(input) < 64 {
		return 0
	}
	left := wasmDecodeInt256(input[:32])
	right := wasmDecodeInt256(input[32:64])
	if left == nil || right == nil {
		return 0
	}
	result := %s(left, right)
	if result == nil {
		return 0
	}
	out := wasmEncodeLengthPrefixed(wasmEncodeInt256(result))
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmDecodeInt256(slot []byte) *big.Int {
	if len(slot) != wasmWordSize {
		return nil
	}
	value := new(big.Int).SetBytes(slot)
	if slot[0]&0x80 != 0 {
		limit := new(big.Int).Lsh(big.NewInt(1), 256)
		value.Sub(value, limit)
	}
	return value
}

func wasmEncodeInt256(value *big.Int) []byte {
	if value == nil {
		return nil
	}
	limit := new(big.Int).Lsh(big.NewInt(1), 256)
	encoded := new(big.Int).Set(value)
	if encoded.Sign() < 0 {
		encoded.Add(encoded, limit)
	}
	return wasmLeftPad32(encoded.Bytes())
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func wasmLeftPad32(raw []byte) []byte {
	if len(raw) >= wasmWordSize {
		return append([]byte(nil), raw[len(raw)-wasmWordSize:]...)
	}
	out := make([]byte, wasmWordSize)
	copy(out[wasmWordSize-len(raw):], raw)
	return out
}

func main() {}
`

const bytesWrapperTemplate = `package main

import (
	"encoding/binary"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	data := wasmReadBytesArg(input, 0)
	if data == nil {
		return 0
	}
	result := %s(data)
	if result == nil {
		return 0
	}
	out := wasmEncodeLengthPrefixed(wasmEncodeABIBytes(result))
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmReadBytesArg(input []byte, index int) []byte {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	offset := int(binary.BigEndian.Uint64(slot[24:32]))
	if offset+32 > len(input) {
		return nil
	}
	length := int(binary.BigEndian.Uint64(input[offset+24 : offset+32]))
	start := offset + 32
	end := start + length
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmReadSlot(input []byte, index int) []byte {
	start := index * wasmWordSize
	end := start + wasmWordSize
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmEncodeABIBytes(data []byte) []byte {
	payloadLen := wasmAlign32(len(data))
	out := make([]byte, 64+payloadLen)
	binary.BigEndian.PutUint64(out[24:32], 32)
	binary.BigEndian.PutUint64(out[56:64], uint64(len(data)))
	copy(out[64:], data)
	return out
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func wasmAlign32(length int) int {
	if length == 0 {
		return 0
	}
	return ((length + wasmWordSize - 1) / wasmWordSize) * wasmWordSize
}

func main() {}
`

const bytes32WrapperTemplate = `package main

import (
	"encoding/binary"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	data := wasmReadBytesArg(input, 0)
	if data == nil {
		return 0
	}
	result := %s(data)
	out := wasmEncodeLengthPrefixed(result[:])
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmReadBytesArg(input []byte, index int) []byte {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	offset := int(binary.BigEndian.Uint64(slot[24:32]))
	if offset+32 > len(input) {
		return nil
	}
	length := int(binary.BigEndian.Uint64(input[offset+24 : offset+32]))
	start := offset + 32
	end := start + length
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmReadSlot(input []byte, index int) []byte {
	start := index * wasmWordSize
	end := start + wasmWordSize
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func main() {}
`

const bytesBytesWrapperTemplate = `package main

import (
	"encoding/binary"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	left := wasmReadBytesArg(input, 0)
	right := wasmReadBytesArg(input, 1)
	if left == nil || right == nil {
		return 0
	}
	result := %s(left, right)
	if result == nil {
		return 0
	}
	out := wasmEncodeLengthPrefixed(wasmEncodeABIBytes(result))
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmReadBytesArg(input []byte, index int) []byte {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	offset := int(binary.BigEndian.Uint64(slot[24:32]))
	if offset+32 > len(input) {
		return nil
	}
	length := int(binary.BigEndian.Uint64(input[offset+24 : offset+32]))
	start := offset + 32
	end := start + length
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmReadSlot(input []byte, index int) []byte {
	start := index * wasmWordSize
	end := start + wasmWordSize
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmEncodeABIBytes(data []byte) []byte {
	payloadLen := wasmAlign32(len(data))
	out := make([]byte, 64+payloadLen)
	binary.BigEndian.PutUint64(out[24:32], 32)
	binary.BigEndian.PutUint64(out[56:64], uint64(len(data)))
	copy(out[64:], data)
	return out
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func wasmAlign32(length int) int {
	if length == 0 {
		return 0
	}
	return ((length + wasmWordSize - 1) / wasmWordSize) * wasmWordSize
}

func main() {}
`

const bytesBoolWrapperTemplate = `package main

import (
	"encoding/binary"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	left := wasmReadBytesArg(input, 0)
	right := wasmReadBytesArg(input, 1)
	if left == nil || right == nil {
		return 0
	}
	result := %s(left, right)
	out := wasmEncodeLengthPrefixed(wasmEncodeBool(result))
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmReadBytesArg(input []byte, index int) []byte {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	offset := int(binary.BigEndian.Uint64(slot[24:32]))
	if offset+32 > len(input) {
		return nil
	}
	length := int(binary.BigEndian.Uint64(input[offset+24 : offset+32]))
	start := offset + 32
	end := start + length
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmReadSlot(input []byte, index int) []byte {
	start := index * wasmWordSize
	end := start + wasmWordSize
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmEncodeBool(value bool) []byte {
	out := make([]byte, wasmWordSize)
	if value {
		out[wasmWordSize-1] = 1
	}
	return out
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func main() {}
`

const bytesBytesUintWrapperTemplate = `package main

import (
	"encoding/binary"
	"math/big"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	left := wasmReadBytesArg(input, 0)
	right := wasmReadBytesArg(input, 1)
	if left == nil || right == nil {
		return 0
	}
	iterations := wasmReadUint256Arg(input, 2)
	keyLength := wasmReadUint256Arg(input, 3)
	if iterations == nil || keyLength == nil {
		return 0
	}
	result := %s(left, right, iterations, keyLength)
	if result == nil {
		return 0
	}
	out := wasmEncodeLengthPrefixed(wasmEncodeABIBytes(result))
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmReadBytesArg(input []byte, index int) []byte {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	offset := int(binary.BigEndian.Uint64(slot[24:32]))
	if offset+32 > len(input) {
		return nil
	}
	length := int(binary.BigEndian.Uint64(input[offset+24 : offset+32]))
	start := offset + 32
	end := start + length
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmReadUint256Arg(input []byte, index int) *big.Int {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	return new(big.Int).SetBytes(slot)
}

func wasmReadSlot(input []byte, index int) []byte {
	start := index * wasmWordSize
	end := start + wasmWordSize
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmEncodeABIBytes(data []byte) []byte {
	payloadLen := wasmAlign32(len(data))
	out := make([]byte, 64+payloadLen)
	binary.BigEndian.PutUint64(out[24:32], 32)
	binary.BigEndian.PutUint64(out[56:64], uint64(len(data)))
	copy(out[64:], data)
	return out
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func wasmAlign32(length int) int {
	if length == 0 {
		return 0
	}
	return ((length + wasmWordSize - 1) / wasmWordSize) * wasmWordSize
}

func main() {}
`

const uint256ArrayPolynomialWrapperTemplate = `package main

import (
	"encoding/binary"
	"math/big"
	"unsafe"
)

const wasmWordSize = 32

//go:wasmexport execute
func execute(inputPtr, inputLen uint32) uint32 {
	input := wasmMemoryBytes(inputPtr, inputLen)
	left := wasmReadUint256ArrayArg(input, 0)
	right := wasmReadUint256ArrayArg(input, 1)
	modulus := wasmReadUint256Arg(input, 2)
	if len(left) == 0 || len(right) == 0 || modulus == nil || modulus.Sign() <= 0 {
		return 0
	}
	result := PolynomialMul(left, right, modulus)
	if len(result) == 0 {
		return 0
	}
	encoded := wasmEncodeABIUint256Array(result)
	out := wasmEncodeLengthPrefixed(encoded)
	return wasmMemoryPointer(&out[0])
}

func wasmMemoryBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

func wasmMemoryPointer(ptr *byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(ptr)))
}

func wasmReadUint256ArrayArg(input []byte, index int) []*big.Int {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	offset := int(binary.BigEndian.Uint64(slot[24:32]))
	if offset+32 > len(input) {
		return nil
	}
	length := int(binary.BigEndian.Uint64(input[offset+24 : offset+32]))
	start := offset + 32
	end := start + length*wasmWordSize
	if end > len(input) {
		return nil
	}
	out := make([]*big.Int, length)
	for i := 0; i < length; i++ {
		elem := input[start+i*wasmWordSize : start+(i+1)*wasmWordSize]
		out[i] = new(big.Int).SetBytes(elem)
	}
	return out
}

func wasmReadUint256Arg(input []byte, index int) *big.Int {
	slot := wasmReadSlot(input, index)
	if slot == nil {
		return nil
	}
	return new(big.Int).SetBytes(slot)
}

func wasmReadSlot(input []byte, index int) []byte {
	start := index * wasmWordSize
	end := start + wasmWordSize
	if end > len(input) {
		return nil
	}
	return input[start:end]
}

func wasmEncodeABIUint256Array(values []*big.Int) []byte {
	out := make([]byte, 64+len(values)*wasmWordSize)
	binary.BigEndian.PutUint64(out[24:32], 32)
	binary.BigEndian.PutUint64(out[56:64], uint64(len(values)))
	for i, value := range values {
		if value == nil {
			continue
		}
		copy(out[64+i*wasmWordSize:64+(i+1)*wasmWordSize], wasmEncodeUint256(value))
	}
	return out
}

func wasmEncodeUint256(value *big.Int) []byte {
	out := make([]byte, wasmWordSize)
	if value == nil {
		return out
	}
	value.FillBytes(out)
	return out
}

func wasmEncodeLengthPrefixed(payload []byte) []byte {
	if payload == nil {
		payload = []byte{}
	}
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(payload)))
	copy(out[4:], payload)
	return out
}

func main() {}
`
