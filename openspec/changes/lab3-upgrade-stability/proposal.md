## Why

实验 2 已经比较升级后算法调用效率，但还没有验证“升级交易进入链后是否会影响私有链一致性”。论文需要证明基于合约事件触发的升级流程不会导致多节点链状态、版本视图或算法输出产生分叉。

## What Changes

- 新增升级稳定性实验能力，在私有链多节点环境中提交升级交易并观测所有目标节点。
- 验证合约侧是否能作为升级触发器和代码管理模块：用户调用升级合约，合约记录算法版本、生效区块和源码元数据，并发出升级事件。
- 验证 Geth 事件监听与动态加载改造是否能按链上事件完成本地算法激活。
- 支持两类版本切换策略：到指定区块启用新算法，以及立即切换到新算法。
- 实验完成后输出链一致性、版本一致性、事件一致性和算法输出一致性数据。

## Capabilities

### New Capabilities

- `cryptoupgrade-upgrade-stability`: 定义升级交易在多节点私有链中的一致性验证、版本生效边界和实验输出要求。

### Modified Capabilities

- None.

## Non-Goals

- 不重新评估实验 2 的执行时间和 Gas 消耗对比。
- 不设计 rollback、权限治理、多签升级或复杂链重组恢复机制。
- 不改变 Geth 原有共识规则、EVM 语义或已有算法 ABI。

## Impact

- Affected code: `cryptoupgrade/contracts/`、`common/cryptoupgrade_contract.go`、`cryptoupgrade/internal/evm/`、`cryptoupgrade/internal/event/`、`cryptoupgrade/internal/activation/`、`experiments/cryptoupgrade/bench/cmd/` 和多节点私链辅助工具。
- Affected artifacts: OpenSpec change、稳定性实验命令、私有链实验配置、JSON/CSV 原始结果和文本摘要。
- No breaking changes to existing execution-efficiency benchmarks, precompile implementations, or single-node upgrade workflows.
