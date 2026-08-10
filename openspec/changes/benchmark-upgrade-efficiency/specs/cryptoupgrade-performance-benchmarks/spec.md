## ADDED Requirements

### Requirement: 升级效率单次实验

系统 SHALL 提供动态升级方案的升级效率实验入口，用于在本地 dev 链上执行一次 Add 算法升级并采集分阶段原始指标。

#### Scenario: 使用既有 dev 链启动命令

- **WHEN** 执行升级效率实验入口
- **THEN** 实验入口 SHALL 使用 `cryptoupgrade/docs/chain_setup.md` 中指定的 `build/bin/chain/./geth --dev` RPC 启动方式
- **AND** 实验入口 SHALL NOT 创建额外私有链或改写链启动配置

#### Scenario: 执行 Add-only 升级

- **WHEN** 实验入口提交升级请求
- **THEN** 实验入口 SHALL 通过 `CodeStorage.uploadCode` 提交 `cryptoupgrade/algorithm/go/add.go`
- **AND** 实验入口 SHALL NOT 批量提交其他密码算法升级

#### Scenario: 分阶段采集升级指标

- **WHEN** Add 升级交易被提交并确认
- **THEN** 实验入口 SHALL 记录 upgrade transaction hash、receipt gas used、提交时 block number 和确认 block number
- **AND** 实验入口 SHALL 分别记录 transaction submission、transaction confirmation、algorithm activation observation 和 upgraded algorithm validation 的时间点
- **AND** 实验入口 SHALL 计算 submit-to-confirm、confirm-to-activate 和 submit-to-algorithm-available latency

#### Scenario: 记录 payload size

- **WHEN** 实验入口构造 `CodeStorage.uploadCode` payload
- **THEN** 实验入口 SHALL 记录源码字节数、gzip 压缩字节数、base64 payload 字节数和完整 upload calldata 字节数

#### Scenario: 标注 activation 可观测性

- **WHEN** 当前实现无法通过 RPC 精确区分 confirmation 和 activation
- **THEN** 实验入口 SHALL 在 JSON 中标注 activation block 的来源和限制
- **AND** 实验入口 SHALL NOT 伪造独立 activation 时间或 block

#### Scenario: 验证升级后的 Add 调用

- **WHEN** Add 升级被观测为可用
- **THEN** 实验入口 SHALL 调用 `CodeStorage.callFunc("Add", encodedInput)` 验证升级后的算法
- **AND** 实验入口 SHALL 记录验证调用使用的 block number 和 ABI 编码返回结果
- **AND** 实验入口 SHALL 校验返回结果等于输入参数之和

#### Scenario: 保存原始制品

- **WHEN** 实验入口完成或失败
- **THEN** 实验入口 SHALL 保存结构化 JSON 原始结果
- **AND** 实验入口 SHALL 保存对应 Geth 日志路径
