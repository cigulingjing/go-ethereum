## Context

当前动态升级路径已经在单节点环境中验证：外部实验命令通过 `CodeStorage.uploadCode(name, base64Code, gasLimit, inputType, outputType)` 提交升级交易，EVM 在执行交易时调用 `cryptoupgrade.ActivateAlgorithm` 完成本节点源码解压、Go plugin 编译、算法信息写入和 `codeUploaded` log 生成。升级后的可用性通过 `CodeStorage.callFunc(name, encodedInput)` 验证；该调用会读取本节点本地算法信息并加载本地 `.so`，因此比单纯查询区块高度或 receipt 更能证明节点已完成本地升级。

`add-multi-node-network` 已经定义 YAML 网络拓扑、节点 RPC、独立 `pluginDir`、Docker Compose 渲染和 `cryptoupgrade/network.ValidateNetwork`。`clique-sealer` 负责让该网络具备 Clique 出块和 observer 同步能力。现有 `benchmark-upgrade-efficiency` 只在单节点 `geth --dev` 上测量 Add 升级效率，不覆盖区块传播、节点导入和多节点本地 plugin 激活的总耗时。

本 change 面向论文实验中的多节点升级延迟：实验控制进程从一个 signer 或配置的 sender RPC 发起升级交易，并使用同一台控制机的单调时钟观测所有目标节点，避免依赖不同服务器本地时钟同步。需要注意，控制机观测到的时间包含 RPC 往返和轮询间隔，因此结果定义为“实验控制端可观测的端到端升级完成时间”。

## Goals / Non-Goals

**Goals:**

- 新增多节点升级延迟实验入口，读取 `cryptoupgrade/network` YAML 配置并复用现有节点 RPC 解析。
- 在测量开始前执行网络前置检查，确认 RPC、chain ID、peer count 和 Clique 出块满足实验条件。
- 通过一次 `CodeStorage.uploadCode` 交易触发升级，并记录提交、交易 hash 返回、sender receipt、每个节点 receipt 可见、每个节点 `callFunc` 验证成功的时间点。
- 以每个节点 `CodeStorage.callFunc` 返回期望结果作为该节点完成升级的判据。
- 计算节点级耗时和全网耗时，其中全网完成耗时为最慢目标节点完成时间减去升级交易提交开始时间。
- 支持重复轮次，并避免上一轮已加载算法污染下一轮测量。
- 输出机器可读 JSON 和 CSV 原始结果，保留实验配置、节点列表、payload size、gas used、区块号、失败原因和观测限制。

**Non-Goals:**

- 不修改 `CodeStorage.uploadCode`、`CodeStorage.callFunc`、算法 ABI 或 EVM 执行语义。
- 不引入新的 plugin 编译模型；实验中发现的 `.so` 写入/打开竞态仅通过同目录临时文件、原子 rename 和 `plugin.Open` 缓存修复，保证升级验证时调用侧只看到完整 artifact。
- 不替代 `cryptoupgrade/bench/cmd/multinode` 的 render/init/validate/compose 职责；实验命令默认连接已经启动的多节点网络。
- 不在本 change 中实现跨服务器文件分发、Docker 编排生命周期管理、故障注入或网络延迟模拟。
- 不把单节点 `benchmark-upgrade-efficiency` 合并到多节点实验命令中。
- 不把本地日志时间戳作为权威耗时来源；节点日志只作为排查和复核材料。

## Decisions

1. 新增独立命令 `cryptoupgrade/bench/cmd/benchupgradelatency`

   该命令专注多节点端到端升级延迟，避免把多节点观测逻辑混入 `benchupgradeefficiency` 的单节点 dev 链自动启动流程，也避免让 `multinode validate` 承担性能实验职责。建议 flags 包括：

   - `-config`：多节点 YAML 配置路径，默认可指向 `cryptoupgrade/examples/networks/local-2nodes.yaml`。
   - `-sender`：发送升级交易的节点 ID，默认第一个 `role: signer` 节点。
   - `-nodes`：参与观测的节点 ID 列表，默认所有配置节点。
   - `-source`、`-algorithm`、`-input-type`、`-output-type`、`-algo-gas`、`-tx-gas`：升级算法和交易参数。
   - `-rounds`、`-poll-interval`、`-timeout`、`-preflight`、`-out`、`-output-json`、`-output-csv`：实验控制和输出参数。

   Alternative considered: 扩展 `cryptoupgrade/bench/cmd/multinode -mode validate`。该方案会把“网络是否可用”和“论文性能测量”耦合在同一入口，且 validate 的 cryptoupgrade smoke test 会上传 Add，容易污染升级延迟实验。

