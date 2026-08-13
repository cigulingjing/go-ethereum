## ADDED Requirements

### Requirement: 基准工具路径跟随源码布局

系统 SHALL 让 cryptoupgrade benchmark 工具入口、默认输入路径和默认输出路径跟随规范化源码布局，同时保持已有实验指标和结果字段语义不变。

#### Scenario: benchmark 命令位置

- **WHEN** 运行 cryptoupgrade 部署资源、调用资源、升级效率或升级延迟 benchmark
- **THEN** 项目内命令路径 SHALL 使用 `cryptoupgrade/bench/cmd/<command>`
- **AND** 文档中的 `go run` 示例 SHALL NOT 继续使用旧的 `cryptoupgrade/cmd/<command>` 路径

#### Scenario: 默认算法输入路径

- **WHEN** benchmark 工具使用默认算法源码或 Solidity 合约路径
- **THEN** Go plugin 源码 SHALL 从 `cryptoupgrade/algorithm/go` 选择
- **AND** Solidity 合约源码 SHALL 从 `cryptoupgrade/algorithm/contracts` 选择
- **AND** 工具 SHALL 保持调用方通过 flag 显式覆盖输入路径的能力

#### Scenario: 默认结果输出路径

- **WHEN** benchmark 工具未显式配置输出文件或输出目录
- **THEN** 结构化结果 SHALL 写入 `cryptoupgrade/results/<experiment>/<run-id>` 或该实验已定义的 `cryptoupgrade/results/<experiment>` 默认目录
- **AND** 输出 SHALL 保持 JSON、CSV、图表或日志的既有字段语义

#### Scenario: 历史结果兼容

- **WHEN** 读取或引用目录重整前已经生成的 benchmark 结果
- **THEN** 系统 SHALL 将其中记录的旧路径视为历史运行环境快照
- **AND** SHALL NOT 要求重写历史结果文件才能运行新的 benchmark
