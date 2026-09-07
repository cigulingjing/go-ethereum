# experiments.md

本文档定义论文实验的目标、变量、指标及结论边界。实验对象统一为 **contract-facing cryptographic algorithms（面向智能合约的密码算法）**。论文结论不得超出本文定义范围。

## 与当前大纲的关系

`paper_outline.md` 是当前论文结构目标；本文档是实验事实和结论边界。二者冲突时，正文结构服从 `paper_outline.md`，实验数值、指标状态和允许结论服从本文档与原始 JSON / CSV。

按当前大纲，实验材料进入正文的位置如下：

| 论文位置 | 对应实验材料 | 正文用途 |
| --- | --- | --- |
| `5.1 Experimental Setup` | Lab1、Lab2 与 WASM rerun 的共同环境说明 | 说明 Geth、Clique、节点配置、WASM runtime、算法集合和指标来源 |
| `5.2 Upgrade Efficiency` | Lab1 + 部署 / 上传成本附属实验 | 报告升级延迟、事件后可调用时间和上传 / 部署成本 |
| `5.3 Execution Efficiency` | Lab2 + Sha256 真实链附属实验 | 比较 WASM-backed Upgrade、Precompile 与 Solidity Contract 的 `eth_call` 延迟和 `gasEstimate` |
| `4.4 Upgrade Consistency` | 不设置实验内容 | 仅从协议机制分析 Activation Block、链上元数据和版本选择规则带来的一致性边界 |
| `2.3 / 3.6 / 4.2` | WASM rerun、WASM 模块与运行时实现信息 | 支撑 WebAssembly Runtime、Prototype Implementation 与 Execution Isolation 的实现叙述 |

`paper_outline.md` 中提到的节点规模扩展、升级期间交易成功率、最长出块间隔、State Root、fail-stop、无 Activation Block 对照或 Lab3 一致性实验，当前都不属于论文实验内容，只能写为 `TODO`、future work 或安全分析边界。

若 `paper_outline.md` 的算法清单仍保留“待按 WASM 路径重做”状态，写作时以本文件和 `wasm-rerun-20260902-225029` 的原始结果为准：当前 WASM rerun 已覆盖 7 个算法的 Lab1/Lab2。该目录中的 upgrade-stability / Lab3 产物不进入论文实验内容。

路径约定：除特别说明外，本文档路径按仓库根目录书写；若从 `paper/` 目录执行命令，访问 `cryptoupgrade/` 或 `experiments/` 下的数据需加 `../` 前缀。

## 数据分层

| 层级 | 用途 | 权威性 |
| --- | --- | --- |
| JSON / CSV | 数值、时间点、一致性字段 | **唯一数值来源** |
| `paper/image/<lab>/` | 论文用图（svg/pdf/png/tiff） | 只用于排版，不单独作为结论来源 |
| 图注 / `paper/image/README.md` | 读图辅助 | 与本文或 JSON/CSV 冲突时，以本文与原始结果为准 |

写作时先读原始结果，再引用对应图片。缺失字段标 `TODO`，不得用图注补数字或补机制。

工程侧实验名与论文 Lab 对应：

| 论文 | 实验命令 | 工程 change / 结果目录名 |
| --- | --- | --- |
| Lab1 | `benchupgradelatency` | `lab1-upgrade-latency` |
| Lab2 | `benchexecutionefficiency` | `lab2-execution-efficiency` |

`benchupgradestability` / `lab3-upgrade-stability` 不设置为论文实验。

---

# 实验目标

围绕两个 Evaluation 问题展开：

1. 动态升级需要多少时间开销，链下升级准备是否脱离交易执行关键路径？
2. 引入动态升级能力后，WASM-backed 密码算法执行效率是否仍接近 Precompiled Contract？

---

# WASM rerun

当前实现基座已经从 Go plugin 切换为 WASM module。`experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/summary.md`、`lab1/`、`lab2/` 及 `figures/` 下与 Lab1/Lab2 对应的源数据，是当前论文中关于 WASM 路径的最新证据包。

