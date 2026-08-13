## ADDED Requirements

### Requirement: 多节点升级延迟实验

系统 SHALL 提供多节点动态升级延迟实验入口，用于测量从升级交易发起到所有目标节点完成本地算法升级并可调用的端到端耗时。

#### Scenario: 载入多节点实验配置

- **WHEN** 执行多节点升级延迟实验入口并提供 `add-multi-node-network` 兼容 YAML 配置
- **THEN** 实验入口 SHALL 载入归一化网络配置、节点角色、RPC URL、chain ID、consensus period 和 cryptoupgrade Add source
- **AND** SHALL 支持按节点 ID 选择参与观测的目标节点
- **AND** SHALL 在没有显式 sender 时选择第一个 signer 节点作为升级交易发送节点

#### Scenario: 前置检查不污染升级测量

- **WHEN** 实验入口在提交升级交易前执行网络前置检查
- **THEN** 实验入口 SHALL 检查目标节点 RPC 可达、chain ID 一致、peer count 达标和区块高度增长
- **AND** 前置检查耗时 SHALL NOT 计入升级延迟指标
- **AND** 前置检查 SHALL NOT 执行会上传同名算法的 cryptoupgrade smoke test

#### Scenario: 提交一次升级交易

- **WHEN** 实验轮次开始
- **THEN** 实验入口 SHALL 通过 sender 节点 RPC 向 `CodeStorage.uploadCode` 提交一次升级交易
- **AND** SHALL 记录提交开始时间、交易 hash 返回时间、发送节点、from 地址、交易 hash、提交前 latest block、payload size 和 upload calldata size
- **AND** SHALL 记录升级交易 receipt status、receipt gas used 和 receipt block number
- **AND** SHALL 计算 submit-to-tx-hash latency，用于表示 sender RPC 接受升级交易的控制端观测耗时
- **AND** SHALL 计算 tx-hash-to-sender-receipt latency，用于表示交易返回后到 sender 节点首次观测到成功 receipt 的耗时

#### Scenario: 执行多算法升级延迟实验

- **WHEN** 用户请求执行多个密码算法的升级延迟实验
- **THEN** 实验入口 SHALL 为每个算法选择对应源码、函数名、ABI 输入输出类型、测试输入和期望输出
- **AND** SHALL 至少支持 Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit 和 SchnorrVerify 中可在当前节点环境成功编译和验证的算法
- **AND** 每个算法 SHALL 单独提交升级交易并保留独立的节点级和全网级升级延迟结果

### Requirement: 节点级升级完成观测

系统 SHALL 分别观测每个目标节点看到升级区块和完成本地算法激活的时间点。

#### Scenario: 观测节点 receipt 可见时间

- **WHEN** 升级交易 hash 已返回
- **THEN** 实验入口 SHALL 并发轮询每个目标节点的 RPC
- **AND** SHALL 在每个节点首次返回该升级交易 receipt 时记录 receipt observed time、receipt block number 和当前 latest block
- **AND** SHALL 将该时间点作为该节点看到升级区块的控制端观测时间
- **AND** SHALL 记录每个节点从交易 hash 返回到 receipt 可见、从 sender receipt 到该节点 receipt 可见的耗时

#### Scenario: 使用 callFunc 判定节点完成升级

- **WHEN** 目标节点已经看到升级交易或仍在等待本地升级完成
- **THEN** 实验入口 SHALL 通过该节点 RPC 调用 `CodeStorage.callFunc(algorithm, encodedInput)` 验证升级后的算法
- **AND** 只有当返回值等于该算法 fixture 定义的期望输出时，节点 SHALL 被标记为 completed
- **AND** 实验入口 SHALL 记录节点 completed time、completion block number、validation output 和 validation call latency
- **AND** SHALL 记录每个节点从 sender receipt 到本节点 completed、从本节点 receipt 到 completed 的耗时

#### Scenario: 节点超时保留部分结果

- **WHEN** 某个目标节点在实验 timeout 前未返回 receipt 或未通过 `callFunc` 验证
- **THEN** 该节点 SHALL 被标记为 failed 或 timed out
- **AND** 该实验轮次 SHALL 标记为 incomplete
- **AND** 已观测到的其他节点阶段数据 SHALL 被保留在原始结果中

