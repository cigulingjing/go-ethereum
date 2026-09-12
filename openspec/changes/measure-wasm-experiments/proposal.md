## Why

dynamic-wasm 已经把升级产物切换为 WASM，但 Lab1/Lab2 现有结果仍缺少节点内 WASM 编译阶段、RPC 入口时间和容器资源利用率等论文审计所需指标。现在需要按固定规模重跑实验，并用更细粒度数据判断结果是否支撑 `paper/outline.md` 的实验叙述。

## What Changes

- 扩展 Lab1 升级延迟实验，记录交易进入 RPC、节点开始处理升级事件、WASM 编译完成和算法可调用完成等阶段时间。
- 新增 20 节点、30 节点和 40 节点本地 Clique 配置，并保证每个 geth 容器使用相同 CPU/内存限制。
- 按顺序运行：20 节点全 WASM 算法升级；SchnorrProof 在 5/10/20/30/40 节点规模下升级；单节点 WASM 执行效率与容器 CPU/内存利用率。
- 扩展 Lab2 WASM-only 单节点执行效率实验，不统计 Gas，记录调用耗时样本和容器 CPU/内存利用率。
- 输出统一任务执行报告，并审计实验数据是否足以支撑 `paper/outline.md`。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 扩展 WASM 升级延迟和执行效率实验的观测指标、网络规模、资源约束和报告要求。

## Impact

影响 `experiments/cryptoupgrade/bench/cmd` 实验命令、`experiments/cryptoupgrade/network` Docker Compose 渲染、部署网络 YAML、实验结果目录和报告生成脚本。需要在 geth WASM 激活路径增加最小日志/指标埋点；不改变 EVM 语义、CodeStorage ABI、WASM guest ABI 或共识规则。

## 非目标

- 不修改密码算法实现或算法语义。
- 不把节点日志时间直接伪装为跨节点精确耗时；跨节点指标仍以控制端观测为主，节点内阶段作为补充。
- 不统计 Lab2 Gas，也不重做 Solidity/precompile 三方对比。
