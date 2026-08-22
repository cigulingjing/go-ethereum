# 论文整体内容编写计划

本文档用于指导 `sn-article-template/sn-article.tex` 的后续英文论文写作。它不是实验数据源，也不替代 `claims.md`、`experiments.md`、`terminology.md`、`related_work.md` 和原始 JSON/CSV。写作正文时，所有数值必须回到原始结果文件核对。

## 1. Source of Truth

| 材料 | 用途 | 写作规则 |
| --- | --- | --- |
| `README.md` | 论文范围与 future work | 决定本次可以写和不能写的内容 |
| `paper_plan.md` | 既有论文逻辑草稿 | 作为本文档的输入，不作为最终执行计划 |
| `claims.md` | 允许提出的核心结论 | Abstract、Introduction、Results、Discussion 和 Conclusion 的主要结论必须能映射到 C1-C9 |
| `experiments.md` | 实验目标、指标、数据路径和结论边界 | 实验数值与结论解释必须以该文件和原始 JSON/CSV 为准 |
| `terminology.md` | 中英文术语映射 | 正文术语必须使用规范英文表达 |
| `related_work.md` | 国内外研究综述 | Related Work 的主要结构来源 |
| `references.md` | 已确认参考文献 | 当前为空，正式写作引用前必须补全或改用已核验 BibTeX |
| `sn-article-template/sn-article.tex` | 论文主文件 | 后续正文落入该 LaTeX 文件 |

当前主文件仍主要是 Springer Nature 模板示例内容。正式写作时应删除模板示例章节、表格、图片、算法和引用，只保留必要的 LaTeX 结构。

## 2. Core Argument

### 一句话中文论点

本文提出 Cryptographic Coprocessor，将 contract-facing cryptographic algorithms 的调用接口与客户端 Native implementation 解耦，并通过 asynchronous upgrade preparation 与 deterministic activation 支持运行时升级，同时在已评估场景下保持接近 Precompiled Contract 的执行效率和跨节点状态一致性。

### One-sentence English argument

The proposed Cryptographic Coprocessor enables runtime upgrades of contract-facing cryptographic algorithms by decoupling contract-facing interfaces from native implementations, while retaining native execution efficiency and preserving deterministic version and state consistency under the evaluated asynchronous upgrade scenarios.

### 论证链

```text
区块链密码算法升级需求
        ↓
Solidity 灵活但开销高；Precompiled Contract 高效但升级刚性
        ↓
Cryptographic Coprocessor 解耦调用接口与 Native implementation
        ↓
链上事件协调 + 链下 asynchronous upgrade preparation
        ↓
Activation Block 实现 deterministic activation
        ↓
Lab1: 升级准备脱离升级交易执行关键路径
Lab2: 动态加载算法保持接近 Native Precompile 的执行效率
Lab3: 已评估场景下版本、输出和链视图一致
        ↓
边界: 不声称 zero-downtime、不声称任意规模一致性证明、不声称所有算法都优于 Solidity 或 Precompile
```

## 3. Paper Type and Reader Path

论文类型：algorithmic systems paper。

读者阅读顺序应被设计为：

1. Relevance: 为什么区块链系统需要更灵活的密码算法升级机制。
2. Novelty: Cryptographic Coprocessor 解决的是 Solidity 与 Precompiled Contract 之间的灵活性和效率冲突。
3. Trust: 系统设计、升级协议和三个实验共同支撑核心结论。
4. Reuse: 给出模块边界、调用路径、升级流程、指标定义和数据路径，便于复现。
5. Meaning: 明确当前结果只覆盖 contract-facing cryptographic algorithms 和已评估私有链场景。

## 4. Recommended Writing Order

1. 术语与引用准备：补齐 `terminology.md`，核验 `related_work.md` 中引用并同步到 `sn-bibliography.bib` 或 `references.md`。
2. Method / System Design：先写系统是什么、接口如何解耦、升级协议如何运行。
3. Experiments / Results：再写 Lab1、Lab2、Lab3 的设置、指标、图表和允许结论。
4. Related Work：根据本文贡献回看已有研究，按主题综合，不按单篇文献罗列。
5. Introduction：用已经稳定的设计和实验边界反推问题、gap 和贡献。
6. Discussion：解释结果意义、适用范围、设计代价和 future work。
7. Conclusion：压缩为 contribution -> evidence -> implication -> boundary。
8. Abstract：最后写，必须包含具体实验支撑或 TODO，不填未核验数字。
9. Title：最后定稿，避免 `novel`、`first`、`zero-downtime` 等过度表达。

