## Context
技术栈：Go、Logrus JSONFormatter、现有 RPC/EVM/activation 链路。现有 activationtrace 保留兼容旧实验。必须保留工作区已有修改。
## Goals / Non-Goals
提供可关联的阶段事件，不修改交易结果、Gas、共识或版本选择。不实现实验聚合器。
## Decisions
- cryptoupgrade/stagelog 为无 core/rpc 依赖的公共模块，独立 Logrus 实例追加 JSONL。路径由 GETH_CRYPTOUPGRADE_STAGE_LOG 指定，或默认 pluginDir/stage_timing.jsonl；没有配置则不写。支持禁用开关。记录 UTC 纳秒时间、进程会话和节点标识；同步追加且失败不向业务传播。
- RPC 在已解码消息交给 handler 时采集接收时间（不是网卡收到首字节时间），每个批次成员拥有独立 requestId。发送接口获得 txHash 后写关联事件，拒绝请求仍有接收记录但无主链落块事件。
- EVM 携带仅供观测的 context；交易执行写 txHash 和 execution phase，eth_call 继承 requestId。通过 CodeStorage facade 和 runtimeCaller 传递到 WASM 调用边界，标记独立 executionId。不使用全局请求变量，避免并发串线。
- WASM 执行前后记录 coprocessor_enter/exit（含错误、durationNs）。保持原 WASM 执行取消语义，不将观测 context 的 RPC 取消改变原执行行为。builtin 不记录 WASM 阶段。
- writeHeadBlock 完成后记录各交易 transaction_included，覆盖本地产块和导入/SetCanonical；同一交易可因重组或重复执行多次记录，blockHash 和 executionId 用于区分。该时间不等于最终性或 receipt 成功。
- 激活服务实际工作开始写 wasm_upgrade_started，运行时加载成功后写 wasm_loaded；失败写 wasm_upgrade_failed。继承升级事件 txHash/version/activationBlock，未来生效高度不能被载入事件替代。
## Risks / Trade-offs
同步日志有观测开销，记录边界时间并提供禁用开关；跨机比较依赖时钟同步。日志追加无自动轮转，实验结束由用户归档。批量、模拟、矿工候选执行与最终落块区分，不能按行号直接相减。
## Migration Plan
加入模块与可关闭插桩，测试并发、失败、请求关联、实际 WASM 调用和区块写入；使用既有 plugin 挂载存放每节点文件，无需变更实验网络配置。
