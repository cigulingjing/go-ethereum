# cryptoupgrade 性能基准规范

## Purpose

定义智能合约升级方案的性能基准实验，用于比较升级方案、Solidity 合约实现和预编译合约实现之间的部署资源与调用资源差异。

## Requirements

### Requirement: 部署资源基准

系统 SHALL 统计算法部署阶段的资源消耗。

#### Scenario: 升级方案部署

- **WHEN** 算法通过 `CodeStorage.uploadCode` 上传并激活
- **THEN** 基准工具 SHALL 记录从 `eth_sendTransaction` 到 receipt 返回的耗时
- **AND** SHALL 记录 receipt 中的 gas used

#### Scenario: Solidity 合约部署

- **WHEN** 等价密码算法通过 Solidity 合约部署
- **THEN** 基准工具 SHALL 记录部署耗时和 receipt gas used

#### Scenario: 预编译合约适配部署

- **WHEN** 等价密码算法通过预编译合约适配器部署
- **THEN** 基准工具 SHALL 记录部署耗时和 receipt gas used

### Requirement: 已部署调用资源基准

系统 SHALL 统计算法完成部署后的调用资源消耗。

#### Scenario: 调用耗时统计

- **WHEN** 算法或合约已经部署完成
- **THEN** 基准工具 SHALL 通过 RPC 进入调用路径并统计 `eth_call` 的 first、mean、p50、p95、min、max 耗时
- **AND** SHALL 对返回值做一致性校验

#### Scenario: 调用 gas 统计

- **WHEN** 算法或合约已经部署完成
- **THEN** 基准工具 SHOULD 通过 `eth_estimateGas` 或等价方式记录调用 gas
- **AND** SHOULD 在结果文档中与调用耗时一起呈现

### Requirement: 三组密码算法对照

系统 SHALL 支持同一密码算法的三组实验对照。

#### Scenario: Solidity 纯合约实现

- **WHEN** 选择 Solidity 对照组
- **THEN** 密码算法 SHALL 由 Solidity 合约实现
- **AND** 不应通过预编译合约或升级方案绕过算法主体执行

#### Scenario: 升级方案实现

- **WHEN** 选择升级方案对照组
- **THEN** 密码算法 SHALL 通过 Go plugin 动态编译、上传和调用
- **AND** 可以与预编译合约实现复用同一 Go 算法文件或同一 Go 算法库

#### Scenario: 预编译合约实现

- **WHEN** 选择预编译合约对照组
- **THEN** 密码算法 SHALL 在 geth 客户端侧以 native precompile 路径执行
- **AND** 调用接口 SHALL 与其他对照组保持语义等价

### Requirement: 可复现实验结果文档

系统 SHALL 保留可复现实验结果文档。

#### Scenario: 记录结果表

- **WHEN** 完成一轮基准实验
- **THEN** 文档 SHALL 记录算法、实现方式、输入、部署耗时、部署 gas、调用耗时和比值
- **AND** SHOULD 记录调用 gas，如果该指标已经采集

#### Scenario: 记录验证方式

- **WHEN** 更新基准实验代码或结果文档
- **THEN** 文档 SHALL 记录验证命令、测试输入和期望返回值