## 5. Manuscript Structure

建议采用独立 Related Work 章节，偏计算机系统论文结构：

```text
Title
Abstract
1. Introduction
2. Related Work
3. Cryptographic Coprocessor Design
4. Upgrade Protocol and Deterministic Activation
5. Experimental Setup
6. Results
7. Discussion
8. Conclusion
```

若目标期刊要求压缩结构，可将第 3 和第 4 合并为 `Method`，将第 5 和第 6 合并为 `Results`，并将 Related Work 部分并入 Introduction。

## 6. Section-by-Section Plan

### Title

章节任务：用可检索、具体、不过度承诺的方式表达系统对象、核心能力和应用场景。

候选方向：

1. `Runtime-Upgradable Cryptographic Coprocessor for Contract-Facing Algorithms in Ethereum`
2. `A Cryptographic Coprocessor for Runtime-Upgradable Contract-Facing Algorithms`
3. `Asynchronous Preparation and Deterministic Activation for Upgradable Cryptographic Algorithms in Ethereum`

推荐暂定标题：`A Cryptographic Coprocessor for Runtime-Upgradable Contract-Facing Algorithms`

边界：标题不写 `zero-downtime`、`formally verified`、`consensus-safe`、`faster than precompiles`。

### Abstract

写作时间：Results 和 Discussion 稳定后再写。

段落移动：

1. Context: 区块链应用需要持续更新密码算法，尤其在后量子转型背景下。
2. Problem: Solidity 实现灵活但 EVM 开销高；Precompiled Contract 高效但与客户端绑定，升级成本高。
3. Approach: 提出 Cryptographic Coprocessor，解耦 contract-facing interface 与 Native implementation。
4. Key result: 用 Lab1、Lab2、Lab3 的核心结果支撑升级延迟、执行效率和一致性。具体数值从 JSON/CSV 填入，当前标记 TODO。
5. Mechanism: asynchronous upgrade preparation + deterministic activation via Activation Block。
6. Boundary: 结论限定在 contract-facing cryptographic algorithms 和已评估私有链场景。

Claim 映射：C1-C8。C9 只作为设计约束，除非正文明确写为设计要求，不写成实验验证结论。

TODO:

- TODO: 从 Lab1 CSV/JSON 提取最关键的升级阶段时间。
- TODO: 从 Lab2 JSON 提取 Upgrade 与 Precompile 的 mean latency 比较。
- TODO: 从 Lab3 samples/result 提取 scheduled 与 immediate 轮次一致性观察。

### Introduction

章节任务：建立问题重要性，说明现有路径的灵活性和效率冲突，提出本文系统和贡献。

段落 outline：

1. 背景：量子计算和后量子转型使密码算法升级成为长期需求。引用 Shor、Grover、区块链密码应用规范等已核验来源。
2. 现有路径：Solidity 可以实现算法但复杂密码计算会带来较高 EVM 执行和 Gas 开销；Precompiled Contract 使用 Native 实现但升级依赖客户端修改、构建和部署。
3. Gap：现有方法难以同时满足运行时可升级、Native 执行效率和多节点确定性版本切换。
4. Present study：提出 Cryptographic Coprocessor，以及链上协调、链下异步准备、Activation Block 确定性激活机制。
5. Contributions：列出三项贡献，分别对应架构解耦、异步升级机制、Geth 原型和三组实验。

Claim 映射：C1、C2、C3、C4、C7、C8。

注意：Introduction 不写详细实验数值，只预告评估维度和总体发现。

### Related Work

章节任务：按技术主题综合既有工作，解释本文与已有区块链扩展、虚拟机优化和热升级机制的区别。

推荐小节：

1. Blockchain modularity and architecture optimization
   - 来源：`related_work.md` 中区块链架构优化部分。
   - 代表工作：模块化区块链综述、Hyperledger Fabric、微服务架构、并行/异步智能合约执行。
   - 与本文关系：这些工作强调系统模块化与执行效率，但不直接解决 Ethereum/Geth 中 contract-facing cryptographic algorithms 的运行时替换。

2. Blockchain virtual-machine optimization and execution interfaces
   - 来源：`related_work.md` 中区块链虚拟机优化部分。
   - 代表工作：VM-Studio、HyperService、Glimpse、WASM/EVM 对比、ZKVM 相关工作。
   - 与本文关系：这些工作扩展虚拟机能力或降低验证成本，本文聚焦 EVM 与客户端 Native 密码算法实现之间的可升级调用边界。

