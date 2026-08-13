## Why

论文实验二需要测量动态升级方案从提交升级交易到算法可用的真实效率，但现有 benchmark 更偏向部署/调用成本对比，尚未把 transaction submission、confirmation、activation 和 validation 分开观测。

本 change 先建立可复现的升级效率实验框架，并只对 `Add` 算法执行一次完整测试，用于验证指标定义、采集方式和本地 dev 链执行流程是否正确。

## What Changes

- 新增 Add-only 升级效率实验命令或脚本，复用当前 `CodeStorage.uploadCode` 和 `CodeStorage.callFunc` 动态升级实现。
- 自动启动 `cryptoupgrade/docs/chain_setup.md` 指定的本地 `geth --dev` 测试链，不搭建额外私有链。
- 自动等待 RPC ready，提交 `Add` 升级交易，等待 receipt，并检测升级后的算法实际可用。
- 分阶段采集 submission、confirmation、activation 和 validation 时间点、区块号、升级交易 hash、gas used、payload/input size、延迟和验证调用结果。
- 将原始实验结果保存为 JSON，并保存对应 Geth 日志。
- 明确标注无法准确观测或在 `geth --dev` 下会合并的指标，不用伪造数据替代。
- 不批量测试 SHA256、PBKDF2、DH2048、Pedersen、Schnorr、BLAKE2b 等其他算法，不生成论文图表。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 增加动态升级方案的升级效率实验要求，覆盖升级交易提交、确认、算法生效和升级后验证调用的分阶段指标。

## Impact

- Affected code: `cryptoupgrade/bench/cmd/` 下新增或扩展实验 tooling；必要时新增 `cryptoupgrade/docs/` 下的实验说明和原始结果目录。
- Affected runtime: 本地 `build/bin/chain/geth --dev` 节点，RPC 默认使用 `http://127.0.0.1:8666`。
- Affected artifacts: OpenSpec change、Add 单次 JSON 原始结果、对应 Geth 日志。
- No breaking changes to Geth consensus, EVM semantics, native precompile registration, or existing benchmark commands.