2. 前置检查复用 `network.ValidateNetwork`，但禁用 cryptoupgrade smoke test

   测量前先确认所有目标节点 RPC 可达、chain ID 一致、peer count 达标、区块高度可增长。该阶段不计入升级耗时，并且必须关闭 `CryptoUpgradeSmoke`，因为 smoke test 会执行 `CodeStorage.uploadCode("Add", ...)`，导致后续 Add 延迟测量失去“未升级”基线。

   Alternative considered: 不做前置检查，直接提交升级交易。该方案会把网络未同步、RPC 不可达和出块异常混入升级延迟，实验失败原因不清晰。

3. 使用控制机单调时钟作为唯一耗时来源

   所有阶段时间由实验控制进程记录：`submissionStartedAt`、`submissionReturnedAt`、`senderReceiptObservedAt`、每个节点的 `receiptObservedAt` 和 `completedAt`。跨服务器时不使用节点本地日志时间做差，避免 NTP 偏差影响结论。结果中保留 wall-clock timestamp 便于人工定位，但延迟计算使用 Go `time.Time` 的单调时钟差值。

   Alternative considered: 解析每个节点 Geth 日志中的 activation 时间。日志格式、时钟同步和容器 stdout 聚合都会引入额外误差，不适合作为主指标。

4. 将“区块导入”和“本地升级完成”拆成两个观测点

   每个节点先轮询 `eth_getTransactionReceipt` 或等价 receipt 查询；当该节点返回升级交易 receipt 时，说明节点已经看到包含升级交易的区块。随后继续对该节点执行 `eth_call CodeStorage.callFunc(name, encodedInput)`，只有返回期望输出才标记该节点 `completed`。这样可以区分链同步延迟和本地算法可调用延迟。

   Alternative considered: 只以 receipt 可见作为完成标准。该方案无法证明本地 `.so` 已生成并可被 `callFunc` 加载；对论文中的“所有节点完成升级操作”定义过弱。

5. 并发轮询所有目标节点

   升级交易提交后，为每个目标节点启动独立观测 goroutine，在统一 timeout 和 poll interval 下记录节点状态。全网完成时间取所有目标节点 `completedAt` 的最大值；任一目标节点超时则该轮 `completed=false`，仍保留已完成节点的部分数据。节点选择默认包含配置中的 signer、observer 和 rpc 节点，但只统计暴露 RPC 的节点。

   Alternative considered: 按节点顺序串行检查。串行检查会把后一个节点的检测延迟人为推大，破坏节点级耗时比较。

6. 通过唯一算法名或清洁网络避免缓存污染

   单轮默认可以使用 `Add`。当 `-rounds > 1` 或检测到目标节点在提交交易前已经可以调用同名算法时，命令必须采用安全策略之一：生成带唯一 exported function 名的 Add-compatible 源码，例如 `AddLatency001`；或失败并提示使用新网络/清空 pluginDir。默认推荐生成每轮唯一算法名，并把生成源码保存到结果目录，保证每轮都测量一次真实上传和编译。

   Alternative considered: 重复上传同名 `Add`。该方案无法区分“已经可用”的本地缓存与本轮升级产生的激活。

