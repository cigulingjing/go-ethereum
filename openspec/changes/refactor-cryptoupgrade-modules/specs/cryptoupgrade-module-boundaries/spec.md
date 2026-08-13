## ADDED Requirements

### Requirement: 稳定的 Geth 集成 facade

系统 SHALL 保留 `github.com/ethereum/go-ethereum/cryptoupgrade` 根 package 作为 Geth 主流程访问动态密码升级能力的稳定 facade，具体实现 SHALL 位于职责明确的内部模块。

#### Scenario: Geth 调用 cryptoupgrade
- **WHEN** `core/vm`、`node` 或 `cmd/geth` 调用 CodeStorage、precompile、event binding、算法加载或元数据持久化能力
- **THEN** 调用方 SHALL 只导入 `github.com/ethereum/go-ethereum/cryptoupgrade`
- **AND** SHALL NOT 直接导入实验部署目录或 `cryptoupgrade/internal` package

#### Scenario: 迁移根目录实现
- **WHEN** 根目录实现迁入内部模块
- **THEN** facade SHALL 保持迁移前已被 Geth 使用的导出 API 和执行语义
- **AND** 根目录 SHALL NOT 新增包含实验编排或部署实现的文件

### Requirement: 单向运行时模块依赖

系统 SHALL 将 EVM/event 适配、升级编排、源码制品编码、调用编码、编译、plugin 运行时及持久化划分为单一职责模块，并保持从适配层到编排层再到基础能力层的单向依赖。

#### Scenario: 处理 codeUploaded event
- **WHEN** event 模块收到并解析有效的 `codeUploaded` event
- **THEN** event 模块 SHALL 将激活请求交给升级编排模块
- **AND** event 模块 SHALL NOT 直接实现源码解码、plugin 编译或元数据持久化

#### Scenario: 编排算法激活
- **WHEN** 升级编排模块激活已上传算法
- **THEN** 编排模块 SHALL 依次协调源码解码、plugin 编译、元数据持久化及 plugin 加载
- **AND** 基础能力模块 SHALL NOT 反向依赖 EVM 或 event 适配模块

#### Scenario: EVM 交易上传源码
- **WHEN** `uploadCode` 在 EVM 交易路径完成源码和元数据登记
- **THEN** 交易路径 SHALL NOT 同步执行 Go plugin 编译
- **AND** upload receipt SHALL NOT 被视为本地算法激活完成

### Requirement: 区分源码制品编码与 EVM 调用编码

系统 SHALL 分别提供源码制品 codec 和 EVM 调用 codec，二者 SHALL 使用独立 package 和可测试接口。

#### Scenario: 打包 Go 候选源码
- **WHEN** benchmark、升级工具或 smoke test 需要上传 Go 候选源码
- **THEN** 调用方 SHALL 使用共享源码制品 codec 生成兼容的 gzip/base64 文本
- **AND** SHALL NOT 在各命令中维护重复的文件压缩实现

#### Scenario: 编解码 callFunc 参数
- **WHEN** 运行时解析 `callFunc` 输入或封装算法输出
- **THEN** 系统 SHALL 使用独立的 EVM 调用 codec
- **AND** 调用 codec SHALL NOT 负责源码文件压缩、解压或文件路径管理

### Requirement: 编译与持久化职责分离

系统 SHALL 由 compiler 模块负责从明确的源码输入生成 Go plugin 制品，并由 repository 模块负责 workspace 路径、制品位置和算法元数据持久化。

#### Scenario: 编译 Go plugin
- **WHEN** compiler 收到源码路径、输出路径和构建上下文
- **THEN** compiler SHALL 执行兼容的 `go build -buildmode=plugin`
- **AND** SHALL 校验构建结果并以原子方式发布目标制品
- **AND** SHALL NOT 自行决定全局 plugin 根目录或修改算法元数据 schema

