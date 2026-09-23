## Why

后量子算法不适合作为 Solidity 对照。需要先验证 EvoCrypt 能否把 Aigis-sig 编成符合现有 `execute` ABI 的 WASM，并走 cryptoupgrade 的上传、激活与调用路径。

## What Changes

- 以 Aigis-sig2 verify 为试点，用 WASI 编译独立 `.wasm`（不是 Emscripten `ENVIRONMENT=web` 产物）。
- 模块导出 `execute(ptr,len)->ptr`，输入为 ABI `(bytes pk, bytes message, bytes signature)`，输出为 ABI `bool` 的 length-prefixed 编码。
- 增加 in-process 冒烟：`uploadCodeVersion` 登记 → activation 编译实例化 → `callFunc` 验签。
- 使用 `AIGIS_SIG_MODE=2` 与 `aigis_sig2` 公开 API，避免现有 wrapper 把 MODE=2 接到 sig3 导致验签失败。

## Capabilities

### New Capabilities

- `aigis-wasm-evocrypt-smoke`: Aigis-sig2 WASM 制品、EvoCrypt 升级激活与 callFunc 验签冒烟。

### Modified Capabilities

- None.

## Impact

新增 `experiments/pqcc/wasi/` 构建脚本与 `cryptoupgrade` 集成测试。不改 EvoCrypt ABI，不注册 native precompile，不实现 Solidity 对照。

## 非目标

- 不覆盖 Dilithium/ML-DSA/SLH-DSA/KEM。
- 不把该 WASM 接到 0x47–0x5a 预编译路径。
- 不在本 change 跑多节点正式 Lab。
