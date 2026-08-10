## 为什么

当前已有升级方案和 native precompile 两条执行路径，但还缺少在真实 geth 链上可复现的时间与执行成本对比实验。需要把实验固定到本地 `http://127.0.0.1:8666` 链，直接量化升级方案与预编译合约方案在部署、调用延迟和 gas 上的差距。

## 变更内容

- 增加真实链 benchmark 流程，用于比较 `CodeStorage.callFunc` 升级执行路径和 cryptoupgrade native precompile 执行路径。
- 默认目标为本地 geth 执行 RPC 端点 `http://127.0.0.1:8666`。
- 统计升级路径的部署/setup 成本，以及 native precompile 路径的无部署/setup 成本。
- 使用 warmup 和重复样本统计 `eth_call` 延迟，包括 first、mean、p50、p95、min、max。
- 对两种方案都通过 `eth_estimateGas` 或等价交易模拟路径统计执行 gas。
- 校验两种方案在相同算法输入下返回等价的 ABI 编码输出。
- 输出机器可读和人工可读的 benchmark 结果，便于形成可复现实验文档。

## 能力

### 新增能力

- 无。

### 修改能力

- `cryptoupgrade-performance-benchmarks`：增加真实链二组 benchmark 要求，用于在本地 8666 geth 链上对比 upgrade-vs-precompile 实验。

## 影响

- 影响代码：`cryptoupgrade/cmd/benchcandidate` 或 `cryptoupgrade/cmd/` 下新的 benchmark 命令。
- 影响文档/结果：`cryptoupgrade/docs/` 下的 benchmark 命令示例和结果文件。
- 影响规格：`openspec/specs/cryptoupgrade-performance-benchmarks/spec.md`。
- 不新增依赖。