## 用途

* 验证 WASM module 替换 Go plugin 后，升级路径和执行路径仍可在同一 5 节点 Clique harness 下复现。
* 为“统一字节码 + 受控运行时”提供当前实现层面的证据，而不是形式化证明。
* 为 `Technical Background / WebAssembly Runtime`、`Coprocessor Architecture / Prototype Implementation`、`Security Analysis / Execution Isolation` 和 `Evaluation` 中的 WASM 相关表述提供实现支撑。

## 允许结论

* Go plugin 替换为 WASM module 后，模块载荷和执行边界更容易保持一致。
* 在当前工具链与运行时下，WASM rerun 可以复现 Lab1/Lab2 的主要观测。
* 该 rerun 可以作为新大纲中 Coprocessor Architecture、Security Analysis 和 Evaluation 相关段落的证据来源。

## 禁止结论

* 不得将该 rerun 表述为对任意平台的形式化一致性证明。
* 不得将 WASM sandbox 表述为对所有攻击都成立的安全证明。
* 不得把 rerun 中的 upgrade-stability / Lab3 / `scheduled` / `immediate` harness 结果写成论文实验内容。

---

# Lab1：Upgrade Efficiency（升级延迟）

## 目的

测量从控制端提交升级交易，到所有目标节点都能通过 `CodeStorage.callFunc` 返回期望结果的可观测端到端延迟，并拆分各阶段耗时。

完成判据不是名为 Ready 的运行时状态，而是该节点 RPC 上 `callFunc` 返回期望输出。结果应解释为**控制端可观测延迟**（含 RPC 往返与 poll interval），不是节点内部编译耗时的精确剖分。

## 正式实验环境

* 网络：`experiments/cryptoupgrade/deployments/networks/local-5nodes.yaml`
* 共识：Clique，`period = 5s`，1 signer + 4 observer
* 节点规模：**5**（论文正式数据）
* 配置存在但无正式结果：`local-2nodes.yaml`、`local-10nodes.yaml`
* 每算法正式采集 1 轮；上传函数名带 `Latency001` 后缀，避免同名算法污染

不得把 2/10 节点配置写成已完成规模对比。

## 流程与 JSON 字段

```text
transactionSubmittedAt
        ↓
txHashObservedAt
        ↓
receiptObservedAt  ≈  eventObservedAt
        ↓
各节点 completedAt（callFunc 返回期望结果）
        ↓
activationObservedAt（全部目标节点完成）
```

正式 5 节点数据中，`eventObservedAt` 与 sender 的 `receiptObservedAt` 为同一时刻，`receiptToEventMillis` 基本为 0。事件确认来自升级交易 receipt 中的 `codeUploaded` log，不是节点内部 listener 日志时间。

## 记录时间点

| 论文记号 | JSON 字段 | 含义 |
| --- | --- | --- |
| `T_submit` | `phaseTimeline.transactionSubmittedAt` | 控制端开始发送升级交易 |
| `T_txhash` | `phaseTimeline.txHashObservedAt` | sender RPC 返回交易哈希 |
| `T_receipt` | `phaseTimeline.receiptObservedAt` | sender RPC 返回成功 receipt |
| `T_event` | `phaseTimeline.eventObservedAt` | 控制端从该 receipt 确认 `codeUploaded` |
| `T_complete_i` | `nodes[].completedAt` | 节点 i 首次 `callFunc` 成功 |
| `T_all_complete` | `phaseTimeline.activationObservedAt` | 全部目标节点 `callFunc` 成功 |

## 核心指标

对应 `rounds[].summary` / 汇总 CSV 字段：

* RPC 提交延迟：`submitToTxHashMillis`（`T_txhash - T_submit`）
* 交易确认延迟：`txHashToSenderReceiptMillis`（`T_receipt - T_txhash`）
* 事件后至全网可调用：`eventToAllCompleteMillis`（`T_all_complete - T_event`）
* 端到端升级延迟：`submitToAllCompleteMillis`（`T_all_complete - T_submit`）
* 节点完成时间离散：`nodes[].submitToCompleteMillis` 的最小/最大差

