## 功能审计记录

### 目标能力矩阵

| 能力 | 状态 | 证据 | 后续任务 |
| --- | --- | --- | --- |
| 合约触发升级交易 | supported | `cryptoupgrade/contracts/code_storage.sol` 提供 `uploadCode`，EVM dispatcher 处理该 selector 并发出 `codeUploaded` event。 | 保留兼容入口，新增版本化入口。 |
| 合约侧代码管理 | partial | 当前 `algo` 只保存 `code/gas/itype/otype`，且 `mapping(string => algo)` 只能保存单版本。 | 2.1、2.3 |
| 合约侧算法版本管理 | missing | Solidity 合约和 `common.CodeStorageABI_json` 均无 `version`、`activationBlock`、版本列表或当前版本查询。 | 2.1、2.2、2.3、2.5 |
| 指定区块启用新算法 | missing | `runCryptoUpgradeCall` 没有向 `RunCodeStorageCall` 传入 EVM block number，dispatcher 也没有按高度选择版本。 | 3.1、3.2 |
| 立即切换到新算法 | partial | 当前单版本上传后，一旦事件激活完成，`callFunc` 会使用本地 active metadata；但该切换不由链上版本计划表达。 | 2.2、3.2、3.5 |
| 上传交易与本地编译解耦 | supported | `cryptoupgrade/internal/evm/code_storage.go` 的 `uploadCode` 只写 uploaded metadata 并发事件，不同步调用 `ActivateAlgorithm`。 | 后续需扩展到版本化事件。 |
| 事件驱动本地激活 | partial | `cryptoupgrade/internal/event/service.go` 解析 `codeUploaded(string)` 后调用 `getInfo(name)` 和 `Activate`；缺少 version 与 activation block。 | 4.1、4.2、4.3 |
| 本地 artifact 版本隔离 | missing | `repository.Workspace.SourcePath(name)` 和 `PluginPath(name)` 只按算法名生成路径；`algorithm_info.json` 只保存 active map。 | 3.3、3.4 |
| 插件缓存支持同名升级 | partial | `pluginruntime.Loader.Activate` 会创建内容寻址 snapshot，能避免 Go `plugin.Open` 同路径缓存问题；但 active key 仍只按 symbolName。 | 3.4、4.4 |
| 多节点升级观测 | supported | `benchupgradelatency` 已有 YAML 网络选择、preflight、升级交易提交、节点并发 receipt 与 `callFunc` 轮询、JSON/CSV 输出。 | 5.1-5.6 复用并扩展。 |
| 版本一致性实验输出 | missing | 现有实验输出延迟和调用成功，不采集版本视图、采样高度、block hash 一致性或事件字段一致性。 | 5.3-5.6 |

### 当前代码结论

- `CodeStorage` Solidity 合约和内置 ABI 目前是单版本接口，不足以直接验证“指定区块启用”。
- Geth 运行时已经完成事件驱动激活边界，这是升级稳定性实验可以复用的基础。
- `callFunc` 当前按本地 active map 选择实现，无法保证不同节点在同一区块高度选择同一版本。
- repository 和 plugin loader 的 snapshot 机制提供了避免半写入和同路径缓存的基础，但还需要按版本隔离 metadata 与 artifact。
- `benchupgradelatency` 可以复用网络配置、RPC、fixture、交易发送、receipt 查询、`callFunc` 验证和 JSON/CSV 写出逻辑；稳定性实验需要新增按区块采样和一致性判定。

### 实现边界

- 保留旧 `uploadCode(name, code, gas, itype, otype)` 和 `callFunc(name, input)`，降低对实验 1、实验 2 的影响。
- 新增版本化入口和查询接口，不把版本号塞入算法名作为长期方案。
- 版本选择基于 EVM block number，不基于事件监听完成时间或本地时钟。
- 本地激活失败只影响节点级完成状态，不改变链上版本计划。
