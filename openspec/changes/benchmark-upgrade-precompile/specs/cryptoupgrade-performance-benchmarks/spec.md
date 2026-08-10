## ADDED Requirements

### Requirement: 本地真实链 upgrade-vs-precompile benchmark

系统 SHALL 提供一个 benchmark 流程，用于在运行中的本地 geth 链上比较当前升级方案和 native precompiled contract 方案。

#### Scenario: 默认本地 geth 端点

- **WHEN** benchmark 命令未显式指定 RPC 端点
- **THEN** 它 SHALL 连接到 `http://127.0.0.1:8666`

#### Scenario: 显式本地 geth 端点

- **WHEN** benchmark 命令显式指定 RPC 端点
- **THEN** 它 SHALL 对所有链操作使用该端点
- **AND** 结果输出 SHALL 记录所用端点

#### Scenario: 仅进行二组对照

- **WHEN** 真实链 benchmark 运行
- **THEN** 它 SHALL 比较 `CodeStorage.callFunc` 升级执行和 native precompile 执行
- **AND** 本实验 SHALL NOT 要求 Solidity 合约对照

### Requirement: 真实链 setup 成本测量

系统 SHALL 测量 upgrade 路径的 setup 成本，并记录 native precompile 路径没有部署成本。

#### Scenario: upgrade setup 测量

- **WHEN** benchmark 通过 `CodeStorage.uploadCode` 上传算法
- **THEN** 它 SHALL 记录 transaction hash、从 `eth_sendTransaction` 到 receipt 的 elapsed time、receipt status、receipt gas used、source file path、source size 和 compressed payload size

#### Scenario: precompile setup 测量

- **WHEN** benchmark 选择已注册 native precompile 地址
- **THEN** 它 SHALL 记录 precompile 地址
- **AND** SHALL 将 setup transaction count 和 setup gas used 记录为 0

#### Scenario: setup 失败报告

- **WHEN** upgrade 上传在交易提交前或提交后失败
- **THEN** benchmark SHALL 返回非零退出状态
- **AND** SHALL 包含足够错误上下文，以区分 RPC failure、gas estimation failure、plugin compilation failure 和 failed receipt status

### Requirement: 真实链调用成本测量

系统 SHALL 在本地 geth 链上测量两种 benchmark 方案的重复只读调用成本。

#### Scenario: 预热后的调用延迟统计

- **WHEN** benchmark 调用 upgrade 和 precompile 路径
- **THEN** 它 SHALL 在记录样本前运行可配置数量的 warmup calls
- **AND** SHALL 为每种方案记录 first、mean、p50、p95、min 和 max latency

#### Scenario: gas estimate 统计

- **WHEN** benchmark 准备 upgrade 和 precompile call data
- **THEN** 它 SHALL 为两种方案记录 `eth_estimateGas` 结果或等价的 call gas estimate
- **AND** SHALL 在结果输出中包含这些值

#### Scenario: 比值报告

- **WHEN** 两种方案都成功完成
- **THEN** benchmark SHALL 报告 mean、p50 和 p95 的 upgrade/precompile latency ratio
- **AND** 当两个 gas 值均非零时，SHALL 报告 upgrade/precompile gas ratio

### Requirement: 真实链结果等价

系统 SHALL 校验 upgrade 和 precompile 执行对已配置算法与输入产生等价结果。

#### Scenario: 输出匹配

- **WHEN** upgrade call 和 precompile call 返回 ABI 编码输出
- **THEN** benchmark SHALL 比较解码输出或规范编码输出
- **AND** 输出匹配时 SHALL 将本次运行标记为 valid

#### Scenario: 输出不匹配

- **WHEN** upgrade call 和 precompile call 返回不同输出
- **THEN** benchmark SHALL 使本次运行失败
- **AND** SHALL 在错误消息中包含两侧输出或 output hash

### Requirement: 可复现真实链结果制品

系统 SHALL 为真实链 upgrade-vs-precompile 实验生成可复现 benchmark 输出。

#### Scenario: 机器可读结果

- **WHEN** benchmark 运行完成
- **THEN** 它 SHALL 能写入 JSON 结果，包含 endpoint、sender、algorithm name、source path、precompile address、ABI types、input、setup metrics、call metrics、gas metrics、ratios 和 validation status

#### Scenario: 人类可读摘要

- **WHEN** benchmark 运行完成
- **THEN** 它 SHALL 打印简明文本摘要，展示两种方案的 setup cost、call latency、gas estimates 和 ratios

#### Scenario: 文档化命令

- **WHEN** benchmark 代码或结果制品被更新
- **THEN** 文档 SHALL 包含针对 `http://127.0.0.1:8666` 使用的精确命令、iteration counts、warmup counts、input values 和 expected output