节点级一手数据在各算法目录的 `nodes.csv`。

与 `paper_outline.md` 的指标对应关系：

* 提案确认时间可由 `submitToTxHashMillis` 和 `txHashToSenderReceiptMillis` 支撑；
* 单节点准备时间和全网升级完成时间可由 `nodes.csv` 与 `eventToAllCompleteMillis` 支撑；
* 模块获取、模块校验和 WASM 加载时间若原始 JSON / CSV 未拆分记录，正文应标 `TODO`，不得从总延迟反推；
* 升级期间交易成功率和最长出块间隔当前未采集，只能列入 future work。

## 测试算法

正式 5 节点延迟实验覆盖：

* Add（非密码计算基线）
* Sha256
* Blake2bSum256
* Pbkdf2Sha256
* Dh2048Secret
* PedersenCommit
* SchnorrVerify

## 数据路径

当前 WASM 路径优先结果：

* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/summary.md`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/lab1/result.json`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/lab1/rounds.csv`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/lab1/nodes.csv`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/figures/lab1_upgrade_latency_source.csv`

历史 Go plugin 路径核对结果：

* `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/summary.csv`
* `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/upgrade_latency_figure_data.csv`
* 分算法原始结果：`cryptoupgrade/results/upgrade-latency/lab1-5nodes-other-algos-20260813-095549/<Algorithm>/result.json`（Add 见 combined summary 中的 `resultJson` 路径）

论文用图：

* 目录：`paper/image/lab1-upgrade-latency-availability/`
* 主图：`lab1_upgrade_latency_5nodes.*`
* WASM rerun 正文图：`paper/image/wasm-rerun/figure_wasm_evaluation.*`，源数据为 `paper/image/wasm-rerun/figure_wasm_evaluation_source_data.csv` 中的 Lab1 panel

## 附属实验：部署 / 上传成本

该图在同一图目录，但**不是** `benchupgradelatency` 的时间线实验。

* 比较对象：动态升级 `uploadCode` 上传 vs Solidity 合约部署；Precompile 无部署交易
* 覆盖算法：Add、Blake2b-256
* 权威说明：`experiments/cryptoupgrade/docs/test_result.md`
* 论文用图：`lab1_deployment_upgrade_cost.*`

部署耗时从 `eth_sendTransaction` 统计到 receipt；调用阶段不计入本附属实验主结论。

## 允许结论

* 给出 5 节点 Clique 下各阶段的控制端可观测时间开销；
* 比较不同算法的升级延迟与节点完成时间离散程度；
* 说明已实现异步升级架构：升级交易先完成链上执行并获得 receipt，编译与加载随后由链下任务完成，因而不位于该升级交易的 EVM 执行关键路径；
* 附属实验可描述上传耗时与部署 Gas 相对 Solidity 部署的差异，不得外推到全部算法。

升级窗口内其他 EVM 交易的延迟/吞吐损耗不在本次实验范围，列为 future work（见 `paper/README.md`）。Discussion 可一句带过，不得当作已测结果。

## 禁止结论

* 不得仅根据升级总时间宣称“零停机”；
* 不得声称已测量升级阶段对其他 EVM 交易或系统可用性的损耗；
* 不得把链下编译时间或 `eventToAllCompleteMillis` 描述为链上交易执行时间；
* 不得比较未采集的 1/10 节点规模；
* 不得把控制端轮询延迟写成节点内部精确编译剖分。

---

# Lab2：Execution Efficiency（密码算法执行效率）

## 目的

验证动态升级能力是否引入明显的单次密码算法调用开销。本实验只比较算法**已经**升级、部署或 precompile 注册之后的调用效率；setup 交易只用于复现，不作为执行效率主结论。

## 对比方案

1. **WASM-backed Upgrade**：经 `CodeStorage.uploadCode` 动态加载的 WASM 实现，调用 `CodeStorage.callFunc`；
2. **Precompile**：客户端内置 Native Precompiled Contract；
3. **Contract**：Solidity 合约实现。

实验命令：`benchexecutionefficiency`。三类实现使用同一组逻辑输入，输出不一致则实验失败。

## 测试算法

正式数据已覆盖下列 7 个算法，不再作为待定项：

* Add：非密码计算基线
* Sha256
* Blake2bSum256
* Pbkdf2Sha256
* Dh2048Secret
* PedersenCommit
* SchnorrVerify

测量参数：warmup = 10，n = 100。

## 核心指标

* 调用延迟：对每条路径重复 `eth_call`，使用 `latency.mean`（可辅以 first / p50 / p95 / min / max）
* 调用 Gas：**`gasEstimate`**，来自一次 `eth_estimateGas`，不是交易 receipt 的 `gasUsed`
* 相对 Precompile 的延迟开销：

```text
(T_upgrade - T_precompile) / T_precompile
```

其中 `T_*` 取该方案 `eth_call` 的 mean latency。JSON 另有 `ratios` 字段，写作时与原始 `latency` 交叉核对。

## 实验原则

三种方案应尽可能使用等价输入和等价算法语义。

延迟与 Gas 分别解释：

* `eth_call` 延迟用于评价 RPC 触发的 EVM 执行路径耗时，不是纯算法核函数时间；
* `gasEstimate` 用于评价当前定价模型下的链上调用成本估算。

## 数据路径

当前 WASM 路径优先结果：

* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/summary.md`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/lab2/result.json`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/figures/lab2_execution_efficiency_source.csv`
* `experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/figures/lab2_latency_samples_source.csv`