3. Dynamic software updating and distributed hot updates
   - 来源：`related_work.md` 中动态软件升级和分布式系统热更新部分。
   - 代表工作：KylinX、Mvedsua、PYLIVE、分布式系统自动动态升级和 version consistency。
   - 与本文关系：这些工作提供 DSU 和分布式一致性背景，本文将问题收窄到区块链执行语义、链上协调和区块高度驱动的确定性激活。

4. Blockchain upgrade mechanisms
   - 来源：`related_work.md` 中区块链系统热更新部分。
   - 代表工作：Cardano HFC、Updatable Blockchains、Soft Power、Phoenix、Polkadot runtime upgrades、EVMPatch。
   - 与本文关系：已有机制覆盖协议升级、链上 runtime、合约补丁或客户端热升级；本文区别在于面向密码算法 Native implementation 的模块化加载、异步准备和 Activation Block 生效规则。

结尾 distinction：

本文不试图替代硬分叉、链上治理或合约补丁框架，而是在 Ethereum/Geth 执行环境中，为 contract-facing cryptographic algorithms 提供一个低侵入的 Native 执行和运行时升级路径。

TODO:

- TODO: `references.md` 当前为空，正式写作前需将 `related_work.md` 中使用的参考文献核验并写入 `sn-bibliography.bib`。
- TODO: `related_work.md` 中部分条目存在拼写或日期异常，例如 `vritual`、`percompiled`、`2024-11-89`、`Nil Foundatio`，正式引用前必须修正。

### Cryptographic Coprocessor Design

章节任务：说明系统是什么，解决哪个边界问题，模块如何协作。

建议结构：

1. Task formulation
   - 输入：智能合约侧算法名称、函数入口、输入参数、版本或 metadata。
   - 输出：Native implementation 返回的算法结果。
   - 范围：contract-facing cryptographic algorithms，不是整个 blockchain client 的升级。

2. System overview
   - EVM 侧调用入口。
   - CodeStorage / Upgrade Contract 的职责。
   - Cryptographic Coprocessor 的注册、查找、调用和版本管理职责。
   - Event Listener 与后台升级任务。

3. Interface and implementation decoupling
   - 解释 contract-facing interface 如何保持稳定。
   - 解释 Native implementation 如何独立注册和替换。
   - 支撑 C1。

4. Native execution path
   - 说明动态加载后的算法仍按客户端 Native 代码执行。
   - 避免把动态库加载本身写成主要创新。
   - 为 Lab2 的执行效率结果铺垫。

5. Implementation boundary
   - 说明原型基于 Geth。
   - TODO: 补充核心模块文件路径、关键函数名和调用流程图。

需要图表：

- TODO: 系统架构图，建议文件名 `fig_cryptographic_coprocessor_architecture.*`。
- TODO: 调用路径图，展示 contract -> CodeStorage.callFunc -> coprocessor -> Native implementation。

Claim 映射：C1、C2、C4、C5。

### Upgrade Protocol and Deterministic Activation

章节任务：解释升级如何发生，以及为什么异步准备不会导致节点按本地完成时间随意切换版本。

建议结构：

1. Upgrade submission
   - 开发者通过 `uploadCode` 或 `uploadCodeVersion` 提交升级信息。
   - 链上记录 metadata 并产生事件。

2. Asynchronous upgrade preparation
   - 节点监听事件，在链下执行编译、加载和本地 artifact 准备。
   - 该过程不在升级交易的 EVM 执行关键路径上。
   - 支撑 C3、C6。

3. Activation Block
   - 将准备完成时间与语义生效时间分离。
   - 节点根据区块高度选择版本，而不是根据本地 ready 时间。
   - 固定表达：Asynchronous Preparation, Deterministic Activation。
   - 支撑 C7。

4. Failure boundary
   - 设计上，未在 Activation Block 前准备完成的节点不得静默继续旧版本执行。
   - 现有正式 Lab3 未注入未就绪节点，不能写成已实验验证。
   - C9 只作为设计约束。

需要图表：

- TODO: 升级时序图，展示 `T_submit`、`T_txhash`、`T_receipt`、`T_event`、`T_complete_i`、`T_all_complete`。
- TODO: Activation Block 版本切换图。

Claim 映射：C2、C3、C6、C7、C9。

### Experimental Setup

章节任务：统一说明实验对象、环境、算法集合、基线和指标，避免每个 Lab 重复解释。

应写内容：

