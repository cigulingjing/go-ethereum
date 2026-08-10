## MODIFIED Requirements

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

#### Scenario: Solidity 单算法部署

- **WHEN** 对 Solidity 对照组统计部署 gas 和部署耗时
- **THEN** 基准工具 SHALL 部署与当前算法对应的单算法 Solidity 合约
- **AND** 不应部署包含多个无关候选算法入口的聚合合约作为该算法的部署成本

#### Scenario: Solidity 单算法调用

- **WHEN** Solidity 单算法合约已经部署完成
- **THEN** 基准工具 SHALL 调用该算法合约中的目标函数并统计调用耗时
- **AND** SHALL 通过 `eth_estimateGas` 或等价方式记录调用 gas
- **AND** SHALL 对返回值做一致性校验
