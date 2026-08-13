## Context

仓库中已经存在多条可复用路径：

- 动态升级方案通过 `CodeStorage.uploadCode` 完成 Go 算法激活，升级完成后通过 `CodeStorage.callFunc(name, encodedInput)` 调用。
- contract 对照代码位于 `cryptoupgrade/algorithm/contracts/`。
- precompile 方案通过 `cryptoupgrade.PrecompileByName(name)` 注册，并使用 `common.CryptoUpgrade*Address` 作为调用地址。
- 现有 `benchcandidate`、`benchrealchain` 和 `benchcall` 已包含 RPC 调用、ABI 编码、receipt 轮询、`eth_call` 计时和部分 Gas 估算逻辑，但没有一个入口同时输出多算法的三类实现调用效率对比。

本 change 只讨论算法已经完成升级、部署或 precompile 注册后的调用效率。为了能一键执行实验，命令仍需要执行必要 setup，但 setup 数据只用于复现实验，不作为“执行效率”结论。

## Goals / Non-Goals

**Goals:**

- 提供实验入口，完成 `upgrade`、`precompile` 和 `contract` 三类实现的调用效率对比。
- 对每个算法 fixture 使用同一组逻辑输入，记录调用时间和调用 Gas 两项核心指标。
- 校验三类实现输出一致，不一致时实验失败。
- 输出 JSON 原始结果和简明文本摘要，便于先反馈已有算法实验结果，再决定是否扩展其他实验维度。
- 保持现有 Geth、EVM、动态升级机制和 precompile 注册行为不变。

**Non-Goals:**

- 不统计升级过程耗时、升级交易 Gas、contract 部署成本或 precompile 注册成本作为核心结论。
- 不新增算法实现，也不补齐缺失的 contract 或 precompile 对照实现。
- 不重新设计算法 ABI、Gas 计费公式或 plugin 生命周期。
- 不把 dev 链出块延迟当作算法执行时间。

## Decisions

1. 新增独立命令 `cryptoupgrade/bench/cmd/benchexecutionefficiency`。

   该命令专门表达“升级后执行效率实验”，避免继续扩大 `benchcall` 的历史 Add/traditional-contract 语义，也避免把 `benchrealchain` 的 upgrade-vs-precompile 两组实验扩展成含 contract 的混合入口。命令内部可以复用或局部抽取现有 helper，但对外输出保持三组统一结构。

2. 使用 per-algorithm fixture 描述三条调用路径。

   不同算法的 Go 函数名、contract 函数名、precompile 名称和 ABI 返回类型并不完全一致，例如 `Blake2bSum256` 在 Go 源码中对应 `Sum256`，contract 返回 `bytes32`，precompile 返回 ABI `bytes`。命令通过 fixture 显式记录这些差异，并在结果中保留实际 ABI 类型。

3. 只比较三方都有可比实现的算法。

   首批覆盖 `Add`、`Sha256`、`Blake2bSum256`、`Pbkdf2Sha256`、`Dh2048Secret`、`PedersenCommit`、`SchnorrVerify`。对 AES、Ed25519、DH key generation 等缺少 contract 对照或只属于辅助生成的算法，命令写入 skipped 列表，而不是输出不完整比较。

4. 调用时间通过重复 `eth_call` 采集。

   每条路径先执行 warmup，再执行 `n` 次测量，记录 first、mean、p50、p95、min、max，并保留必要时复核的原始样本数组。这样测到的是调用方通过 RPC 触发 EVM 执行时看到的延迟，避免交易确认和 dev 链出块行为污染“算法调用效率”指标。

5. 调用 Gas 通过 `eth_estimateGas` 采集。

   对 `upgrade`、`precompile` 和 `contract` 使用各自真实 calldata 执行一次 `eth_estimateGas`，结果命名为 `gasEstimate`。`eth_call` 本身不产生 receipt，因此不声称该字段来自交易 receipt。后续如果需要 receipt `gasUsed`，应作为新的实验维度设计。

6. setup 只负责使三条调用路径可用。

   动态升级路径可在命令内执行一次 `uploadCode`；contract 路径可部署对应 contract 或复用传入地址；precompile 使用固定地址。JSON 可以记录 setup tx hash、地址和状态用于复现，但文本摘要和核心比较只展示调用时间与调用 Gas。

7. 输出以算法为主键。

   JSON 顶层记录实验环境、命令行、样本数和时间戳；`results` 数组按算法分组，每组包含 `upgrade`、`precompile`、`contract` 的 address、calldata size、gasEstimate、latency stats、output 和 setup reference；`ratios` 记录 upgrade/contract、upgrade/precompile、contract/precompile 的耗时与 Gas 比值。

## Risks / Trade-offs

- [Risk] 轻量算法中 RPC 往返和 ABI 编解码可能主导耗时。→ Mitigation: 明确结果解释为端到端 RPC 调用路径耗时，并保留样本数、warmup 和原始样本用于后续重复实验。
- [Risk] `eth_estimateGas` 与真实交易 receipt `gasUsed` 的语义不同。→ Mitigation: 字段命名为 `gasEstimate`，文档和摘要中不称为 receipt gas；如需交易 gas used，另开实验。
- [Risk] 三类实现 ABI 不完全一致。→ Mitigation: fixture 中显式记录 ABI 类型，结果比较使用解码后的逻辑输出。
- [Risk] 扩展现有命令可能引入历史行为回归。→ Mitigation: 新增独立入口，现有 benchmark 命令保持兼容；公共 helper 抽取仅在不会扩大修改范围时进行。
- [Risk] 本地 RPC 节点或 plugin 目录残留可能影响 setup。→ Mitigation: 命令提供输出目录和可选 plugin 目录隔离参数；正式实验记录命令行与相关地址，必要时使用全新 dev 链运行。