7. 结果结构分为 round、node 和 summary 三层

   JSON 保留完整原始数据；CSV 用于论文表格和后续画图。建议字段包括：

   - Round: round index、algorithm、source path、payload sizes、tx hash、submit block、receipt block、gas used、sender node、started/finished timestamps、completed、error。
   - Node: node ID、role、RPC URL、preflight block、receipt observed time、receipt block、completion time、completion block、validation output、submit-to-receipt latency、receipt-to-complete latency、submit-to-complete latency、error。
   - Summary: node count、completed node count、submit-to-sender-confirm latency、submit-to-all-receipt latency、submit-to-all-complete latency、slowest node、poll interval、timeout、limitations。

   Alternative considered: 只输出汇总耗时。该方案不能定位延迟来自出块确认、某个 observer 同步慢，还是本地 plugin 编译失败。

8. 多算法实验使用内置 fixture，而不是让用户手工拼 ABI 参数

   `benchupgradelatency` 在 Add-only 验证基础上增加 fixture 集合：每个 fixture 明确算法展示名、上传函数名、源码路径、`uploadCode` 的 input/output type、验证调用输入和期望输出。这样可以批量执行 Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit、SchnorrVerify 等算法，并保证每个节点 completed 的判断仍然来自 `callFunc` 返回值校验。

   为避免已有同名算法污染正式实验，批量模式默认在上传函数名后追加稳定前缀或轮次后缀，并在结果中记录 `algorithm` 和实际 `upgradeName`。需要固定原始函数名时，用户仍可指定单算法并关闭唯一命名策略，但污染检查会在提交前阻止不可信测量。

   Alternative considered: 让用户通过命令行传入任意 ABI 类型和值。该方案灵活但不可复现，且容易把错误编码的验证输入误当成升级失败。

9. 同时报告全网最慢完成耗时和所有节点平均耗时

   论文主指标仍保留 `submit-to-all-complete`，因为它回答“所有节点完成升级需要多久”。新增 `meanNodeSubmitToCompleteMillis`、`meanNodeSubmitToReceiptMillis` 和 `meanNodeReceiptToCompleteMillis` 作为 round-level 辅助统计，用于反映典型节点升级体验。平均值只从 node-level 一手观测数据计算，节点级 CSV 继续保留每个节点的阶段耗时，便于后续计算方差、分位数和排查异常节点。

   10 节点本地 Clique 实验使用与 2 节点相同的 sender 和 fixture 逻辑，但每个节点必须拥有独立 datadir、pluginDir、node key 和 HTTP RPC 端口。该规模用于检查升级交易传播、observer 导入区块和本地 plugin 激活在更大节点数下的可观测延迟变化。

   Alternative considered: 只报告平均值。该方案会掩盖最慢节点决定全网可用时间的问题，不符合“所有节点完成升级操作”的实验目标。

10. 将升级交易拆分为控制端可观测阶段

   阶段拆分采用控制端 RPC 可观测事件，不把节点日志或内部 Clique 状态作为主数据源：

   `rounds[].phaseTimeline` 显式保存五个绝对时间点：

   - `transactionSubmittedAt`：控制端发起升级交易。
   - `txHashObservedAt`：sender RPC 返回交易 hash。
   - `receiptObservedAt`：sender RPC 返回成功 receipt。
   - `eventObservedAt`：控制端从升级交易 receipt 中确认 `codeUploaded` 事件。
   - `activationObservedAt`：所有目标节点都能通过 `CodeStorage.callFunc` 返回期望结果。

   - `submit-to-tx-hash`：从控制端开始调用 `eth_sendTransaction` 到 sender RPC 返回交易 hash，表示 sender RPC 接受升级交易。
   - `tx-hash-to-sender-receipt`：从交易 hash 返回到 sender RPC 首次返回 receipt，作为“控制端观测到的交易已被打包执行”的时间。
   - `sender-receipt-to-all-receipt`：从 sender receipt 到所有目标节点都能返回该交易 receipt，表示升级交易所在区块已在所有目标节点可见。
   - `receipt-to-event`：从 sender receipt 可见到控制端从该 receipt 确认 `codeUploaded` 事件。当前二者来自同一次 receipt 观测，通常为 0。
   - `event-to-all-complete`：从控制端确认 `codeUploaded` 事件到所有目标节点 `callFunc` 返回期望结果。
   - `all-receipt-to-all-complete`：从所有目标节点 receipt 可见到所有目标节点 `callFunc` 返回期望结果，表示 receipt 可见后剩余的本地可调用确认耗时。
   - `submit-to-all-complete`：端到端总耗时，仍作为主指标。

   节点级结果额外记录每个节点相对交易 hash 返回和 sender receipt 的耗时，例如 `txHashToReceiptMillis`、`senderReceiptToReceiptMillis`、`senderReceiptToCompleteMillis`。这些字段回答每个节点“什么时候接收到包含升级交易的区块”和“什么时候算法升级完成并可调用”。

   “共识完成”在本地单 signer Clique 实验中定义为 sender RPC 首次返回成功 receipt 的控制端观测时间；它不是 BFT 终局性或链上最终确认。需要更细粒度的节点内执行/编译时间时，后续应增加节点内埋点，而不是从现有 RPC 结果反推。

   Alternative considered: 使用区块时间戳或节点日志拆分共识、执行和 plugin 编译。区块时间戳粒度太粗且由 proposer 设置；节点日志存在跨容器时钟同步和日志缓冲问题，不适合作为主实验数据。

