## 上下文

`cryptoupgrade` 会将上传的算法源码编译为 Go plugin。候选文件因此需要是独立的 `package main` 文件，并暴露 plugin loader 可通过名称解析的导出入口。参考密码实现位于 `/home/liuqi/project/blockchain-crypto`，但这些文件不在本次修改范围内。

blockchain-crypto 仓库同时包含紧凑 primitives 和复杂系统。hash、AES、PBKDF2、Diffie-Hellman、Ed25519、Shamir sharing、Pedersen commitments 和 Schnorr proofs 等紧凑 primitives 可以用单文件纯 Go 候选实现表示。VDF runner、VRF、verifiable draw、shuffle、voting、BLS/KZG proof stack 和 oblivious transfer 等系统依赖外部可执行文件、CGO/Rust、生成的 protobuf 或大量曲线/协议包，因此不适合作为本轮单文件 plugin 候选。

## 目标 / 非目标

**目标：**

- 在 `cryptoupgrade/algorithm` 下增加独立候选源码文件。
- 新增候选仅使用 Go 标准库。
- 在可行时保持导出入口对 ABI 友好：`[]byte`、`bool` 和 `*big.Int`。
- 使用 cryptoupgrade 相同的 plugin 构建命令验证每个新源码文件。
- 记录被跳过的 blockchain-crypto 类别及未转换原因。

**非目标：**

- 不修改 `/home/liuqi/project/blockchain-crypto`。
- 不新增依赖。
- 不将大型多包实现移植到 cryptoupgrade 中。
- 不声称简化后的 benchmark 候选可以直接替代完整协议实现。

## 设计决策

- 将候选实现为独立文件，而不是 import blockchain-crypto 包。这样可以保持 plugin 构建自包含，并遵守不修改源仓库的约束。
- 当标准库实现能匹配候选类别时，优先使用标准库。例如：SHA-256、AES-CBC、Ed25519、PBKDF2 使用 HMAC-SHA256，DH/Pedersen/Schnorr/Shamir 使用 `math/big` 模运算。
- 对 Shamir shares、Schnorr proofs 等多值算法使用紧凑字节编码格式。这样可以让导出签名更简单，便于 ABI 编码和 plugin lookup。
- 当忠实的单文件实现需要复制大型依赖树或外部二进制时，将复杂协议类别标记为跳过。

候选映射：

- `rand`：`RandomBytes`
- `hash`：`Sha256`
- `encrypt`：`AesCBCEncrypt`、`AesCBCDecrypt`
- `signature`：`Ed25519Keygen`、`Ed25519PublicKey`、`Ed25519Sign`、`Ed25519Verify`
- `key_exchange`：`Dh2048Private`、`Dh2048Public`、`Dh2048Secret`
- `kdf`：`Pbkdf2Sha256`
- `share`：`ShamirSplit`、`ShamirRecover`
- `commit`：`PedersenCommit`、`PedersenVerify`
- `proof`：`SchnorrPublicKey`、`SchnorrProve`、`SchnorrVerify`

Solidity 对照映射：

- 已实现：`Sha256`、`Pbkdf2Sha256`、`Dh2048Public`、`Dh2048Secret`、`PedersenCommit`、`PedersenVerify`、`SchnorrPublicKey`、`SchnorrVerify`
- 已跳过：需要安全随机数的 `RandomBytes` 和私钥/证明生成例程；缺少 native Solidity/precompile 支持的 AES-CBC 和 Ed25519 例程；Go 候选使用 521-bit field 的 Shamir split/recover，因为紧凑 Solidity 支持需要更大的任意精度 field 库。

跳过类别：

- `vdf`：当前实现调用外部 VDF 可执行文件或 Rust-backed runner。
- `vrf`：实现依赖多文件 Edwards25519/secp256k1 VRF 栈。
- `protocols`：oblivious transfer 和 transmission evidence 依赖 Ristretto/BLS 协议包和多轮状态。
- `shuffle`、`verifiable_draw` 和 `vote`：实现依赖 BLS/KZG/Caulk+/DLEQ/Schnorr proof tree、SRS 对象、protobuf 或预加载投票流程。
- `types`、`utils` 和 `change`：属于支撑包，不是适合作为 plugin 候选的独立密码算法类别。

## 风险 / 权衡

- 简化候选是 benchmark 代表实现，不是原 blockchain-crypto 协议的完整替代。缓解方式：记录跳过类别，并保持文件名与类别映射明确。
- 部分候选使用便于 benchmark 的字节编码，而不是类型化 Go struct。缓解方式：导出函数在代码注释中记录确定性字节格式。
- 单文件候选会复制 RFC 3526 group prime 等小型常量。缓解方式：复制可以保持每个文件都能通过 `go build ... srcpath` 独立构建。
- 2048-bit 模运算的 Solidity 对照依赖 EVM `modexp` precompile 和紧凑字节数组算术。缓解方式：将这些合约定位为 benchmark 对照，并用 solc 做编译检查。
