## Context

现有 `benchsolidityupgradelatency` 和 `benchupgradelatency` 已分别实现多节点观测，但它们是独立命令，结果字段和网络生命周期也分别管理。两者都使用控制端单调时间观测 sender RPC、各节点 receipt 和最终可调用状态；本 change 只增加比较编排，不改变这两个 benchmark 的链上语义。

技术栈固定为 Go、`go/ethereum` JSON-RPC、现有 `experiments/cryptoupgrade/network` 配置/渲染包、Docker Compose、CSV/JSON 标准库和现有 Solidity/WASM fixture。

## Goals / Non-Goals

**Goals:**

- 新增 `benchlab1`，默认执行 `Sha256` fixture 的 Solidity/WASM 对比。
- 对每个节点规模和方法使用独立网络目录、Docker network、datadir、pluginDir、RPC/P2P 端口和 benchmark 结果目录。
- 让子 benchmark 继续负责方法特有的交易提交、receipt 轮询、合约调用或 WASM 激活判定；编排器只做生命周期控制和结果规范化。
- 生成可直接绘图的汇总 CSV，并保留逐节点完成时间和原始 JSON。

**Non-Goals:**

- 不把两个已有 benchmark 重构成共享运行时包。
- 不在本 change 中增加节点内新的事件埋点或改变 `CodeStorage` ABI。
- 不把网络启动、镜像构建、preflight 和 Solidity 编译耗时纳入 `totalLatencyMs`。

## Decisions

1. **使用独立编排命令调用现有 benchmark**

   `benchlab1` 通过 `go run` 子进程调用 `genlocalnetwork`、`multinode`、`benchsolidityupgradelatency` 和 `benchupgradelatency`，读取子命令的 `result.json` 后生成统一结果。这样可以保持已有单方法实验入口的测试和观测逻辑，避免复制交易/RPC 轮询实现。

   Alternative: 将两个命令重构为共享 internal package。该方案可减少子进程开销，但会触及大量已有结果结构和用户未完成的工作区改动，不适合本次实验入口的最小范围。

2. **每个 method/nodeCount 建立一套干净网络**

   编排器为 `solidity-1`、`wasm-1` 等每个系列生成独立 YAML、render 目录和 Compose project/network，并在系列结束后 `down -v`。同一系列的重复轮次由子 benchmark 使用新合约地址或唯一 WASM upload name 隔离。

   Alternative: 在一套网络中交替提交两种交易。该方案会让合约、plugin cache、区块高度和本地磁盘状态互相影响，不能支撑方法间公平比较。

3. **固定 SHA-256 fixture**

   默认算法为 `Sha256`，输入由已有 fixture 固定为 `hello cryptoupgrade`，两种子 benchmark 各自使用同一 expected output 校验。`-algorithm` 只允许已有两边共同支持的命名，便于后续扩展到其它密码学 fixture。

   Alternative: 让用户在命令行分别传入 Solidity calldata 和 WASM ABI。该方案容易造成输入不一致，也会使 CSV 难以复现。

4. **以节点完成时间重新计算比较字段**

   编排器从每个子结果的 `transaction_submitted_at`/`phaseTimeline` 和 node-level `completedAt` 解析时间。`allReadyTime` 取所有目标节点 ready 时间的最大值，`slowestNode` 取对应节点；`totalLatencyMs` 使用两者的时间差而不是复制一个可能缺失的汇总字段。失败轮次保留行，但其延迟字段为空。

   Alternative: 只读取子 benchmark 的平均延迟或最后一个 CSV 行。该方案会丢失最慢节点和部分失败信息。

5. **保留两层 CSV**

   `summary.csv` 一行对应一个 method/nodeCount/round，是绘图的主输入；`nodes.csv` 一行对应一个节点完成观测，是阶段和异常审计输入。`result.json` 记录每个 series 的命令、路径、状态和原始 benchmark JSON。

   Alternative: 只输出 summary CSV。该方案无法验证 `allReadyTime` 是否由最慢节点决定，也无法定位单节点超时。

6. **归档 fixture 路径兼容**

   现有工作区将部分算法源文件移动到 `archive/`。两个子 benchmark 和网络生成器在原始 fixture 路径不存在时按同目录 `archive/<basename>` 回退，结果记录实际使用的路径；不复制或修改算法内容。

## Risks / Trade-offs

- [Risk] 子进程编译和 Docker 生命周期增加实验总墙钟时间。→ Mitigation: 这些阶段在测量区间外，并在 manifest 中单独记录；支持 `-skip-build` 和 `-keep-containers`。
- [Risk] 控制端轮询间隔和 RPC 往返影响 ready 观测。→ Mitigation: 两种方法使用相同 `poll-interval`，CSV/JSON 记录配置和限制。
- [Risk] 某个系列失败时可能留下运行中的容器。→ Mitigation: 每个 series 使用 `defer` 清理 Compose；`-keep-containers` 仅用于故障排查。
- [Risk] `solc`、Docker 或 WASM runtime 环境缺失。→ Mitigation: 在 manifest 和 series 错误中保留命令日志；单元测试覆盖解析/归一化，不把本地环境可用性伪装成成功实验。

## Migration Plan

1. 先运行 `go test` 验证新编排器、CSV 归一化和 fixture 路径回退。
2. 使用 `-rounds 1 -node-counts 1` 做最小 Docker smoke，确认两种方法均能产生完成行。
3. 使用默认 `-node-counts 1,2,5,10` 和正式重复次数采集 `summary.csv`、`nodes.csv`、`result.json`。
4. 若实现需要回滚，只删除新增命令和本次结果目录；已有两个 benchmark 和 Geth 运行时不受影响。
