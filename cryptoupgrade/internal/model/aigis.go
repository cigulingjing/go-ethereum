package model

import _ "embed"

// 制品与向量放在 model 包内，测试和构建脚本共用同一份文件，不必再从 cryptoupgrade 根目录拼路径。

//go:embed aigis_sig2/aigis_sig2_verify.wasm
var AigisSig2VerifyWASM []byte

//go:embed aigis_sig2/pk.bin
var AigisSig2PublicKey []byte

//go:embed aigis_sig2/message.bin
var AigisSig2Message []byte

//go:embed aigis_sig2/signature.bin
var AigisSig2Signature []byte
