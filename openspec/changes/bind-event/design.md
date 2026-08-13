## Context

当前 `cryptoupgrade/code_storage.go` 的 `RunCodeStorageCall` 会在 `uploadCode` 分支中构造 `algoInfo` 后立即调用 `ActivateAlgorithm`。这意味着 EVM 执行升级交易时会直接访问本地文件系统、解压源码、调用 Go toolchain 编译 plugin、更新内存 map 并持久化 `algorithm_info.json`。这种实现便于单节点实验，但把链上交易执行和节点本地运行时升级混在一起：不同节点的本地编译环境、文件系统权限和插件缓存会影响交易执行路径，也让 upload receipt 被误用为“升级已完成”的判断。

`cryptoupgrade/bind_event.go` 已经提供更合适的边界：节点通过 `SubscribeFilterLogs` 订阅 `common.CodeStorageAddress` 的 `codeUploaded` event，解析算法名后调用 `CodeStorage.getInfo` 读取链上算法信息，再执行 `ActivateAlgorithm`。本 change 将该路径定义为升级激活的唯一自动入口。

## Goals / Non-Goals

**Goals:**

- 让 `CodeStorage.uploadCode` 不再在交易执行路径中调用 `ActivateAlgorithm`。
- 保留 `uploadCode` 的链上数据提交和 `codeUploaded` event 语义，使事件监听线程有确定的触发源。
- 明确 `BindCodeUploaded` 负责解析事件、读取链上元数据、去重和激活算法。
- 更新实验和 benchmark 的完成判据，避免把 receipt 成功直接解释为本地算法可调用。
- 保持 `CodeStorage.callFunc` 的本地 plugin 调用模型不变。

**Non-Goals:**

- 不改变 `CodeStorage` ABI、`codeUploaded` event ABI、`CodeStorage.callFunc` ABI 或算法源码压缩格式。
- 不新增跨节点源码分发协议；算法源码仍来自 `uploadCode` 写入的链上元数据。
- 不在本 change 中实现 rollback、重试队列、历史事件补扫或 reorg 处理策略。
- 不重构 `ActivateAlgorithm` 的编译、路径、缓存或持久化实现。
- 不修改 Solidity 对照组或 native precompile 对照组。

## Decisions

1. `uploadCode` 只提交算法信息并发出 event

   `RunCodeStorageCall` 在 `uploadCode` 分支中仍进行 read-only 检查、算法名规范化和 ABI 参数解析，但不调用 `ActivateAlgorithm`。它需要把 `algoInfo` 写入 `CodeStorage.getInfo` 可读的状态，然后发出 `codeUploaded` event。这样链上升级请求的交易结果不依赖本地 Go plugin 编译是否成功。

   Alternative considered: 在 `uploadCode` 中调用 `ActivateAlgorithm` 后再发 event。该方案保留了当前问题，event listener 只能做重复检查，无法成为升级控制的主路径。

2. `BindCodeUploaded` 是自动激活的唯一入口

   节点启动后由专门 goroutine 运行 `BindCodeUploaded(client)`。监听到 `codeUploaded` 后，它解析 name，调用 `lookupCodeInfo(client, name)`，再执行 `ActivateAlgorithm(name, *info)`。已有的“当前本地信息与链上信息完全一致则跳过”逻辑应保留，避免同一事件被重复处理时反复编译。

   Alternative considered: 在 `callFunc` 发现算法缺失时懒加载链上信息并激活。该方案会把用户调用路径变成潜在编译路径，仍然让普通 EVM/RPC 调用承担升级副作用，不适合作为清晰的升级流程。

3. upload 成功与 activation 成功分离

   `uploadCode` receipt 只表示升级数据和 event 已进入链上执行结果；本节点是否可调用算法必须通过事件监听线程完成激活后再判断。若 `ActivateAlgorithm` 失败，监听线程记录错误并继续运行，不回滚 upload 交易，也不把本节点状态标记为激活成功。

   Alternative considered: 事件监听失败后通过本地 panic 或终止节点暴露错误。该方案会把单个算法升级失败扩大为节点可用性问题，不适合实验网络。

