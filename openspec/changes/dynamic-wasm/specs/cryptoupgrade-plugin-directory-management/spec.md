## MODIFIED Requirements

### Requirement: 集中式插件制品路径

cryptoupgrade 运行时 SHALL 从同一个已解析的基础目录推导 WASM bytecode 路径、编译缓存路径和算法元数据路径。

#### Scenario: 默认插件路径来自同一个基础目录

- **WHEN** 未配置插件目录覆盖值
- **THEN** WASM bytecode 目录 SHALL 解析到进程启动目录下的 `plugin/wasm`
- **AND** 编译缓存目录 SHALL 解析到进程启动目录下的 `plugin/compiled`
- **AND** 元数据文件 SHALL 解析到进程启动目录下的 `plugin/algorithm_info.json`

#### Scenario: 算法专属路径使用集中式解析器

- **WHEN** 运行时需要算法 `Sha256` 版本 `1` 的路径
- **THEN** WASM 路径 SHALL 推导为 `<base>/wasm/Sha256-1.wasm`
- **AND** 编译缓存路径 SHALL 推导为 `<base>/compiled/Sha256-1`
- **AND** 系统 MUST NOT 再把激活路径解析为 `<base>/src/Sha256.go` 或 `<base>/so/Sha256.so`

### Requirement: 可配置的插件基础目录

cryptoupgrade 运行时 SHALL 支持显式的进程级插件基础目录覆盖，同时保持 `./plugin` 作为默认值。

#### Scenario: 配置了覆盖目录

- **WHEN** 已配置的插件基础目录是 `/tmp/geth-node-a/cryptoupgrade-plugin`
- **THEN** 运行时 SHALL 使用 `/tmp/geth-node-a/cryptoupgrade-plugin/wasm`、`/tmp/geth-node-a/cryptoupgrade-plugin/compiled` 和 `/tmp/geth-node-a/cryptoupgrade-plugin/algorithm_info.json`

#### Scenario: 配置了相对覆盖目录

- **WHEN** 已配置的插件基础目录是相对路径
- **THEN** 运行时 SHALL 先将其基于进程启动工作目录解析为路径，再推导子路径

### Requirement: 目录初始化错误处理

cryptoupgrade 运行时 SHALL 在写入 WASM bytecode、编译缓存或元数据文件之前创建必需的制品目录，并 SHALL 将任何目录创建失败返回给调用方。

#### Scenario: 目录创建成功

- **WHEN** 激活算法或存储算法元数据
- **THEN** 运行时 SHALL 在写入文件前确保 WASM 目录和编译缓存目录存在

#### Scenario: 目录创建失败

- **WHEN** WASM 目录或编译缓存目录无法创建
- **THEN** 算法激活或元数据存储 SHALL 失败，并返回能够标识失败目录操作的错误

### Requirement: 向后兼容的元数据加载

cryptoupgrade 运行时 SHALL 从已解析的基础目录加载 `algorithm_info.json`。WASM 路径切换 MUST NOT 把缺失旧 Go plugin 源码视为启动失败。

#### Scenario: 默认元数据已经存在

- **WHEN** 未配置插件目录覆盖值，并且 `./plugin/algorithm_info.json` 存在
- **THEN** package 初始化 SHALL 加载已有元数据文件
- **AND** 若记录指向 Go 源码或 `.so` 而不是 WASM 模块，运行时 SHALL 不把该算法视为 prepared

#### Scenario: 元数据文件不存在

- **WHEN** 已解析的元数据文件不存在
- **THEN** package 初始化 SHALL 继续执行，且不将缺失文件视为错误

### Requirement: 执行语义保持不变

cryptoupgrade 运行时 SHALL 在改变本地制品路径的同时，保持 `CodeStorage` 调用入口、gas 查询和按区块选择版本的行为，但激活产物 MUST 为 WASM 模块。

#### Scenario: 算法激活使用新的路径解析器

- **WHEN** WASM 升级交易登记成功并且节点完成协处理器准备
- **THEN** 算法 SHALL 通过既有 `CodeStorage.callFunc` 被调用
- **AND** 本地制品 SHALL 写入 WASM 路径而不是 Go plugin 路径
