## MODIFIED Requirements

### Requirement: 部署资源基准
系统 SHALL 统计算法部署阶段的资源消耗，并 SHALL 区分升级交易确认与节点本地事件驱动激活完成。

#### Scenario: 升级方案部署
- **WHEN** 算法通过 `CodeStorage.uploadCode` 上传
- **THEN** 基准工具 SHALL 记录从 `eth_sendTransaction` 到 receipt 返回的耗时
- **AND** SHALL 记录 receipt 中的 gas used
- **AND** SHALL NOT 将 upload receipt 返回视为算法已经完成本节点本地激活
- **AND** SHALL 继续观测直到目标节点通过事件监听线程完成激活并可通过 `CodeStorage.callFunc` 验证

#### Scenario: Solidity 合约部署
- **WHEN** 等价密码算法通过 Solidity 合约部署
- **THEN** 基准工具 SHALL 记录部署耗时和 receipt gas used

#### Scenario: 预编译合约适配部署
- **WHEN** 等价密码算法通过预编译合约适配器部署
- **THEN** 基准工具 SHALL 记录部署耗时和 receipt gas used
