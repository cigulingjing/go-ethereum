## ADDED Requirements

### Requirement: 独立候选 plugin 源码

系统 SHALL 在 `cryptoupgrade/algorithm` 下提供独立密码算法候选源码文件，用于表示可改造为单文件纯 Go plugin 的 blockchain-crypto 类别。

#### Scenario: 候选源码布局

- **WHEN** 增加候选算法
- **THEN** 其源码文件 SHALL 使用 `package main`
- **AND** SHALL 暴露至少一个导出的算法入口
- **AND** SHALL NOT import `/home/liuqi/project/blockchain-crypto` 中的包

#### Scenario: 覆盖实际可行类别

- **WHEN** 某个 blockchain-crypto 类别具有实际可行的单文件纯 Go 代表实现
- **THEN** 系统 SHALL 在 `cryptoupgrade/algorithm` 中为该类别包含一个候选源码文件

### Requirement: plugin 构建验证

候选算法源码文件 SHALL 使用 cryptoupgrade plugin 构建命令验证。

#### Scenario: 候选构建成功

- **WHEN** 验证候选源码文件
- **THEN** `go build -buildmode=plugin -tags=urfave_cli_no_docs,ckzg -trimpath -o <outputPath> <srcpath>` SHALL 成功完成

#### Scenario: 构建输出处理

- **WHEN** plugin 验证创建构建制品
- **THEN** 这些制品 SHALL 写入已提交源码路径之外，或以其他方式从提交中排除

### Requirement: 跳过类别报告

系统 SHALL 报告未转换的 blockchain-crypto 类别，原因是单文件纯 Go plugin 改造不实际可行。

#### Scenario: 跳过复杂类别

- **WHEN** 某个类别依赖外部可执行文件、CGO、Rust、生成的协议类型或大型多包曲线/证明栈
- **THEN** 该类别 SHALL 被报告为跳过，并附带简明原因

#### Scenario: 源仓库隔离

- **WHEN** 生成候选或跳过类别
- **THEN** `/home/liuqi/project/blockchain-crypto` 下的文件 SHALL 保持未修改

### Requirement: Solidity 候选对照

系统 SHALL 为可在不新增依赖的前提下实现为确定性或可验证合约逻辑的候选操作提供 Solidity 对照。

#### Scenario: Solidity 对照布局

- **WHEN** 增加 Solidity 对照
- **THEN** 其源码文件 SHALL 放在 `cryptoupgrade/contracts` 下
- **AND** SHALL 能使用 Solidity 0.8.26 编译

#### Scenario: 不实际可行的 Solidity 对照

- **WHEN** Go 候选依赖安全随机数、AES/Ed25519 primitives，或不适合在紧凑 Solidity 合约中实现的 field arithmetic
- **THEN** Solidity 对照 SHALL 报告被跳过的操作，并附带简明原因
