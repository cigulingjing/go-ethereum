## ADDED Requirements

### Requirement: 稳定运行时根包

系统 SHALL 保留 `github.com/ethereum/go-ethereum/cryptoupgrade` 作为 geth 调用 cryptoupgrade 运行时能力的稳定 Go package 入口。

#### Scenario: geth 集成代码导入 cryptoupgrade

- **WHEN** `core/vm`、`node` 或 `cmd/geth` 需要调用 cryptoupgrade 的 CodeStorage、precompile、event binding 或算法调用能力
- **THEN** 代码 SHALL 继续通过 `github.com/ethereum/go-ethereum/cryptoupgrade` 根包导入运行时入口
- **AND** SHALL NOT 为目录整理而要求 geth 主流程改为导入实验目录

#### Scenario: 新增运行时文件

- **WHEN** 新增与 EVM 调用、CodeStorage ABI、plugin 加载、precompile 注册或持久化算法元数据直接相关的 Go 文件
- **THEN** 文件 SHALL 位于 `cryptoupgrade/` 根包或该根包明确导入的运行时子包
- **AND** SHALL NOT 位于 benchmark、docs、examples 或 results 目录

### Requirement: 算法资产目录边界

系统 SHALL 使用 `cryptoupgrade/algorithm` 存放可复用的算法源码资产，并保持 Go plugin 候选与 Solidity 对照合约的既有分层。

#### Scenario: Go plugin 候选算法

- **WHEN** 新增或维护可作为动态升级输入的 Go plugin 候选源码
- **THEN** 源码 SHALL 位于 `cryptoupgrade/algorithm/go`
- **AND** 无 Solidity 对照的 Go-only 算法 SHALL 位于 `cryptoupgrade/algorithm/go/archive`

#### Scenario: Solidity 对照算法

- **WHEN** 新增或维护用于性能对照的 Solidity 算法合约
- **THEN** 合约 SHALL 位于 `cryptoupgrade/algorithm/contracts`
- **AND** SHALL NOT 与运行时根包文件或实验命令文件混放

#### Scenario: 算法一致性检查

- **WHEN** 运行算法接口或目录布局检查
- **THEN** 检查 SHALL 以 `cryptoupgrade/algorithm/go`、`cryptoupgrade/algorithm/go/archive` 和 `cryptoupgrade/algorithm/contracts` 作为算法资产输入目录

### Requirement: 实验命令目录边界

系统 SHALL 将 benchmark、升级流程和实验网络辅助命令放置在实验工具目录下，而不是与运行时根包混放。

#### Scenario: benchmark 命令入口

- **WHEN** 新增或维护测量部署耗时、调用耗时、gas、升级延迟或多方案对照的 Go `main` 命令
- **THEN** 命令 SHALL 位于 `cryptoupgrade/bench/cmd/<command>`
- **AND** 命令默认参数 SHALL 指向规范化的算法、示例和结果目录

#### Scenario: 多节点实验辅助命令

- **WHEN** 新增或维护用于渲染、初始化或验证 cryptoupgrade 实验网络的 Go `main` 命令
- **THEN** 命令 SHALL 位于 `cryptoupgrade/bench/cmd/<command>`
- **AND** 可复用库逻辑 SHALL 保留在 `cryptoupgrade/network`

### Requirement: 实验配置与结果目录边界

系统 SHALL 区分可复现实验输入、实验输出结果和操作文档。

#### Scenario: 可复现实验配置

- **WHEN** 新增多节点网络 YAML、测试账户、静态输入样例或可复现实验配置
- **THEN** 文件 SHALL 位于 `cryptoupgrade/examples`
- **AND** SHALL NOT 写入 `cryptoupgrade/results`

#### Scenario: 实验结果输出

- **WHEN** benchmark 或升级延迟实验产生 JSON、CSV、日志、图表或临时生成源码
- **THEN** 输出 SHALL 位于 `cryptoupgrade/results/<experiment>/<run-id>` 或调用方显式配置的输出目录
- **AND** 默认输出目录 SHALL NOT 覆盖算法源码、网络配置或运行时 plugin 制品目录

#### Scenario: 实验文档

- **WHEN** 更新实验运行步骤、结果解读或环境说明
- **THEN** 文档 SHALL 位于 `cryptoupgrade/docs`
- **AND** 文档 SHALL 使用当前规范目录中的命令路径和默认输入路径

### Requirement: 运行时 plugin 目录语义保持不变

系统 SHALL 在整理仓库源码目录时保持节点本地运行时 plugin 制品路径语义不变。

#### Scenario: 默认运行时 plugin 路径

- **WHEN** 未配置 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`
- **THEN** 运行时源码、已编译 plugin 和算法元数据 SHALL 继续解析到进程启动目录下的 `plugin/src`、`plugin/so` 和 `plugin/algorithm_info.json`

#### Scenario: 配置运行时 plugin 路径

- **WHEN** 已配置 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`
- **THEN** 运行时 plugin 路径 SHALL 继续按现有插件目录管理规范解析
- **AND** 源码布局整理 SHALL NOT 改变已持久化算法元数据 schema

### Requirement: 目录迁移兼容性

系统 SHALL 在源码目录整理后更新所有项目内路径引用，并提供验证方式防止新增文件回退到旧布局。

#### Scenario: 项目内引用更新

- **WHEN** 文件或命令目录被移动到规范化路径
- **THEN** Go import、`go run` 路径、测试 fixture、shell 脚本和文档示例 SHALL 指向新路径
- **AND** 迁移后仓库内 SHALL NOT 保留指向旧实验命令目录的默认命令示例

#### Scenario: 布局检查

- **WHEN** 运行目录布局检查
- **THEN** 检查 SHALL 标记新增的 `cryptoupgrade/cmd/*` 实验命令、根目录实验结果文件或偏离规范的默认路径
- **AND** 检查 SHALL 对历史 `cryptoupgrade/results` 数据保持只读兼容
