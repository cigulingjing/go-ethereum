## Why

现有实验分别记录 Solidity 合约部署和 WASM 动态升级，但缺少统一的 Lab 1 采集入口，无法在相同计算逻辑、相同输入和相同节点规模下直接比较两种方案的端到端升级延迟。需要把 1、2、5、10 节点场景和重复轮次整理成同一份可绘图 CSV。

## What Changes

- 新增 Lab 1 对比实验入口，按 1、2、5、10 个节点分别创建隔离的私有 Clique 网络。
- 固定同一个密码学 fixture 和输入，分别运行 Solidity 合约部署与 WASM Coprocessor 动态升级。
- 将“提交交易”到“所有目标节点确认可调用”的时间定义统一为 `submitTime` 到 `allReadyTime`，并保留每个节点完成时间和最慢节点。
- 支持重复轮次，输出包含 `method`、`nodeCount`、`round`、`submitTime`、`allReadyTime`、`totalLatencyMs`、`slowestNode` 的汇总 CSV，以及 JSON、逐节点 CSV 和子实验原始结果。
- 提供网络启动、验证、方法运行和清理的可复现命令参数与失败记录。

## Capabilities

### New Capabilities

- `lab1-upgrade-latency-comparison`: 定义 Solidity/WASM 两种方案在固定 fixture、节点规模和重复轮次下的端到端延迟采集与结果格式。

### Modified Capabilities

- None.

## Impact

- 新增 `experiments/cryptoupgrade/bench/cmd/benchlab1` 命令及其测试、文档。
- 复用 `genlocalnetwork`、`multinode`、`benchsolidityupgradelatency` 和 `benchupgradelatency`，不改变 Geth 共识、EVM 或动态加载语义。
- 必要时让现有 benchmark 在算法制品被归档到 `archive/` 后仍能解析对应 fixture。

## 非目标

- 不比较算法执行吞吐、Gas 或运行期资源消耗。
- 不把编译 Solidity、构建 WASM、启动 Docker 网络和 preflight 时间计入升级/部署总耗时。
- 不在同一网络中复用已部署合约或已加载模块作为下一轮基线。
