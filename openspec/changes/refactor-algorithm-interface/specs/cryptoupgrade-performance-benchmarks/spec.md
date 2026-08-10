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
- **AND** Go plugin 源码 SHALL 从 `cryptoupgrade/algorithm/go` 中选择

#### Scenario: 预编译合约实现

- **WHEN** 选择预编译合约对照组
- **THEN** 密码算法 SHALL 在 geth 客户端侧以 native precompile 路径执行
- **AND** 调用接口 SHALL 与其他对照组保持语义等价

#### Scenario: Solidity 候选算法对照

- **WHEN** 准备 Solidity 纯合约对照组的候选算法
- **THEN** 可实现的候选算法 SHALL 使用 `cryptoupgrade/algorithm/contracts` 下的源码文件
- **AND** Solidity 文件名 SHALL 与对应 Go 文件的算法名保持一致
- **AND** Solidity contract name SHALL 使用稳定名称并避免与外部函数名冲突
- **AND** 对无法合理转换为 Solidity 的候选算法 SHALL 不创建占位 Solidity 合约
