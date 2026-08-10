## Why

论文实验需要在算法完成升级或部署以后，比较同一算法在动态升级方案、contract 方案和 precompile 方案中的执行效率。现有实验已经覆盖升级过程和部分部署成本，但还缺少一个只聚焦“升级后调用效率”的统一测试入口，无法稳定收集调用时间与 Gas 消耗两项指标。

本 change 只关注升级完成以后的算法调用，不统计、不分析升级过程、部署过程、版本切换或注册过程本身的成本。

## What Changes

- 新增升级后执行效率实验入口，用于在同一条本地 Geth 测试链上执行三类实现的调用对比。
- 三类实现统一命名为：`upgrade`、`precompile`、`contract`。
- 每次调用实验 SHALL 收集调用时间和调用 Gas 两项核心指标，并校验三类实现的逻辑输出一致。
- 实验入口先覆盖 `Add`，并继续扩展到已有算法中三类方案都具备可比实现的算法。
- 对缺少三方可比实现的算法，实验结果 SHALL 明确记录跳过原因，不输出部分对比数据。
- 实验准备阶段可以执行必要的升级、部署或地址初始化，但本 change 不统计、不分析升级过程耗时或升级成本。
- 输出机器可读 JSON 与简明文本摘要，便于把实验结果反馈给用户并复现实验。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 增加升级后算法调用效率实验要求，覆盖已有可对齐算法在动态升级、contract 和 precompile 三类实现中的调用时间与调用 Gas 对比。

## Impact

- Affected code: `cryptoupgrade/cmd/` 下新增或扩展实验命令；同步将项目内实验和 precompile 相关的 `native` 命名改为 `precompile`。
- Affected inputs: 已有 Go 算法源码、Solidity contract 算法、precompile 地址、测试输入和期望输出。
- Affected artifacts: OpenSpec change、批量算法 JSON 原始结果、文本摘要和必要的实验说明。
- No breaking changes to Geth consensus, EVM semantics, dynamic crypto upgrade activation, Solidity algorithm contracts, or precompile behavior.