### Requirement: 全网升级耗时统计

系统 SHALL 基于节点级观测结果计算全网升级完成耗时和可用于论文分析的阶段耗时。

#### Scenario: 计算全网完成耗时

- **WHEN** 所有目标节点均已 completed
- **THEN** 实验入口 SHALL 将最晚的节点 completed time 减去升级交易 submission started time，计算 submit-to-all-complete latency
- **AND** SHALL 记录最慢节点 ID、完成节点数、目标节点数和每个节点 submit-to-complete latency
- **AND** SHALL 基于每个已观测节点的一手数据计算所有节点平均 submit-to-receipt、receipt-to-complete 和 submit-to-complete latency
- **AND** SHALL 在 round-level 汇总中保留上述平均值，便于论文实验直接引用

#### Scenario: 拆分出块同步和本地完成阶段

- **WHEN** 节点同时具有 receipt observed time 和 completed time
- **THEN** 实验入口 SHALL 计算该节点 submit-to-receipt latency、receipt-to-complete latency 和 submit-to-complete latency
- **AND** SHALL 计算全网 submit-to-all-receipt latency 和 submit-to-all-complete latency
- **AND** node-level 结果 SHALL 保留每个节点的阶段耗时，作为后续统计平均值、方差和异常节点分析的一手实验数据

#### Scenario: 拆分升级交易全流程阶段

- **WHEN** 实验轮次完成或失败并产生部分观测数据
- **THEN** 实验入口 SHALL 输出 round-level 阶段耗时，包括 submit-to-tx-hash、tx-hash-to-sender-receipt、sender-receipt-to-all-receipt、all-receipt-to-all-complete 和 submit-to-all-complete
- **AND** sender receipt SHALL 被标注为控制端观测到的“交易已被打包执行”时间点
- **AND** 结果 SHALL 明确该“共识完成”口径不是 BFT finality 或节点内部 Clique 状态

#### Scenario: 扩展到 10 节点本地 Clique 网络

- **WHEN** 用户请求在更大本地规模下复现实验
- **THEN** 实验环境 SHALL 支持 10 个 Clique 节点参与同一轮升级观测
- **AND** 每个节点 SHALL 使用独立 datadir、pluginDir、node key 和可访问 RPC 端口
- **AND** 实验结果 SHALL 同时报告全网最慢完成耗时和所有 completed 节点的平均完成耗时

#### Scenario: 多轮实验避免缓存污染

- **WHEN** 实验入口执行多轮升级延迟实验
- **THEN** 每轮 SHALL 使用不会在提交前已被目标节点成功调用的算法名
- **AND** 若提交前同名算法已经可调用且无法生成唯一算法名或清洁状态，实验入口 SHALL 拒绝该轮测量并记录污染原因

### Requirement: 多节点升级延迟结果制品

系统 SHALL 保存可复现、机器可读的多节点升级延迟实验结果。

#### Scenario: 输出 JSON 原始结果

- **WHEN** 实验轮次完成或失败
- **THEN** 实验入口 SHALL 写出 JSON 原始结果
- **AND** JSON SHALL 包含配置文件路径、chain ID、consensus period、节点列表、sender 节点、算法名、payload size、交易 hash、receipt、节点级观测、全网汇总、timeout、poll interval、完成状态和失败原因

#### Scenario: 输出 CSV 汇总

- **WHEN** 用户请求 CSV 输出或执行多轮实验
- **THEN** 实验入口 SHALL 输出 round-level 和 node-level CSV 数据
- **AND** CSV SHALL 包含足以复现实验表格的轮次编号、节点 ID、节点角色、阶段耗时、全网完成耗时、所有节点平均耗时和完成状态

#### Scenario: 记录观测限制

- **WHEN** 实验入口保存结果
- **THEN** 结果 SHALL 标注升级耗时由控制端 RPC 观测得到
- **AND** SHALL 记录 poll interval、RPC timeout、目标节点范围和未暴露 RPC 的 excluded nodes
- **AND** SHALL NOT 将节点日志时间戳伪装为跨节点精确耗时
