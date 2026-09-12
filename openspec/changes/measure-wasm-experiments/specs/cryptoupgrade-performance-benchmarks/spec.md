## ADDED Requirements

### Requirement: WASM 升级阶段精细观测

系统 SHALL 在 WASM 升级延迟实验中记录控制端 RPC 观测时间、节点内事件处理阶段时间和 WASM 编译完成时间，用于区分交易打包、事件激活和模块准备成本。

#### Scenario: 记录交易 RPC 入口

- **WHEN** 实验入口提交 WASM 升级交易
- **THEN** 结果 SHALL 记录控制端开始调用 `eth_sendTransaction` 的时间
- **AND** SHALL 记录 RPC 返回交易 hash 的时间
- **AND** SHALL 记录 sender 节点首次返回成功 receipt 的时间和 receipt gas used

#### Scenario: 记录节点内激活阶段

- **WHEN** 节点收到 `codeUploaded` 或 `codeVersionUploaded` 事件并开始本地激活
- **THEN** 节点 SHALL 记录事件处理开始时间、算法名、版本、区块号和交易 hash
- **AND** 节点 SHALL 记录 WASM bytecode 持久化完成时间
- **AND** 节点 SHALL 记录 WASM 编译完成时间和本地激活完成时间
- **AND** 结果 SHALL 保留每个节点的上述阶段数据

#### Scenario: 体现交易处理与 WASM 编译并行

- **WHEN** 升级交易 receipt 已在控制端可见但目标节点仍在编译 WASM
- **THEN** 实验结果 SHALL 分别输出 receipt 可见时间和 WASM 编译完成时间
- **AND** SHALL NOT 把 receipt gas used 或交易执行耗时解释为 WASM 编译成本

### Requirement: 固定资源的规模化升级实验

系统 SHALL 支持在本地 Docker 环境中以相同容器资源限制运行多规模 Clique 私有网络，并对指定算法集合执行可复现的 WASM 升级实验。

#### Scenario: 容器资源一致

- **WHEN** 实验网络由 Docker Compose 启动
- **THEN** 每个 geth 节点容器 SHALL 使用相同的 CPU 限制
- **AND** 每个 geth 节点容器 SHALL 使用相同的内存限制
- **AND** 实验结果 SHALL 记录这些资源限制

#### Scenario: 20 节点全算法升级实验

- **WHEN** 用户运行 20 节点 WASM 升级实验
- **THEN** 系统 SHALL 搭建 20 个节点的 Clique 私有网络
- **AND** SHALL 依次对所有可用 WASM 算法提交升级交易
- **AND** SHALL 记录每个算法的升级完成时间、receipt gas used、wasmHash、activationBlock 和节点级阶段数据

#### Scenario: SchnorrProof 多规模升级实验

- **WHEN** 用户运行 SchnorrProof 规模实验
- **THEN** 系统 SHALL 分别在 5、10、20、30、40 个节点的 Clique 私有网络上执行 SchnorrProof WASM 升级
- **AND** SHALL 对每个规模记录全网完成时间、平均节点完成时间、receipt gas used 和节点级阶段数据

### Requirement: WASM-only 执行效率与资源利用率实验

系统 SHALL 支持在单节点网络中仅测试动态 WASM 运行算法的执行耗时和容器资源利用率，不统计 Gas。

#### Scenario: 单节点执行所有 WASM 算法

- **WHEN** 用户运行单节点 WASM 执行效率实验
- **THEN** 系统 SHALL 准备每个可用 WASM 算法并通过 `CodeStorage.callFunc` 调用
- **AND** SHALL 对每个算法记录 warmup、样本数、first、mean、p50、p95、min 和 max 调用耗时
- **AND** SHALL 校验返回值等于 fixture 期望输出
- **AND** SHALL NOT 输出 Lab2 Gas 统计作为本实验结论

#### Scenario: 记录容器 CPU 与内存利用率

- **WHEN** WASM-only 执行效率实验采集调用样本
- **THEN** 系统 SHALL 记录目标节点容器的 CPU 利用率样本
- **AND** SHALL 记录目标节点容器的内存使用量和内存限制
- **AND** SHALL 在算法级结果中输出 CPU 与内存利用率的 summary

### Requirement: WASM 实验执行报告与论文审计

系统 SHALL 在三个实验完成后输出任务执行报告，并审计实验数据是否足以支撑论文实验内容。

#### Scenario: 输出任务执行报告

- **WHEN** 三个实验执行完成或部分失败
- **THEN** 系统 SHALL 输出包含命令、网络规模、资源限制、结果路径、完成状态、关键指标和失败原因的报告
- **AND** 报告 SHALL 按 20 节点全算法升级、SchnorrProof 多规模升级、单节点 WASM 执行效率的顺序组织

#### Scenario: 审计 paper outline 支撑度

- **WHEN** 实验报告生成
- **THEN** 系统 SHALL 读取 `paper/outline.md` 中实验相关内容
- **AND** SHALL 对照已采集数据评估哪些结论有数据支撑、哪些结论仍缺少样本或对照组
- **AND** SHALL 明确不能由当前实验数据推出的结论
