# experiments.md

本文档定义论文实验的目标、变量、指标及结论边界。实验对象统一为 **contract-facing cryptographic algorithms（面向智能合约的密码算法）**。论文结论不得超出本文定义范围。

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
| Lab3 | `benchupgradestability` | `lab3-upgrade-stability` |

---

# 实验目标

围绕三个研究问题展开：

1. 动态升级需要多少时间开销，链下升级准备是否脱离交易执行关键路径？
2. 引入动态升级能力后，密码算法执行效率是否仍接近 Native Precompile？
3. 多节点异步完成升级准备时，能否在统一 Activation Block 完成确定性切换，并保持链视图、版本与输出一致？

---

# Lab1：升级延迟

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

权威结果：

* `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/summary.csv`
* `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/upgrade_latency_figure_data.csv`
* 分算法原始结果：`cryptoupgrade/results/upgrade-latency/lab1-5nodes-other-algos-20260813-095549/<Algorithm>/result.json`（Add 见 combined summary 中的 `resultJson` 路径）

论文用图：

* 目录：`paper/image/lab1-upgrade-latency-availability/`
* 主图：`lab1_upgrade_latency_5nodes.*`

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

# Lab2：密码算法执行效率

## 目的

验证动态升级能力是否引入明显的单次密码算法调用开销。本实验只比较算法**已经**升级、部署或 precompile 注册之后的调用效率；setup 交易只用于复现，不作为执行效率主结论。

## 对比方案

1. **Upgrade**：经 `CodeStorage.uploadCode` 动态加载的 Native 实现，调用 `CodeStorage.callFunc`；
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

权威结果：

* `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.json`
* `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.txt`

论文用图：

* 目录：`paper/image/lab2-execution-efficiency/`
* 主图：`lab2_execution_efficiency_latency.*`、`lab2_execution_efficiency_gas_estimate.*`

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

* 判断 Upgrade 与 Precompile 的 `eth_call` 延迟是否处于相近水平；
* 分析复杂密码算法在 Solidity 实现下的延迟或 `gasEstimate` 开销；
* 说明动态调用机制是否引入明显的运行时额外开销；
* 附属实验可描述 Sha256 真实链上 Upgrade 相对 Precompile 的 setup 成本、调用延迟与 `gasEstimate`。

## Gas 结论边界

Upgrade 与 Precompile 的 `gasEstimate` 采用人为定义的 Native 定价模型，例如：

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

> Upgrade 在获得动态升级能力的同时保持与 Native Precompile 相近的执行效率。

---

# Lab3：升级过程状态一致性

论文标题使用“状态一致性”；工程实验名为 `lab3-upgrade-stability`，命令为 `benchupgradestability`。二者指同一组正式数据。

## 目的

验证升级交易进入私有链后，各节点是否在同一区块高度选择相同算法版本，并保持链视图、receipt/event 字段与 `callFunc` 输出一致。本地 plugin 编译完成时刻不决定生效语义；生效由链上 `activationBlock` 与执行区块号决定。

## 正式实验环境

* 网络：`experiments/cryptoupgrade/deployments/networks/local-5nodes.yaml`
* 共识：Clique，`period = 5s`，5 节点
* 对照：**scheduled** vs **immediate**（两种模式都带 `activationBlock`）
* 轮次：每种模式 2 轮，共 4 轮，全部通过

`immediate` 将 `activationBlock` 设为升级交易生效区块（当前/收据区块），不是“关闭 Activation Block 的对照”。仓库中不存在

```text
Without Deterministic Activation
vs.
With Activation Block
```

对应的实验命令或结果。不得把 scheduled / immediate 写成有/无确定性激活。

## 核心场景

```text
uploadCodeVersion（含 version 与 activationBlock）
        ↓
Upgrade Event
        ↓
各节点异步准备本地 artifact
        ↓
按区块号选择版本
  scheduled: H < activationBlock → V_old；H >= activationBlock → V_new
  immediate: 升级交易生效区块起使用 V_new
```

## 记录数据

每个采样点按节点记录，对应 `samples.csv` / `result.json` 字段。正式数据**没有** `stateRoot`。

* `block_number` / `block_hash`
* `head_number` / `head_hash`
* `receipt_visible`、`receipt_block`、`receipt_block_hash`
* `version`、`activation_block`、`metadata_hash`
* `output` / `output_hex`
* `ok` / `error`
* `mode`（`scheduled` / `immediate`）、`stage`（`before` / `after`）

## 一致性判定

### Version Consistency

同一采样区块上，各节点 `version` 与预期版本一致：

```text
scheduled:  H < H_activation  → V_old
            H >= H_activation → V_new
immediate:  升级交易生效区块起 → V_new
```

### Execution Consistency

相同采样区块和交易输入下，各节点 `output` 一致，且符合该高度应使用的版本。

### Chain-view Consistency

对于同一 canonical 采样区块，各节点 `head_hash`、`block_hash`、receipt 可见性与 `metadata_hash` 一致。

不得把上述观察写成 State Root 等式。若正文需要 state root，标 `TODO`（当前结果未采集）。

## 观察窗口

正式 scheduled 轮次覆盖 activation block 前（`stage=before`）与 activation block（`stage=after`）。immediate 轮次在生效区块采样 `after`。

## 未做实验：fail-stop / 未就绪节点

设计要求：若某节点在 Activation Block 到达时尚未完成本地 artifact 准备，不得静默回退到旧版本并声称执行成功。

正式 5 节点 4 轮实验是全部节点激活成功后的一致性检查，**没有**注入未就绪节点，也没有 fail-stop 结果文件。

不得根据现有数据描述未就绪节点的拒绝/停机行为。

## 数据路径

权威结果：

* `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/result.json`
* `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/samples.csv`
* `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/conclusion.md`

论文用图：

* 目录：`paper/image/lab3-state-consistency/`
* 主图：`lab3_upgrade_state_consistency_5nodes.*`

## 允许结论

* 链上 `activationBlock` 决定版本生效语义，本地 activation 只准备对应版本 artifact；
* scheduled 模式在 activation block 前保持旧版本输出，在 activation block 切换到新版本输出；
* immediate 模式在升级交易生效区块返回新版本输出；
* 在本次 5 节点无故障私有链中，各节点的链视图、receipt/event、版本与算法输出保持一致。

## 禁止结论

* 不得由有限规模私有链实验推导出任意网络规模下的一致性保证；
* 不得将实验观察直接描述为形式化安全证明；
* “状态一致性”不得泛化表述为“系统绝对稳定”，也不得改写成 State Root 一致（未采集）；
* 不得声称已验证 fail-stop 或“无 Activation Block 会分叉”；
* 正式数据未注入显著的节点就绪时间差，不得把 Lab3 写成“在刻意拉大的异步准备条件下的压力测试”。

---

# 实验与论文设计对应关系

| Research Question | Design | Experiment | 验证属性 | 正式数据规模 |
| --- | --- | --- | --- | --- |
| RQ1 | 链上协调 + 链下异步准备 | Lab1（+ 部署成本附属实验） | Upgrade Latency | 5 节点 Clique，7 算法 |
| RQ2 | Cryptographic Coprocessor | Lab2（+ 真实链附属实验） | Execution Efficiency | 7 算法三方案；真实链仅 Sha256 |
| RQ3 | Activation Block | Lab3 | Version / output / chain-view consistency | 5 节点，scheduled vs immediate，4 轮 |

最终论证链：

```text
Cryptographic Coprocessor
        ↓
支持 Native 密码算法动态替换
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
Lab3：验证版本、输出与链视图一致
```