## Risks / Trade-offs

- [Risk] 控制机轮询会把 RPC 往返和 poll interval 计入观测延迟。→ Mitigation: 结果记录 poll interval，并把指标命名为控制端可观测延迟；需要更细粒度时后续再增加节点内埋点。
- [Risk] 如果配置节点未暴露 HTTP RPC，就无法确认该节点本地 `callFunc` 是否可用。→ Mitigation: 默认只把可达 RPC 节点纳入全网完成统计，并在结果中标明 excluded nodes；正式实验配置应为所有参与节点暴露受控 RPC。
- [Risk] 同名算法或遗留 pluginDir 会导致提交交易前算法已经可用。→ Mitigation: 提交前执行 precheck；重复实验默认生成唯一算法名，或在污染时失败。
- [Risk] `network.ValidateNetwork` 的 cryptoupgrade smoke test 会污染 Add 实验。→ Mitigation: 本命令 preflight 必须禁用 smoke test，并在文档中说明不要在同一算法名上先跑 smoke test。
- [Risk] 单 signer Clique 的出块 period 会显著影响 submit-to-confirm。→ Mitigation: 结果记录 consensus period、chain ID、节点数和 signer 数量，实验报告按网络配置分组解释。
- [Risk] 节点本地 Go plugin 编译环境不一致会造成部分节点超时。→ Mitigation: 输出每个节点失败原因和日志引用；正式实验前通过多节点 validate 和基础上传验证确认镜像环境一致。
- [Risk] `eth_call` 验证和事件触发编译可能在同一节点内并发访问同一个 `.so` 文件。→ Mitigation: 编译先写入临时文件并原子替换正式 artifact，调用侧缓存已打开的 plugin，避免读取半写入文件导致节点崩溃。

## Migration Plan

1. 保留现有单节点 `benchupgradeefficiency` 和 `multinode` 命令不变，新增独立 `benchupgradelatency`。
2. 先对 `cryptoupgrade/network` 中 RPC URL、节点选择和 Add calldata 构造逻辑做最小复用，避免复制大段网络配置解析代码。
3. 使用本地 2 节点 Clique Compose 网络完成单轮实验，再扩展到 3 个及更多节点配置。
4. 对 `-rounds` 实现唯一算法名或污染检测后，再进行重复实验统计。
5. 若实验命令失败，不需要迁移链状态；停止网络、删除本轮结果目录或重新渲染网络即可回到初始实验环境。

Rollback 策略：删除新增命令和结果目录即可，不影响 Geth 默认启动、动态升级运行时或现有 benchmark。

## Open Questions

- 正式论文实验是否要求把 signer 节点和 observer/rpc 节点分别列为不同组，还是统一统计所有暴露 RPC 的节点。
- 多轮实验默认生成唯一算法名，还是强制每轮使用新网络以保持算法名固定为 `Add`。
- 是否需要在后续 change 中增加节点内 activation timestamp 埋点，用于拆分“区块执行完成”和“plugin 编译完成”的更细粒度延迟。
