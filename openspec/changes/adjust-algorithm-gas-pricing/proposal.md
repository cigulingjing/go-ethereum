## Why

当前 `CodeStorage.callFunc` 按算法元数据收取固定 Gas，无法反映输入规模、区块间负载及单笔交易对同一算法的密集调用。本变更提出可复现、确定性的动态计价规则，以支持抗滥用和性能实验。

## What Changes

- 将算法执行费用定义为 `g_{k,i} = [m_k · g_0(a_i,x_i) · (1+α)^{max(0,i-N_0)}]`：`m_k` 依赖上一区块调用量，`g_0` 根据前期实验标定，`i` 为交易内同算法调用序号。
- 明确区块与交易计数边界、固定点计算、取整、溢出、版本和激活规则，确保节点重放一致。
- 为不同输入建立基础费用标定数据与复现实验，并测试 Gas 估算、区块重组和阈值前后行为。
- **BREAKING**：启用新规则的实验链上，同一 `callFunc` 的 Gas 与旧规则可能不同；须显式激活，不追溯修改既有区块。

## Capabilities

### New Capabilities

- `cryptoupgrade-algorithm-gas-pricing`: 动态算法计价、区块与交易计数、确定性计算及激活约束。

### Modified Capabilities

None.

## Impact

涉及 `cryptoupgrade` 的 `callFunc` 计费、`core/vm` 的调用上下文、必要的区块执行集成，以及 `experiments/cryptoupgrade` 的标定与对照实验；不改变现有 ABI。

## 非目标

- 不调整普通 EVM opcode、原生 precompile、交易基础 Gas 或 EIP-1559 base fee。
- 不在本提案中指定未经实验验证的 `g_0`、`α`、`N_0`、`m_k` 数值。
