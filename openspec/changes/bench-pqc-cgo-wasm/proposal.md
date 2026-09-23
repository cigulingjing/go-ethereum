## Why

后量子验签等算法不适合作为 Solidity 对照。现有 Lab2 要求三方案齐全，Aigis 单点对比也无法覆盖一组算法。需要单独测量 CGO 原生实现与 EvoCrypt WASM 的执行耗时，并归档可复现制品。

## What Changes

- 对 Solidity 不适合实现的算法，建立 CGO verify 与 WASM `callFunc` 的 in-process 对照实验。
- 每条结果记录算法名、CGO 耗时、WASM 耗时、原生实现文件大小、WASM 字节码长度。
- 按算法名把原生源码与 `.wasm` 备份到 `experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/`。

## Capabilities

### New Capabilities

- `pqc-cgo-wasm-benchmark`: Solidity 不适用算法的 CGO/WASM 执行耗时测量、指标落盘与制品备份。

### Modified Capabilities

- None.

## Impact

新增实验入口与 `pqcgo` 归档目录；复用已有 Aigis CGO/WASM 路径。不改 EvoCrypt ABI、共识规则或 Lab2 三方案语义。

## 非目标

- 不补 Solidity 对照，不跑多节点 RPC Lab2。
- 不把本次结果写成正式论文表。
- 不强制把每个算法注册为生产预编译地址。
