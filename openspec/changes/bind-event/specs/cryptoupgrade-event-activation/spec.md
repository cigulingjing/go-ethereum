## ADDED Requirements

### Requirement: CodeStorage upload only submits upgrade data
系统 SHALL 将 `CodeStorage.uploadCode` 作为链上升级数据提交入口，并 SHALL NOT 在交易执行路径中执行节点本地算法激活。

#### Scenario: uploadCode stores metadata and emits event
- **WHEN** 调用方向 `CodeStorage.uploadCode(name, code, gas, itype, otype)` 提交升级交易
- **THEN** 系统 SHALL 将算法元数据保存到 `CodeStorage` 可查询状态
- **AND** 系统 SHALL 发出 `codeUploaded` event，event data SHALL 包含规范化后的算法名
- **AND** 系统 SHALL NOT 在该交易执行路径中调用 `ActivateAlgorithm`
- **AND** 系统 SHALL NOT 在该交易执行路径中解压源码、编译 Go plugin 或写入本地 plugin 元数据文件

#### Scenario: uploadCode remains forbidden in static context
- **WHEN** 调用方在 static context 中调用 `CodeStorage.uploadCode`
- **THEN** 系统 SHALL 拒绝该调用
- **AND** 系统 SHALL NOT 保存算法元数据或发出 `codeUploaded` event

### Requirement: Event listener activates uploaded algorithms
节点 SHALL 通过链下事件监听线程处理 `codeUploaded` event，并以该线程作为本地算法激活的唯一自动入口。

#### Scenario: listener activates algorithm from event
- **WHEN** 节点监听到 `common.CodeStorageAddress` 发出的 `codeUploaded` event
- **THEN** 监听线程 SHALL 从 event data 解析算法名
- **AND** 监听线程 SHALL 通过 `CodeStorage.getInfo(name)` 查询链上算法元数据
- **AND** 监听线程 SHALL 调用 `ActivateAlgorithm(name, info)` 完成本节点源码解压、Go plugin 编译、内存元数据更新和持久化

#### Scenario: listener ignores invalid event payload
- **WHEN** 节点监听到无法解析算法名或算法名为空的 `codeUploaded` event
- **THEN** 监听线程 SHALL 记录错误
- **AND** 监听线程 SHALL NOT 调用 `ActivateAlgorithm`
- **AND** 监听线程 SHALL 继续监听后续事件

#### Scenario: listener skips already active algorithm
- **WHEN** 节点监听到的算法元数据与本节点当前已激活元数据完全一致
- **THEN** 监听线程 SHALL 跳过重复激活
- **AND** 监听线程 SHALL 继续监听后续事件

### Requirement: Activation failures are isolated from upload transaction execution
事件驱动激活失败 SHALL 只影响本节点本地算法可用性，并 SHALL NOT 回滚已经成功执行的 `CodeStorage.uploadCode` 交易。

#### Scenario: local activation fails after successful upload
- **WHEN** `CodeStorage.uploadCode` 交易已经成功并发出 `codeUploaded` event
- **AND** 监听线程执行 `ActivateAlgorithm` 时发生源码解压、Go plugin 编译或本地元数据持久化错误
- **THEN** 监听线程 SHALL 记录包含算法名和错误原因的日志
- **AND** 对应 upload 交易 receipt SHALL 保持成功状态
- **AND** 本节点 SHALL NOT 将该算法标记为已激活

### Requirement: callFunc depends on local event activation
系统 SHALL 保持 `CodeStorage.callFunc` 使用本节点本地已激活算法状态执行调用。

#### Scenario: callFunc before local activation
- **WHEN** `CodeStorage.uploadCode` 交易已经确认，但本节点事件监听线程尚未完成对应算法激活
- **THEN** `CodeStorage.callFunc(name, input)` SHALL NOT 假定该算法已经可用
- **AND** 调用结果 SHALL 由本节点当前本地算法状态决定

#### Scenario: callFunc after local activation
- **WHEN** 本节点事件监听线程已经完成对应算法的 `ActivateAlgorithm`
- **THEN** `CodeStorage.callFunc(name, input)` SHALL 使用已激活的本地 Go plugin 执行算法
- **AND** 返回值 SHALL 继续遵循该算法配置的 ABI 输入输出类型
