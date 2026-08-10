## ADDED Requirements

### Requirement: 统一算法源码目录

系统 SHALL 将候选算法 Go 源码放置在 `cryptoupgrade/algorithm/go`，将没有 Solidity 对照的 Go-only 源码放置在 `cryptoupgrade/algorithm/go/archive`，并将 Solidity 算法合约放置在 `cryptoupgrade/algorithm/contracts`。

#### Scenario: Go 算法源码目录

- **WHEN** 新增或维护 Go 候选算法源码
- **THEN** 源码文件 SHALL 位于 `cryptoupgrade/algorithm/go`
- **AND** SHALL NOT 再放置在 `cryptoupgrade/algorithm` 根目录

#### Scenario: Go-only 算法归档目录

- **WHEN** Go 候选算法当前没有 Solidity 对照
- **THEN** 源码文件 SHALL 位于 `cryptoupgrade/algorithm/go/archive`
- **AND** SHALL NOT 放置在 `cryptoupgrade/algorithm/go` 根目录

#### Scenario: Solidity 算法合约目录

- **WHEN** 新增或维护 Solidity 算法合约
- **THEN** 合约文件 SHALL 位于 `cryptoupgrade/algorithm/contracts`
- **AND** SHALL NOT 再放置在 `cryptoupgrade/contracts`

### Requirement: 单一暴露入口

系统 SHALL 为每个算法源码文件保留唯一的对外算法入口。

#### Scenario: Go 文件唯一导出入口

- **WHEN** 检查 `cryptoupgrade/algorithm/go` 下的候选 Go 文件
- **THEN** 每个文件 SHALL 只包含一个导出的算法函数
- **AND** helper 函数 SHALL 使用非导出名称

#### Scenario: Solidity 合约唯一外部入口

- **WHEN** 检查 `cryptoupgrade/algorithm/contracts` 下的可部署算法合约
- **THEN** 每个合约 SHALL 只包含一个 external 或 public 算法函数
- **AND** helper 函数 SHALL 使用 internal 或 private 可见性

### Requirement: Go 与 Solidity 命名一致

系统 SHALL 让 Solidity 算法文件、contract name 和对应 Go 文件保持稳定映射。

#### Scenario: 文件名映射

- **WHEN** Go 文件名是 `add.go`
- **THEN** 对应 Solidity 文件 SHALL 命名为 `Add.sol`
- **AND** 对应 contract name SHALL 使用稳定名称
- **AND** 如果 Solidity 外部函数名需要使用 `Add`，contract name SHALL NOT 与该函数名冲突

#### Scenario: 入口名映射

- **WHEN** Go 算法文件有 Solidity 对照
- **THEN** Solidity 外部算法函数 SHALL 与 Go 导出入口使用相同名称

### Requirement: 困难算法只保留 Go 实现

系统 SHALL 对当前未实现或实现困难的 Solidity 算法只保留 Go 文件，不创建占位 Solidity 算法合约。

#### Scenario: 无 Solidity 对照的算法

- **WHEN** 算法依赖安全随机数、AES/Ed25519 primitive 或复杂 field arithmetic，且当前没有实际可行 Solidity 对照
- **THEN** 系统 SHALL 在 `cryptoupgrade/algorithm/go/archive` 保留 Go 算法文件
- **AND** SHALL NOT 为该算法保留占位 Solidity 合约

### Requirement: 算法一致性检查

系统 SHALL 提供静态检查来统计 Go 算法和 Solidity 算法是否匹配。

#### Scenario: 输出匹配统计

- **WHEN** 运行一致性检查
- **THEN** 检查 SHALL 输出 Go 算法入口、Solidity 算法入口、匹配项和仅 Go 保留项
- **AND** 对 Solidity 已实现算法 SHALL 标记操作逻辑是否与 Go 入口一致
