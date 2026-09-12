## Why

20 节点 Lab1 第一组数据需要同时比较本文升级方案和 Solidity 合约方案在不同密码算法下的升级耗时与 Gas 消耗。现有结果只完整覆盖本文方案，Solidity 合约仅有单节点调用效率或旧的少量部署记录，不能支撑论文表格。

## What Changes

- 新增 Solidity 合约多节点部署/升级延迟实验入口，复用现有多节点网络 YAML、sender/target 节点选择和 RPC 前置检查。
- 对 Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit、SchnorrVerify 逐一编译并部署等价 Solidity 合约。
- 记录部署交易提交、tx hash 返回、sender receipt、各目标节点 receipt 可见、各目标节点校验调用成功的时间点。
- 输出 round-level 和 node-level JSON/CSV，字段对齐现有 Lab1 WASM `rounds.csv`/`nodes.csv`，便于直接合并出第一组对照表。
- 不修改 Geth、`CodeStorage`、precompile 或现有 WASM 升级实验语义。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 增加多节点 Solidity 合约部署/升级延迟实验要求，用于补齐 20 节点 Lab1 第一组对照数据。

## Non-Goals

- 不重新设计 Solidity 算法实现，不引入代理合约或可升级合约框架。
- 不把 Solidity 部署耗时混入 Lab2 已部署调用效率指标。
- 不用历史硬编码部署数据替代新实验原始数据。

## Impact

- Affected code: `experiments/cryptoupgrade/bench/cmd/` 下新增实验命令。
- Affected artifacts: 新增 OpenSpec change、Solidity 部署延迟 JSON/CSV 原始结果。
- Affected runtime: 已启动的本地 20 节点 Clique 网络，需要所有目标节点 HTTP RPC 可达，sender 账户可通过 `eth_sendTransaction` 发送部署交易。
