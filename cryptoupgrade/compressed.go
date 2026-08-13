package cryptoupgrade

import "github.com/ethereum/go-ethereum/cryptoupgrade/internal/sourcecodec"

// EncodeSource 将 Go 源码编码为 CodeStorage uploadCode 使用的 gzip/base64 文本。
func EncodeSource(source []byte) (string, error) {
	return sourcecodec.Encode(source)
}

// EncodeSourceFile 读取并编码 Go 源码文件，供实验工具通过稳定 facade 复用传输格式。
func EncodeSourceFile(filePath string) (string, error) {
	return sourcecodec.EncodeFile(filePath)
}
