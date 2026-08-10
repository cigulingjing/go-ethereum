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

#### Scenario: 升级候选算法集

- **WHEN** 准备升级方案对照组的候选算法
- **THEN** 候选算法 SHALL 使用 `cryptoupgrade/algorithm` 下可通过 Go plugin 构建验证的源码文件
- **AND** 对无法单文件纯 Go 转换的密码库类型 SHALL 保留跳过说明

#### Scenario: Solidity 候选算法对照

- **WHEN** 准备 Solidity 纯合约对照组的候选算法
- **THEN** 可实现的候选算法 SHALL 使用 `cryptoupgrade/contracts` 下可通过 solc 编译的源码文件
- **AND** 对无法合理转换为 Solidity 的候选算法 SHALL 保留跳过说明
