## Why

`PedersenCommit`、`Pbkdf2Sha256` 和 `Ed25519Verify` 的实验源码导入 Go `crypto` 标准库，导致 TinyGo WASM 附带不必要的依赖，增大升级交易负担。

## What Changes

- 用源码实现三种算法所需的 SHA-256、HMAC-SHA256 和 Ed25519 验签计算，保持公开接口与输出兼容。
- 补充标准库对照及异常输入测试，重新编译已提供的 WASM 制品。
- 为 Ed25519 验签补全现有构建器的三 `bytes` 参数、`bool` 返回 ABI。

## Capabilities

### New Capabilities

- `crypto-free-experiment-wasm`: 三种实验算法可在不导入 `crypto` 包时正确计算、构建和调用。

### Modified Capabilities

无。

## Impact

仅影响实验 Go 源码、WASM 制品、必要的 ABI 构建包装器及测试；不修改 Geth 预编译实现和计价规则。

## 非目标

不宣称实验性独立实现已经过密码学安全审计，也不替换链上历史版本。
