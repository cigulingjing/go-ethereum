# Cryptoupgrade Performance Benchmarks Delta

## ADDED Requirements

### Requirement: Deployment Resource Benchmark

系统 SHALL 统计算法部署阶段的资源消耗，包括部署耗时和 receipt gas used。

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

### Requirement: Deployed Call Resource Benchmark

系统 SHALL 统计算法完成部署后的调用资源消耗。

#### Scenario: 调用耗时统计

- **WHEN** 算法或合约已经部署完成
- **THEN** 基准工具 SHALL 通过 RPC 进入调用路径并统计 `eth_call` 耗时
- **AND** SHALL 对返回值做一致性校验

#### Scenario: 调用 gas 统计

- **WHEN** 算法或合约已经部署完成
- **THEN** 基准工具 SHOULD 通过 `eth_estimateGas` 或等价方式记录调用 gas
- **AND** SHOULD 在结果文档中与调用耗时一起呈现

### Requirement: Three-Way Cryptographic Algorithm Comparison

系统 SHALL 支持同一密码算法的三组实验对照。

#### Scenario: Solidity 纯合约实现

- **WHEN** 选择 Solidity 对照组
- **THEN** 密码算法 SHALL 由 Solidity 合约实现

#### Scenario: 升级方案实现

- **WHEN** 选择升级方案对照组
- **THEN** 密码算法 SHALL 通过 Go plugin 动态编译、上传和调用

#### Scenario: 预编译合约实现

- **WHEN** 选择预编译合约对照组
- **THEN** 密码算法 SHALL 在 geth 客户端侧以 native precompile 路径执行
