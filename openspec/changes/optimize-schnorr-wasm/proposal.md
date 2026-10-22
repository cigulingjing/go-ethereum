## Why

`SchnorrVerify` 实验模块通过 `crypto/sha256` 计算挑战值，导致编译出的 WASM 附带通用密码学库代码，提高升级上传成本。

## What Changes

- 用算法源码中的 SHA-256 替换标准库导入，保留挑战值输入顺序及验签语义。
- 增加标准库对照、有效和无效证明及 WASM 执行测试，重建实验制品并记录体积。

## Capabilities

### New Capabilities

- `lightweight-schnorr-wasm`: 独立 Schnorr 验签 WASM 不依赖 `crypto` 标准库且输出兼容。

### Modified Capabilities

无。

## Impact

仅涉及 `experiments/cryptoupgrade/algorithm/go/schnorr_proof.go`、其 WASM 制品、测试和实验文档，不改变 Geth 预编译接口。

## 非目标

不修改 Schnorr 群参数、协议格式或 Gas 计价，不将实验性实现作为经过审计的生产密码库。
