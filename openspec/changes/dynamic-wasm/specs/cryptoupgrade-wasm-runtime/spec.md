## ADDED Requirements

### Requirement: WASM bytecode is the upgrade artifact

系统 SHALL 将 CodeStorage 升级交易中的 `code` 字段解释为 WASM bytecode 编码，而不是 Go 源码。解码后的模块 MUST 以 `\0asm` 开头。

#### Scenario: 上传合法 WASM 模块

- **WHEN** 用户通过 `uploadCodeVersion` 提交可解码为合法 WASM bytecode 的 `code`
- **THEN** 系统 SHALL 在链上登记该版本的 name、version、gas、itype、otype 和 Activation Block
- **AND** 系统 SHALL 发出 `codeVersionUploaded` 事件
- **AND** 系统 MUST NOT 在该交易的 EVM 执行路径中编译或实例化 WASM

#### Scenario: 拒绝 Go 源码或其他非 WASM 载荷

- **WHEN** 解码后的字节不以 WASM magic `\0asm` 开头
- **THEN** 协处理器准备流程 SHALL 失败
- **AND** 系统 MUST NOT 将该版本标记为 prepared

### Requirement: wasmHash identifies the broadcast module

系统 SHALL 使用 `keccak256(rawWasm)` 作为模块身份。在线节点从升级事件对应的链上 `code` 取出同一份字节码；离线恢复路径 MUST 按版本索引取出同一 `code` 并得到相同 `wasmHash`。

#### Scenario: 广播后各节点得到相同哈希

- **WHEN** 两个节点处理同一笔 `codeVersionUploaded` 交易并成功解码 WASM
- **THEN** 两节点计算的 `wasmHash` SHALL 相同
- **AND** 本地 metadata SHALL 记录该 `wasmHash`

#### Scenario: 字节码被篡改

- **WHEN** 节点本地缓存的 WASM 字节与链上 `code` 解码结果不一致
- **THEN** 系统 SHALL 拒绝将该版本标记为 prepared
- **AND** 系统 MUST NOT 执行该本地文件

### Requirement: Coprocessor compiles WASM asynchronously

系统 SHALL 由协处理器在 event/activation 路径完成 WASM 校验、编译和实例化。升级交易 receipt 不表示本地 prepared。

#### Scenario: 事件触发编译

- **WHEN** 节点监听到 `codeVersionUploaded` 并查询到对应版本 metadata
- **THEN** 系统 SHALL 解码 WASM bytecode
- **AND** 系统 SHALL 校验 magic、size limit、`execute` 导出和 forbidden host import
- **AND** 系统 SHALL 使用 WASM runtime 编译并实例化模块
- **AND** 系统 SHALL 仅在上述步骤成功后标记 prepared version

#### Scenario: 编译失败保持链上计划不变

- **WHEN** 编译超时、实例化失败或导出符号缺失
- **THEN** 系统 SHALL 记录算法名、版本号和失败原因
- **AND** 系统 MUST NOT 标记该版本 prepared
- **AND** 系统 MUST NOT 修改链上升级计划

### Requirement: Solidity callFunc remains the invocation entry

系统 SHALL 保持 `CodeStorage.callFunc(name, input)` 作为合约侧唯一调用入口。EVM MUST NOT 解释 WASM；执行 MUST 发生在协处理器 WASM runtime。

#### Scenario: 合约调用已准备版本

- **WHEN** 当前区块高度已选择某 prepared WASM 版本
- **AND** 调用方通过 `callFunc` 传入该算法名和 ABI-encoded `input`
- **THEN** 宿主 SHALL 把 `input` 写入模块线性内存并调用导出 `execute`
- **AND** 系统 SHALL 将模块返回的 bytes 作为 `callFunc` 输出交回 EVM

#### Scenario: 调用形状保持不变

- **WHEN** 现有合约或实验脚本调用 `callFunc`
- **THEN** 方法选择器、参数顺序和返回类型 SHALL 与切换 WASM 之前一致

### Requirement: Sandboxed execution with explicit limits

系统 SHALL 在沙箱中执行 WASM 模块，并强制执行资源上限。

#### Scenario: 拒绝非法 host import

- **WHEN** 模块请求文件系统、时钟、随机数、网络或其他未授权 host import
- **THEN** 实例化 SHALL 失败
- **AND** 该版本 MUST NOT 变为 prepared

#### Scenario: 执行超出资源上限

- **WHEN** 调用耗尽 fuel 或超过内存页上限
- **THEN** 本次 `callFunc` SHALL 失败
- **AND** 宿主进程 MUST NOT 因该调用崩溃

### Requirement: Fail-closed activation for missing WASM modules

当区块高度要求某版本生效时，系统 MUST NOT 因为本地未准备而执行旧版本语义。

#### Scenario: 未就绪节点拒绝新语义调用

- **WHEN** 当前区块号大于或等于目标版本 `activationBlock`
- **AND** 本地不存在该版本的已编译 WASM 实例
- **THEN** `callFunc` SHALL 返回 required-version-missing 或等价错误
- **AND** 系统 MUST NOT 返回旧版本的成功输出

#### Scenario: 生效前仍使用旧版本

- **WHEN** 当前区块号小于新版本 `activationBlock`
- **AND** 旧版本已经 prepared
- **THEN** 系统 SHALL 继续执行旧版本 WASM 模块
