## ADDED Requirements

### Requirement: 集中式插件制品路径

cryptoupgrade 运行时 SHALL 从同一个已解析的插件基础目录推导算法源码路径、已编译 plugin 路径和算法元数据路径。

#### Scenario: 默认插件路径来自同一个基础目录

- **WHEN** 未配置插件目录覆盖值
- **THEN** 源码目录 SHALL 解析到进程启动目录下的 `plugin/src`
- **AND** 已编译 plugin 目录 SHALL 解析到进程启动目录下的 `plugin/so`
- **AND** 元数据文件 SHALL 解析到进程启动目录下的 `plugin/algorithm_info.json`

#### Scenario: 算法专属路径使用集中式解析器

- **WHEN** 运行时需要算法 `Sha256` 的路径
- **THEN** 源码路径 SHALL 推导为 `<base>/src/Sha256.go`
- **AND** 已编译 plugin 路径 SHALL 推导为 `<base>/so/Sha256.so`

### Requirement: 可配置的插件基础目录

cryptoupgrade 运行时 SHALL 支持显式的进程级插件基础目录覆盖，同时保持 `./plugin` 作为默认值。

#### Scenario: 配置了覆盖目录

- **WHEN** 已配置的插件基础目录是 `/tmp/geth-node-a/cryptoupgrade-plugin`
- **THEN** 运行时 SHALL 使用 `/tmp/geth-node-a/cryptoupgrade-plugin/src`、`/tmp/geth-node-a/cryptoupgrade-plugin/so` 和 `/tmp/geth-node-a/cryptoupgrade-plugin/algorithm_info.json`

#### Scenario: 配置了相对覆盖目录

- **WHEN** 已配置的插件基础目录是相对路径
- **THEN** 运行时 SHALL 先将其基于进程启动工作目录解析为路径，再推导子路径

### Requirement: 目录初始化错误处理

cryptoupgrade 运行时 SHALL 在写入源码、已编译 plugin 或元数据文件之前创建必需的插件制品目录，并 SHALL 将任何目录创建失败返回给调用方。

#### Scenario: 目录创建成功

- **WHEN** 激活算法或存储算法元数据
- **THEN** 运行时 SHALL 在写入文件前确保源码目录和已编译 plugin 目录存在

#### Scenario: 目录创建失败

- **WHEN** 源码目录或已编译 plugin 目录无法创建
- **THEN** 算法激活或元数据存储 SHALL 失败，并返回能够标识失败目录操作的错误

### Requirement: 向后兼容的元数据加载

cryptoupgrade 运行时 SHALL 从已解析的插件基础目录加载 `algorithm_info.json`，且不改变持久化元数据 schema。

#### Scenario: 默认元数据已经存在

- **WHEN** 未配置插件目录覆盖值，并且 `./plugin/algorithm_info.json` 存在
- **THEN** package 初始化 SHALL 使用当前 JSON schema 加载已有元数据文件

#### Scenario: 元数据文件不存在

- **WHEN** 已解析的元数据文件不存在
- **THEN** package 初始化 SHALL 继续执行，且不将缺失文件视为错误

### Requirement: 执行语义保持不变

cryptoupgrade 运行时 SHALL 在仅改变路径管理的同时，保持算法上传、编译、加载、调用、ABI 打包和 gas 行为不变。

#### Scenario: 算法激活使用新的路径解析器

- **WHEN** 算法上传成功
- **THEN** 算法 SHALL 仍通过既有 `CodeStorage` 行为完成解压、编译、注册、持久化，并可被调用