1. 实验对象：contract-facing cryptographic algorithms。
2. 算法集合：Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit、SchnorrVerify。
3. 网络环境：Lab1 和 Lab3 使用 5 节点 Clique，`period = 5s`；不得写 2/10 节点规模对比。
4. 对比方案：Upgrade、Precompile、Contract。
5. 指标定义：
   - Lab1: `submitToTxHashMillis`、`txHashToSenderReceiptMillis`、`eventToAllCompleteMillis`、`submitToAllCompleteMillis`、节点完成时间离散。
   - Lab2: `eth_call` mean latency、`gasEstimate`、相对 Precompile latency overhead。
   - Lab3: version consistency、execution consistency、chain-view consistency。

TODO:

- TODO: 从原始结果文件提取并填入操作系统、Geth 版本、硬件环境、编译环境等复现实验配置。

### Results

#### Result 1: Upgrade latency and asynchronous preparation

问题：动态升级需要多少控制端可观测时间，链下准备是否脱离升级交易执行关键路径。

数据源：

- `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/summary.csv`
- `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/upgrade_latency_figure_data.csv`
- `cryptoupgrade/results/upgrade-latency/lab1-5nodes-other-algos-20260813-095549/<Algorithm>/result.json`

图：

- `image/lab1-upgrade-latency-availability/lab1_upgrade_latency_5nodes.*`
- 附属图：`image/lab1-upgrade-latency-availability/lab1_deployment_upgrade_cost.*`

段落 outline：

1. 实验设置和测量定义：解释各时间点和控制端可观测延迟。
2. 阶段拆分结果：用具体数值说明 receipt 先于全网可调用完成。
3. 节点完成时间离散：说明不同节点可以异步完成准备。
4. 附属部署/上传成本：只比较 Add 和 Blake2b-256，不外推到全部算法。

允许结论：

- 升级交易先获得 receipt，后续编译加载由链下任务完成。
- 编译与加载不位于该升级交易的 EVM 执行关键路径。
- 不同节点可以在不同物理时间完成 upgrade preparation。

禁止结论：

- 不写 `zero-downtime upgrade`。
- 不写升级期间其他 EVM 交易吞吐或延迟不受影响。
- 不把 `eventToAllCompleteMillis` 写成链上交易执行时间。

Claim 映射：C2、C3、C6。

#### Result 2: Execution efficiency of dynamically loaded algorithms

问题：引入动态升级能力后，单次算法调用是否仍接近 Native Precompile。

数据源：

- `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.json`
- `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.txt`

图：

- `image/lab2-execution-efficiency/lab2_execution_efficiency_latency.*`
- `image/lab2-execution-efficiency/lab2_execution_efficiency_gas_estimate.*`
- 附属图：`image/lab2-execution-efficiency/lab2_real_chain_precompile_comparison.*`

段落 outline：

1. 三方案对比设置：Upgrade、Precompile、Contract，说明 n = 100、warmup = 10。
2. Latency 结果：比较 Upgrade 与 Precompile mean latency，具体数值从 JSON 填入。
3. Gas 估算结果：比较 `gasEstimate`，强调它不是 receipt `gasUsed`。
4. Solidity 对比：只对复杂密码算法讨论 EVM 开销，不声称所有算法都显著优于 Solidity。
5. 附属真实链实验：只写 Sha256 Upgrade vs Precompile，不扩展为全部算法真实链结果。

允许结论：

- Upgrade 与 Precompile 的 `eth_call` latency 处于相近水平。
- 部分复杂密码算法的 Solidity 实现会产生较高 EVM 执行或 Gas 估算开销。

禁止结论：

- 不写 Upgrade 普遍快于 Precompile。
- 不写所有算法都优于 Solidity。
- 不写 statistically significant，除非后续补统计检验。

Claim 映射：C4、C5。

#### Result 3: Version, execution, and chain-view consistency during activation

问题：多节点异步准备后，是否能在统一 Activation Block 按相同规则切换版本并保持输出一致。

数据源：

- `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/result.json`
- `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/samples.csv`
- `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/conclusion.md`

图：

- `image/lab3-state-consistency/lab3_upgrade_state_consistency_5nodes.*`

段落 outline：

1. scheduled 与 immediate 模式定义：两种模式都有 Activation Block，不是有/无 Activation Block 对照。
2. Version consistency：scheduled 在 activation block 前后按预期切换；immediate 从升级交易生效区块起使用新版本。
3. Execution consistency：相同采样区块和输入下各节点输出一致。
4. Chain-view consistency：各节点 `head_hash`、`block_hash`、receipt 可见性与 `metadata_hash` 一致。