历史 Go plugin 路径核对结果：

* `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.json`
* `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.txt`

论文用图：

* 目录：`paper/image/lab2-execution-efficiency/`
* 主图：`lab2_execution_efficiency_latency.*`、`lab2_execution_efficiency_gas_estimate.*`
* WASM rerun 正文图：`paper/image/wasm-rerun/figure_wasm_evaluation.*`，源数据为 `paper/image/wasm-rerun/figure_wasm_evaluation_source_data.csv` 中的 Lab2 panels

## 附属实验：真实链 Upgrade vs Precompile

该图在同一图目录，但**不是**三方案执行效率主实验。

* 命令：`benchrealchain`
* 算法：仅 Sha256
* 对比：Upgrade vs Precompile（无 Solidity Contract）
* 测量：Upgrade 的 setup 交易成本；Precompile setup 为 0；随后两条路径的 `eth_call` 延迟与 `gasEstimate`
* 权威结果：`experiments/cryptoupgrade/docs/real_chain_upgrade_vs_precompile_result.json`、`experiments/cryptoupgrade/docs/real_chain_upgrade_vs_precompile.md`
* 论文用图：`lab2_real_chain_precompile_comparison.*`

不得把该附属实验写成“全部算法的真实链三方案对比”。

## 允许结论

* 判断 WASM-backed Upgrade 与 Precompile 的 `eth_call` 延迟是否处于相近水平；
* 分析复杂密码算法在 Solidity 实现下的延迟或 `gasEstimate` 开销；
* 说明动态调用机制是否引入明显的运行时额外开销；
* 附属实验可描述 Sha256 真实链上 WASM-backed Upgrade 相对 Precompile 的 setup 成本、调用延迟与 `gasEstimate`。

当前 WASM rerun 支持的正文表述应写为：7 个算法均完成三方案输出一致性检查；其中 6 个算法的 WASM-backed Upgrade mean `eth_call` 延迟与 Precompile 处于毫秒级相近范围，SchnorrVerify 是明显尾部例外，不得被写进“全部算法接近 Precompile”的结论。

## Gas 结论边界

WASM-backed Upgrade 与 Precompile 的 `gasEstimate` 采用人为定义的 Native 定价模型，例如：

```text
Base Gas + Input-dependent Gas
```