#### Scenario: 解析运行时 plugin 目录
- **WHEN** repository 解析算法源码、`.so` 制品和元数据路径
- **THEN** 未设置环境变量时 SHALL 继续使用 `plugin/src`、`plugin/so` 和 `plugin/algorithm_info.json`
- **AND** 设置 `GETH_CRYPTOUPGRADE_PLUGIN_DIR` 时 SHALL 保持既有覆盖语义

### Requirement: 算法资产与 builtin 实现分离

系统 SHALL 区分作为动态升级输入的算法源码资产和编译进节点的 builtin 算法实现。

#### Scenario: 新增动态候选算法
- **WHEN** 新增用于上传、编译和替换的 Go 候选源码或 Solidity 对照合约
- **THEN** 资产 SHALL 位于 `cryptoupgrade/algorithm` 对应目录
- **AND** 运行时 package SHALL NOT 将候选源码目录作为静态实现依赖

#### Scenario: 新增编译进节点的算法
- **WHEN** 新增由节点二进制直接注册和调用的算法实现
- **THEN** 实现 SHALL 位于 `cryptoupgrade/builtin`
- **AND** EVM 适配层 SHALL 只负责 registry、ABI 和 gas 边界

### Requirement: 实验部署层独立

系统 SHALL 将 cryptoupgrade 的 benchmark、网络工具、Docker、YAML、实验文档及结果放置在仓库级 `experiments/cryptoupgrade`，并禁止运行时反向依赖实验层。

#### Scenario: 运行论文实验
- **WHEN** 用户运行 benchmark、多节点网络工具或部署脚本
- **THEN** 实验入口 SHALL 位于 `experiments/cryptoupgrade`
- **AND** 实验代码 MAY 通过根 facade 调用 cryptoupgrade 能力

#### Scenario: 构建 Geth 运行时
- **WHEN** 构建或测试 `cryptoupgrade` 运行时 package
- **THEN** `cryptoupgrade` SHALL NOT 导入 `experiments/cryptoupgrade`
- **AND** 运行时构建 SHALL NOT 依赖 Docker、YAML 配置或历史实验结果

#### Scenario: 写入新实验结果
- **WHEN** 实验未显式指定输出目录
- **THEN** 新结果 SHALL 写入 `experiments/cryptoupgrade/results/<experiment>/<run-id>`
- **AND** SHALL NOT 覆盖运行时 plugin、算法源码或网络配置

### Requirement: 网络验证与密码升级 smoke 分离

系统 SHALL 将网络健康验证和动态密码升级端到端 smoke test 实现为可独立运行、可组合的实验模块。

#### Scenario: 验证实验网络
- **WHEN** 仅执行 network 验证
- **THEN** 系统 SHALL 检查配置、RPC、chain ID、peer、出块及容器状态
- **AND** network 模块 SHALL NOT 上传算法或依赖 CodeStorage ABI

#### Scenario: 验证动态升级链路
- **WHEN** 执行 cryptoupgrade smoke test
- **THEN** smoke 模块 SHALL 负责算法源码上传、等待本地激活并调用算法验证结果
- **AND** smoke 模块 SHALL 可在网络验证成功后按需组合执行

### Requirement: 可验证的分阶段迁移

系统 SHALL 分阶段迁移模块和目录，更新项目内引用并提供自动检查，且 SHALL 保持历史实验数据只读兼容。

#### Scenario: 更新项目内路径
- **WHEN** package、命令、Docker、YAML、文档或结果目录被迁移
- **THEN** 项目内 Go import、脚本、fixture、默认路径和文档示例 SHALL 指向新位置
- **AND** SHALL 记录影响实验复现的旧新路径映射

#### Scenario: 检查模块边界
- **WHEN** 运行布局和依赖检查
- **THEN** 检查 SHALL 识别运行时对实验目录的反向依赖、实验资产回流 `cryptoupgrade` 根目录及重复源码压缩实现

#### Scenario: 处理历史实验结果
- **WHEN** 迁移已有 JSON、CSV、日志、图表或生成源码
- **THEN** 系统 SHALL 保留原始内容
- **AND** SHALL NOT 自动改写结果中记录的历史绝对路径或测量值