4. 保持 ABI 和用户提交方式兼容

   外部脚本、控制台和 benchmark 仍使用 `CodeStorage.uploadCode(name, code, gas, itype, otype)` 提交升级请求。变化只在完成判据上：提交端在 receipt 后还要等待目标节点 `callFunc` 成功，或等待本地可观测的 activation 信号。

   Alternative considered: 新增 `submitCode` / `activateCode` 两个 ABI 方法。该方案需要迁移现有脚本和文档，且当前 event-driven 模型已经能表达提交与激活的分离。

5. benchmark 文档和测量字段需要同步修正

   已有 `benchmark-upgrade-efficiency`、`lab1-upgrade-latency` 和相关 docs 中有“uploadCode 在交易执行期间同步 ActivateAlgorithm”或“receipt 成功意味着激活”的描述。实现该 change 后，这些描述必须改为：receipt 是 upload confirmation；activation 由 event listener 完成；升级完成以 `CodeStorage.callFunc` 返回期望结果为准。升级 latency 可以拆分为 submit-to-receipt 和 receipt-to-activation/callFunc-success。

   Alternative considered: 仅修改运行时代码，不更新实验说明。该方案会让论文实验继续按旧语义解释数据，造成结果不可复现或指标定义错误。

## Risks / Trade-offs

- [Risk] 节点启动后若未运行 `BindCodeUploaded` goroutine，`uploadCode` 交易会成功但算法不会自动激活。→ Mitigation: 实现时检查 geth 启动绑定点，并增加测试或手工验证确保事件监听线程被启动。
- [Risk] `uploadCode` 需要有可查询的算法元数据；如果当前元数据只写入本地文件而非链上可读状态，删除 `ActivateAlgorithm` 后 `getInfo` 可能读不到新信息。→ Mitigation: 实现前确认 `getAlgorithmInfo` / `setAlgorithmInfo` 在 CodeStorage special-case 中的状态来源，并补齐 upload 分支的元数据写入路径。
- [Risk] 事件订阅只处理节点在线期间的新日志，错过历史 upload event 的节点不会自动补激活。→ Mitigation: 本 change 先保持现有订阅模型；历史补扫作为后续 change 单独设计。
- [Risk] 事件驱动后 activation 比 receipt 晚，现有 benchmark 可能在 receipt 后立即 `callFunc` 导致偶发失败。→ Mitigation: 调整 benchmark 等待逻辑，以轮询 `callFunc` 成功作为升级完成判据。
- [Risk] 多节点环境中各节点本地 Go toolchain 或 plugin 目录异常会导致部分节点激活失败，但链上 upload 仍成功。→ Mitigation: 日志记录算法名和错误原因，实验结果保留节点级失败信息。

## Migration Plan

1. 修改 `cryptoupgrade/code_storage.go` 的 `uploadCode` 分支，移除直接 `ActivateAlgorithm` 调用，保留或补齐算法元数据写入和 `codeUploaded` event 发出。
2. 检查 `cryptoupgrade/bind_event.go` 的事件解析、`lookupCodeInfo` 和去重逻辑，确保它可以独立完成激活，并修正明显错误日志或空 channel select 风险。
3. 检查 geth 启动路径，确认 `BindCodeUploaded` 由独立 goroutine 启动，并不会阻塞节点主流程。
4. 更新受影响 benchmark 和文档中的升级完成判据，等待 `callFunc` 成功而不是只等待 upload receipt。
5. 增加聚焦测试：`uploadCode` 不调用 `ActivateAlgorithm`，event listener 收到 `codeUploaded` 后能调用激活路径；必要时用 fake client 隔离测试事件解析和 `getInfo` 查询。

Rollback 策略：恢复 `uploadCode` 中的直接 `ActivateAlgorithm` 调用即可回到旧的同步激活语义；同时撤回 benchmark 文档中对事件驱动完成判据的修改。

## Open Questions

- 是否需要在本 change 中实现节点重启后的历史 `codeUploaded` event 补扫，还是保留给后续 `add-event-replay` 类 change。
- `updataGas` 是否也应发出事件并由监听线程更新本地 gas 元数据，还是继续保持当前同步本地更新路径。
- benchmark 是否需要新增显式 `activationObservedAt` 字段，还是复用已有 `callFunc` 验证成功时间。