因此 Gas 数据不能用于证明本文方案具有天然更低的经济成本，只能描述**当前定价模型下的调用成本估算**。不得把 `gasEstimate` 写成 receipt `gasUsed`。

## 禁止结论

未经统计检验，不使用 `statistically significant`。

除非数据明确支持，不声称：

* Upgrade 比 Precompile 更快；
* Upgrade 在所有算法上优于 Solidity；
* 当前 Gas 定价优于 Ethereum 官方定价。

核心目标是：

> WASM-backed Upgrade 在获得动态升级能力的同时保持与 Precompiled Contract 相近的执行效率。

---

# 不纳入实验内容：Upgrade Consistency

当前论文不设置一致性实验。`Security Analysis / Upgrade Consistency` 只能从协议机制讨论一致性边界，包括链上升级元数据、`wasmHash`、`activationHeight`、区块高度驱动的版本选择，以及未就绪节点不应静默回退到旧版本的设计要求。

仓库中已有的 `benchupgradestability`、`upgrade-stability`、`wasm-rerun-20260902-225029/lab3/` 和 `lab3_upgrade_stability_source.csv` 产物不进入论文实验体系，不出现在 `Evaluation` 中，不设置图表，不作为 formal claim 的实验支撑。

正文允许在安全分析中写：

* 升级一致性依赖链上元数据、模块哈希和 Activation Block 的确定性选择规则；
* 节点本地时间和本地准备完成时刻不应参与版本选择；
* 若节点在 Activation Block 到达时尚未准备好目标 WASM artifact，设计上不应继续执行旧版本并声称成功。

正文不得写：

* 本文设置或完成了一项 Lab3 一致性实验；
* 本文通过实验验证了所有节点的版本、输出、receipt/event 或 chain-view 一致；
* `scheduled` / `immediate` 是正文实验 baseline；
* 已采集 `stateRoot` 或已验证 State Root 一致；
* 已验证 fail-stop、未就绪节点行为或无 Activation Block 会导致分叉。

---

# 实验与论文大纲对应关系

| 论文大纲位置 | Design | Experiment | 验证属性 | 正式数据规模与边界 |
| --- | --- | --- | --- | --- |
| `3.3 Upgrade Protocol` / `3.4 Deterministic Activation` / `5.2 Upgrade Efficiency` | 链上协调 + 链下异步准备 | Lab1（+ 部署成本附属实验） | Upgrade latency / post-event readiness | 5 节点 Clique，7 算法；不得写成 2/10 节点规模实验 |
| `3.5 Resource Accounting` / `5.3 Execution Efficiency` | Cryptographic Coprocessor + deterministic gas model | Lab2（+ 真实链附属实验） | `eth_call` latency / `gasEstimate` | 7 算法三方案；真实链仅 Sha256；SchnorrVerify 为尾部例外 |
| `4.4 Upgrade Consistency` | Activation Block | 不设置实验 | 协议属性 / 安全边界分析 | 不报告一致性实验结果；不得引用 `scheduled` / `immediate` harness 作为 baseline |
| `2.3 WebAssembly Runtime` / `3.6 Prototype Implementation` / `4.2 Execution Isolation` | Go plugin → WASM module | WASM rerun | WASM payload boundary / Lab1-Lab2 rerun consistency | 5 节点 Clique，7 算法，同一 harness；不构成形式化沙箱或跨平台一致性证明 |

最终论证链：

```text
Cryptographic Coprocessor
        ↓
支持 WASM 模块动态替换
        ↓
Lab2：动态能力没有明显牺牲执行效率

Asynchronous Upgrade Preparation
        ↓
编译加载脱离交易执行关键路径
        ↓
Lab1：量化升级各阶段开销

Deterministic Activation
        ↓
异步准备、按区块统一生效
        ↓
Security Analysis / Upgrade Consistency：从协议机制说明版本切换边界，不设置实验

WASM rerun
        ↓
Go plugin 替换为 WASM module
        ↓
支撑 WebAssembly Runtime、Prototype Implementation、Execution Isolation 与 WASM 路径下的 Lab1/Lab2 复现
```
