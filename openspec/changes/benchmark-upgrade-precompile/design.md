## 上下文

仓库中已经有 `cryptoupgrade/bench/cmd/` 下的 benchmark tooling 和性能基准 spec。现有工具覆盖 upgrade 调用、Solidity 对比，以及至少一个 Blake2b precompile 对比，但还缺少一个聚焦实验：在运行中的本地 geth 链上，只比较当前升级方案和新增 cryptoupgrade native precompile 路径。

目标测试链是 geth 执行 RPC 端点 `http://127.0.0.1:8666`。benchmark 必须使用真实 RPC 调用，以便测量调用方看到的客户端侧 EVM 执行、升级 setup 的交易提交/receipt timing，以及 `eth_call`/`eth_estimateGas` 行为。

## 目标 / 非目标

**目标：**

- 为本地 8666 geth 链上的 upgrade-vs-precompile 实验提供可复现 benchmark 流程。
- 在 `CodeStorage.callFunc` 和 native precompile 调用之间比较同一算法和同一逻辑输入。
- 测量升级 setup 成本：上传交易耗时、receipt gas used、上传源码大小和压缩后大小。
- 测量调用成本：warmup 和重复 `eth_call` 延迟样本、first/mean/p50/p95/min/max，以及两条路径的 gas estimate。
- 输出可保存为结果制品并写入 `cryptoupgrade/docs/` 的结构化结果。
- 当输出不等价时让 benchmark 失败。

**非目标：**

- 本 change 不 benchmark Solidity。
- 不自动启动或配置 geth 节点。
- 不新增依赖。
- 不改变 native precompile 语义或 gas 公式。
- 不要求 benchmark 提交改变状态的算法调用；重复调用计时使用 `eth_call`。

## 设计决策

1. 增加或改造一个专用 benchmark 命令。

   benchmark 应位于 `cryptoupgrade/bench/cmd/` 下，可以扩展 `benchcandidate` 的 precompile mode，也可以新增聚焦命令，例如 `benchrealchain`。如果扩展 `benchcandidate` 会让既有 Solidity 相关 flags 变得含糊，则优先使用独立命令。

2. 默认 RPC 目标为 `http://127.0.0.1:8666`。

   命令默认连接用户指定的本地 geth 端点，同时保留可配置 `-rpc`，便于对其他端点重复实验。

3. 使用地址选择的 precompile 调用。

   upgrade 调用将 `CodeStorage.callFunc(name, encodedInput)` 发送到 `common.CodeStorageAddress`。precompile 调用直接向已注册 native precompile 地址发送 ABI 编码参数，不包含 method selector。

4. 分开报告 setup 阶段和 call 阶段。

   upgrade 路径有真实 setup 阶段，因为需要上传和激活代码。native precompile 路径没有部署交易，因此 setup 结果应明确记录 deploy tx count 为 0、deploy gas 为 0，同时记录所用 precompile 地址。

5. 同时输出文本和 JSON。

   文本输出便于实验过程中查看；JSON 用于可复现结果文档。JSON 应包含链端点、sender、算法、输入、输出 hash 或原始输出、setup metrics、call metrics、gas metrics、ratios 和 validation status。

## 风险 / 权衡

- 本地节点账户被锁定或缺失 -> 要求使用 `eth_accounts[0]` 或 `-from` 地址，并暴露清晰错误。
- upgrade 上传可能因为 geth 无法编译 plugin 而失败 -> 复用现有上传工具的可操作错误提示。
- RPC 噪声可能影响 timing -> 使用 warmup、重复样本和 percentile 统计，并在结果中记录 iteration count。
- 现有算法 ABI 可能与 precompile ABI 不同 -> 要求显式 input/output ABI type flags，并在输出不匹配时失败。
- precompile gas 和 upgrade gas 不是通过完全相同的内部路径计费 -> 分别报告 gas estimate 和调用延迟，而不是压缩为单一分数。

## 迁移计划

1. 实现 benchmark 命令，或扩展现有命令并提供清晰的 upgrade-vs-precompile mode。
2. 在 `cryptoupgrade/docs/` 下增加命令示例和结果模板。
3. 针对运行中的本地 geth 节点 `http://127.0.0.1:8666` 验证。
4. 保持现有 benchmark 命令兼容。

## 开放问题

- 默认 fixture 使用哪个算法：`Sha256` 最适合 ABI bytes 输出；`Add` 适合观察微小调用开销；两者都可以通过 flags 支持。
- 结果制品应存储完整原始输出，还是为大 byte 输出存储 output hash。
