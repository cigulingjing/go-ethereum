## ADDED Requirements

### Requirement: 按算法拆分的 Solidity 合约

系统 SHALL 提供与 `cryptoupgrade/algorithm` 下实际可行 Go plugin 候选对应的按算法拆分 Solidity 候选合约。

#### Scenario: Go 候选映射

- **WHEN** Go 候选具有实际可行的 Solidity 对照
- **THEN** Solidity 对照 SHALL 实现为 `cryptoupgrade/contracts` 下可独立部署的合约
- **AND** contract name SHALL 标识其 benchmark 的算法

#### Scenario: 独立部署单元

- **WHEN** 选择 Solidity 候选进行部署 benchmark
- **THEN** 仅选定算法合约 SHALL 作为 benchmark 目标被部署
- **AND** 无关候选算法入口 SHALL NOT 包含在该可部署合约中

### Requirement: Solidity 编译

Solidity 候选合约 SHALL 使用仓库的 Solidity 编译器设置完成编译。

#### Scenario: 编译候选合约

- **WHEN** 验证 Solidity 候选
- **THEN** 每个可部署候选合约 SHALL 使用 `solc-0.8.26 --optimize --via-ir` 编译
- **AND** 编译后的 bytecode SHALL 非空

#### Scenario: 共享 helper 代码

- **WHEN** Solidity 候选需要共享算术或编码 helper
- **THEN** helper 代码 SHALL 通过 internal library 或 abstract base contract 引入
- **AND** helper SHALL NOT 强制所有算法进入同一个可部署 benchmark 合约

### Requirement: Solidity 跳过报告

系统 SHALL 报告不适合作为 Solidity benchmark 目标的 Go 候选算法。

#### Scenario: 缺少 Solidity primitive

- **WHEN** 候选需要安全随机数、AES、Ed25519，或不适合在不新增依赖时实现的大 field 运算
- **THEN** 该候选 SHALL 被列为 Solidity benchmark 跳过项
- **AND** 跳过原因 SHALL 记录在 Solidity 候选合约附近或 OpenSpec 制品中

#### Scenario: Benchmark 排除

- **WHEN** 算法被跳过 Solidity benchmark
- **THEN** benchmark 报告 SHALL NOT 像该算法已实现一样包含 Solidity 部署或调用测量
