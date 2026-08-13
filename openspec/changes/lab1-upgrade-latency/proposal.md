## Why

论文实验需要回答动态密码算法升级方案在多节点网络中的端到端升级耗时：从升级交易发起，到所有参与节点都完成本地算法激活并可调用为止。现有 `benchmark-upgrade-efficiency` 只覆盖单节点 `geth --dev` 的 Add-only 升级效率，无法反映交易出块、区块同步、事件处理和各节点本地 plugin 激活共同造成的多节点升级延迟。

## What Changes

- 新增多节点升级延迟实验入口，复用 `add-multi-node-network` 生成的私链拓扑和 `clique-sealer` 提供的 Clique 出块/同步能力。
- 以一次升级交易为实验起点，记录交易提交、receipt 确认、各节点看到升级区块、各节点完成本地激活、各节点验证调用成功的时间点。
- 输出每个节点的阶段耗时，以及从交易发起到所有节点完成升级操作的总耗时。
- 支持可复现实验配置，至少覆盖节点列表、RPC URL、升级算法、等待超时、重复次数、结果目录和是否复用已有网络。
- 保存机器可读 JSON/CSV 原始结果、升级交易 hash、区块号、gas used、节点级状态、失败原因和对应日志引用。
- 不改变 `CodeStorage.uploadCode`、`CodeStorage.callFunc`、动态插件加载语义或 Clique 共识语义。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 增加多节点动态升级端到端延迟实验要求，覆盖从升级交易发起到所有节点完成本地升级并验证可调用的节点级和全网级指标。

## Impact

- Affected code: `cryptoupgrade/bench/cmd/` 下新增或扩展实验命令；必要时复用或扩展 `cryptoupgrade/network` 的配置载入、RPC 节点发现和验证工具。
- Affected runtime: `add-multi-node-network` 渲染出的多节点 Clique 私链，节点需暴露 `eth`、`net` 和 cryptoupgrade 升级/调用所需 RPC。
- Affected artifacts: OpenSpec change、实验配置示例、原始 JSON/CSV 结果、节点日志路径或摘要。
- No breaking changes to Geth consensus, EVM execution semantics, dynamic upgrade ABI, existing single-node benchmarks, or precompiled/Solidity comparison experiments.