重要修正：

- `experiments.md` 明确正式数据没有 `stateRoot`。因此正文不得写 State Root 等式或 State Root 一致。
- `claims.md` 的 C8 evidence 中出现 state root，但正式写作必须以 `experiments.md` 为准；若必须使用 state root，标 TODO。

允许结论：

- 在 5 节点无故障私有链正式实验中，版本、输出和链视图保持一致。
- deterministic activation 在已评估场景下避免节点按本地 ready 时间独立切换版本。

禁止结论：

- 不写形式化安全证明。
- 不写任意网络规模下的一致性保证。
- 不写已验证 fail-stop。
- 不写无 Activation Block 会分叉。

Claim 映射：C7、C8。C9 仅作为设计约束。

### Discussion

章节任务：解释本文结果意味着什么，哪些条件下成立，哪些问题未测。

段落 outline：

1. Central advance：Cryptographic Coprocessor 提供了一条介于 Solidity 与 Precompiled Contract 之间的路径。
2. Evidence meaning：Lab1 说明升级准备脱离升级交易执行关键路径；Lab2 说明动态能力没有明显牺牲 Native 执行效率；Lab3 说明 Activation Block 在已评估场景下保持版本、输出和链视图一致。
3. Relation to prior work：与 Precompiled Contract、Phoenix、Polkadot runtime upgrade、EVMPatch 等机制比较差异。
4. Limitations：
   - 未测升级准备窗口内其他 EVM 交易延迟或吞吐。
   - 未做 2/10 节点正式规模对比。
   - 未采集 State Root。
   - 未注入未就绪节点或 fail-stop 场景。
   - Gas 结果依赖当前 Native 定价模型。
5. Future work：
   - 测量 `T_receipt` 到 `T_all_complete` 期间普通交易延迟和吞吐。
   - 采集 State Root。
   - 设计未就绪节点和 fail-stop 实验。
   - 扩展多规模、多共识环境和更复杂密码算法。

Claim 映射：C1-C8，C9 作为设计边界。

### Conclusion

章节任务：用短段落收束 contribution、evidence、implication 和 boundary。

段落 outline：

1. Contribution：重申 Cryptographic Coprocessor 解耦 contract-facing interface 与 Native implementation。
2. Evidence：概括三组实验支持升级延迟、执行效率和一致性。
3. Implication：说明该机制为区块链密码算法演进提供低侵入、模块化的升级路径。
4. Boundary：限定在 contract-facing cryptographic algorithms 和已评估私有链场景，不扩展到完整客户端热升级或形式化共识安全证明。

禁止：Conclusion 不引入新引用、新数据或新机制。

## 7. Claim-Evidence-Section Map

| Claim | 主要章节 | 证据来源 | 写作状态与边界 |
| --- | --- | --- | --- |
| C1 | Introduction, Design, Conclusion | 系统架构、注册与调用路径 | 可写；不得声称解耦整个客户端 |
| C2 | Introduction, Upgrade Protocol, Lab1 | 动态编译、模块加载、Upgrade Contract、Lab1 | 可写；不得写零成本或无需任何客户端修改 |
| C3 | Upgrade Protocol, Lab1, Discussion | Event Listener、后台任务、Lab1 阶段时间 | 可写；仅表示不阻塞升级交易关键路径 |
| C4 | Design, Lab2, Conclusion | Lab2 Upgrade vs Precompile latency | 可写；使用 comparable performance |
| C5 | Lab2, Discussion | Lab2 Solidity / Precompile / Upgrade 对比 | 部分可写；不得推广到所有算法 |
| C6 | Upgrade Protocol, Lab1 | Lab1 多节点完成时间 | 可写；异步仅指 preparation |
| C7 | Upgrade Protocol, Lab3 | Activation Block 设计与 Lab3 | 可写；固定表达 Asynchronous Preparation, Deterministic Activation |
| C8 | Lab3, Discussion, Conclusion | Lab3 version/output/chain-view consistency | 可写但需改写为 chain-view consistency；不得写 State Root 一致 |
| C9 | Upgrade Protocol, Discussion | 协议设计与版本检查机制 | 只写设计约束；未做 fail-stop 实验 |

## 8. Data and Figure Map

