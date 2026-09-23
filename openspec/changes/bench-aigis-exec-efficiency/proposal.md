## Why

Aigis-sig2 已能走 EvoCrypt WASM 上传与 `callFunc`，但还没有同一算法的 native 预编译对照，无法量化 WASM 调用相对预编译合约的执行开销。

## What Changes

- 用 CGO 链接与 WASM 相同的 Aigis-sig2 verify 源码，注册 native precompile（0x5b）。
- 增加 in-process 效率对比：EvoCrypt `callFunc` vs `Precompile.Run`，同一组 pk/message/signature。
- 输出 JSON 与文本摘要到 `experiments/cryptoupgrade/results/`。

## Capabilities

### New Capabilities

- `aigis-exec-efficiency`: Aigis-sig2 WASM 与预编译调用延迟对比及结果落盘。

### Modified Capabilities

- `cryptoupgrade-precompiled-algorithms`: 允许 CGO 实现确定性 Aigis-sig2 verify 预编译，禁止走 WASM。

## Impact

影响 `cryptoupgrade/internal/pqcnative`、`precompile.go`、`common/cryptoupgrade_contract.go` 与 cryptoupgrade 测试。Geth 构建该入口需要 `CGO_ENABLED=1`。不改 EvoCrypt ABI 与共识规则。

## 非目标

- 不覆盖 Dilithium/ML-DSA/SLH-DSA，不接 Solidity 对照。
- 不跑多节点 Lab2 RPC，不把本结果写成正式论文表。
