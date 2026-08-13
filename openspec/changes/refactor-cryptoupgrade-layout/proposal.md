## Why

`cryptoupgrade/` 当前同时承载运行时核心代码、算法样例、实验命令、网络环境、文档和历史结果，目录边界不够清晰，后续维护和实验复现需要频繁凭经验判断文件用途。现在已有算法接口、插件目录和 benchmark 能力逐步稳定，需要整理源码布局，降低新增算法、运行实验和归档结果时的路径歧义。

## What Changes

- **BREAKING**：整理 `cryptoupgrade/` 下的源码目录结构，按运行时核心、算法资产、实验工具、网络环境、文档和结果归档划分稳定目录。
- 将 benchmark、升级流程和网络辅助命令归入清晰的实验工具层，避免 `cmd/` 下命令名称和实验目的混杂。
- 将实验输入、实验输出和示例网络配置分别归档，保证可复现实验资产与一次性结果文件边界明确。
- 更新代码引用、默认路径、文档示例和测试 fixture，使目录重整后现有 cryptoupgrade 行为保持一致。
- 保留已确认的运行时 plugin 基础目录语义，不改变节点本地 `plugin/src`、`plugin/so`、`plugin/algorithm_info.json` 的解析规则。

## Capabilities

### New Capabilities

- `cryptoupgrade-source-layout`: 定义 `cryptoupgrade/` 源码、算法资产、实验工具、网络环境、文档和结果归档的目录边界与迁移兼容要求。

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: benchmark 命令位置、默认输入路径和结果输出路径 SHALL 跟随新的目录布局，并保持既有实验指标和结果格式可复现。

## Impact

- 影响代码：`cryptoupgrade/*.go`、`cryptoupgrade/cmd/*` 到 `cryptoupgrade/bench/cmd/*` 的迁移、`cryptoupgrade/algorithm/*`、`cryptoupgrade/network/*`、测试文件和引用旧路径的工具代码。
- 影响实验资产：`cryptoupgrade/examples/*`、`cryptoupgrade/results/*`、benchmark 默认输出目录和文档中的命令示例。
- 影响 OpenSpec：新增源码布局规范，并为性能基准规范补充目录迁移后的路径要求。
- 不新增外部依赖；不改变 EVM 调用语义、ABI 编解码、gas 计费、动态 plugin 加载行为或已持久化 plugin 元数据 schema。
