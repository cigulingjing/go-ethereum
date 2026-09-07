package cryptoupgrade

import "github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmcodec"

// EncodeWasm 将 WASM bytecode 编码为 CodeStorage uploadCode 使用的 gzip/base64 文本。
func EncodeWasm(wasm []byte) (string, error) {
	return wasmcodec.Encode(wasm)
}

// EncodeWasmFile 读取并编码 WASM 文件，供实验工具通过稳定 facade 复用传输格式。
func EncodeWasmFile(filePath string) (string, error) {
	return wasmcodec.EncodeFile(filePath)
}

// EncodeSource 保留为兼容别名，实际编码对象已经切换为 WASM bytecode。
func EncodeSource(wasm []byte) (string, error) {
	return EncodeWasm(wasm)
}

// EncodeSourceFile 保留为兼容别名，实际读取对象已经切换为 WASM 文件。
func EncodeSourceFile(filePath string) (string, error) {
	return EncodeWasmFile(filePath)
}