| 实验 | 论文问题 | 权威数据 | 论文图 | 主要 claims |
| --- | --- | --- | --- | --- |
| Lab1 Upgrade Latency | 升级准备是否脱离升级交易关键路径 | `cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/summary.csv`; `upgrade_latency_figure_data.csv`; 分算法 `result.json` | `image/lab1-upgrade-latency-availability/lab1_upgrade_latency_5nodes.*` | C2, C3, C6 |
| Lab1 附属部署/上传成本 | 上传交易与 Solidity 部署成本对比 | `experiments/cryptoupgrade/docs/test_result.md` | `image/lab1-upgrade-latency-availability/lab1_deployment_upgrade_cost.*` | C2 辅助 |
| Lab2 Execution Efficiency | 动态能力是否保持 Native 执行效率 | `cryptoupgrade/results/execution-efficiency/all-20260810-133058/result.json`; `result.txt` | `image/lab2-execution-efficiency/lab2_execution_efficiency_latency.*`; `lab2_execution_efficiency_gas_estimate.*` | C4, C5 |
| Lab2 附属真实链 Sha256 | 真实链 Upgrade vs Precompile | `experiments/cryptoupgrade/docs/real_chain_upgrade_vs_precompile_result.json`; `.md` | `image/lab2-execution-efficiency/lab2_real_chain_precompile_comparison.*` | C4 辅助 |
| Lab3 State Consistency | Activation Block 下版本、输出和链视图一致性 | `experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/result.json`; `samples.csv`; `conclusion.md` | `image/lab3-state-consistency/lab3_upgrade_state_consistency_5nodes.*` | C7, C8 |

## 9. TODO and Missing Inputs

1. TODO: `references.md` 当前为空。正式写 Related Work 和 Introduction 前，必须核验参考文献并同步到 `sn-bibliography.bib` 或 `references.md`。
2. TODO: 从 Lab1、Lab2、Lab3 原始 JSON/CSV 提取具体数值，不能从图片或图注补数值。
3. TODO: 补充系统架构图、调用路径图、升级时序图和 Activation Block 版本切换图。
4. TODO: 当前 `tables/` 目录为空。如需表格，应建立实验设置表、claim-evidence 表或性能汇总表。
5. TODO: 目标期刊未确定，暂按 generic Nature-leaning structure 规划；目标期刊确定后需重检 abstract 结构、字数限制和引用格式。
6. TODO: C8 中 State Root 表述与正式实验数据不一致。正文以 `experiments.md` 为准，不写 State Root 一致。
7. TODO: 未做升级准备窗口内其他 EVM 交易延迟/吞吐实验，只能在 future work 中提出。
8. TODO: 未做 fail-stop / 未就绪节点实验，C9 只能作为设计约束。

## 10. Pre-Writing Quality Gate

写每个章节前检查：

1. 该章节要支撑哪些 claims。
2. 所需证据是否在 `claims.md` 和 `experiments.md` 中允许。
3. 所有数值是否能定位到 JSON/CSV。
4. 所有引用是否已经核验。
5. 术语是否在 `terminology.md` 中有规范表达。
6. 是否有未完成图表或 TODO 会影响该章节完整性。

## 11. Post-Edit Quality Gate

修改 `sn-article-template/sn-article.tex` 后检查：

1. LaTeX 是否能编译。
2. `\cite{}` 是否都有对应 BibTeX 条目。
3. `\label{}` 和 `\ref{}` 是否一致。
4. 图表文件是否存在，图题和表题是否与正文一致。
5. 是否仍有模板示例文字、示例表格、示例图片或无关算法环境。
6. 是否存在未解释的 TODO。
7. 是否使用了禁止结论：`zero-downtime`、`no availability impact`、`formal proof`、`universally better`、`faster than precompiled contracts`。
8. 是否把 `gasEstimate` 误写为 receipt `gasUsed`。
9. 是否把 Lab3 写成 State Root 一致或有/无 Activation Block 对照。

## 12. Definition of Done

论文整体内容编写完成的最低标准：

1. `sn-article-template/sn-article.tex` 中模板正文全部替换为本文内容。
2. 每个主要结论都能映射到 C1-C9 中的至少一个 claim。
3. 每个实验数值都能追溯到 JSON/CSV。
4. Related Work 按主题综合，并与本文 contribution 建立明确区别。
5. Introduction、Abstract 和 Conclusion 不超过 `claims.md` 与 `experiments.md` 的边界。
6. 所有图表、引用、标签和术语通过最终检查。
7. 剩余 TODO 只允许是明确标注的 future work 或投稿信息，不允许是核心证据缺口。
