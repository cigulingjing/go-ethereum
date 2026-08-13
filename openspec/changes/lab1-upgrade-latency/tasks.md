## 1. 实验入口与配置

- [x] 1.1 新增 `cryptoupgrade/bench/cmd/benchupgradelatency` 命令结构、flags 和基础结果目录初始化。
- [x] 1.2 复用 `cryptoupgrade/network.LoadConfig` 和 `network.RPCURL` 载入多节点 YAML，并实现 sender 节点、目标节点和 from 地址选择逻辑。
- [x] 1.3 实现测量前 preflight，检查目标节点 RPC、chain ID、peer count 和区块增长，并确保该检查不执行 cryptoupgrade smoke test。
- [x] 1.4 对缺失 sender、重复目标节点、不可达 RPC、无 signer 账户和空目标节点给出明确错误。

## 2. 升级交易构造与提交

- [x] 2.1 实现 Add-compatible 升级源码准备逻辑，支持单轮使用原始 Add，多轮生成唯一算法名和源码制品。
- [x] 2.2 实现 gzip/base64 压缩、`CodeStorage.uploadCode` calldata 构造、payload size 和 calldata hash 统计。
- [x] 2.3 在提交交易前对目标节点执行同名 `callFunc` 污染检查，发现已可调用时拒绝该轮或切换唯一算法名。
- [x] 2.4 通过 sender RPC 提交升级交易，记录 submission started/returned 时间、交易 hash、提交前 block 和 receipt gas used。

## 3. 节点级并发观测

- [x] 3.1 实现每个目标节点的并发 observer，轮询升级交易 receipt 可见时间、receipt block 和节点 latest block。
- [x] 3.2 实现每个目标节点的 `CodeStorage.callFunc` 验证调用，使用期望 Add 输出判定节点 completed。
- [x] 3.3 对每个节点记录 submit-to-receipt、receipt-to-complete、submit-to-complete 和 validation call latency。
- [x] 3.4 实现 timeout 和部分失败处理，确保失败节点不丢失已采集阶段数据。

## 4. 汇总结果与制品输出

- [x] 4.1 定义 round-level、node-level 和 summary JSON 结构，覆盖 spec 要求的配置、节点、交易、区块、gas、payload、耗时和失败原因。
- [x] 4.2 实现 JSON 原始结果写出，并在失败时也保留可解析结果文件。
- [x] 4.3 实现 round-level 和 node-level CSV 输出，用于论文表格和后续绘图。
- [x] 4.4 在结果中记录 poll interval、timeout、excluded nodes、控制端 RPC 观测限制和可选节点日志路径。

## 5. 验证与文档

- [x] 5.1 为节点选择、唯一算法名生成、payload 统计、耗时汇总和 timeout 处理添加 Go 单元测试。
- [x] 5.2 对新增 Go 代码运行 `gofmt` 和相关 `go test`。
- [x] 5.3 编写或更新多节点升级延迟实验说明，给出基于本地 Clique 多节点网络的示例命令和输出字段解释。
- [x] 5.4 运行 `openspec validate lab1-upgrade-latency`。
- [x] 5.5 在可用的本地 2 节点 Clique 网络上执行至少一轮升级延迟实验，并检查 JSON/CSV 中全网完成耗时和节点级阶段数据。

## 6. 多算法升级延迟实验

- [x] 6.1 为升级延迟命令增加内置算法 fixture，覆盖 Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit 和 SchnorrVerify。
- [x] 6.2 支持 `-algorithms` 批量选择和每个算法唯一 upload function name，避免同名算法或 preload/precompile 状态污染测量。
- [x] 6.3 将结果结构和 CSV 扩展为同时记录算法展示名、实际上传函数名、fixture 输入摘要和期望输出。
- [x] 6.4 添加多算法 fixture、ABI 编码和输出校验单元测试。
- [x] 6.5 在本地 2 节点 Clique 网络上运行多算法升级延迟实验，保存 JSON/CSV 原始结果并汇总每个算法全网完成耗时。
- [x] 6.6 运行 `gofmt`、相关 `go test` 和 `openspec validate lab1-upgrade-latency`。

## 7. 10 节点平均升级延迟实验

- [x] 7.1 在 round-level JSON/CSV 中增加所有节点平均 submit-to-receipt、receipt-to-complete 和 submit-to-complete 耗时。
- [x] 7.2 更新单元测试，验证平均值来自每个节点的一手观测时间。
- [x] 7.3 新增本地 10 节点 Clique 实验配置，确保所有节点暴露 RPC 并使用独立 pluginDir。
- [x] 7.4 在本地 10 节点 Clique 网络上运行多算法升级延迟实验，保存 JSON、round CSV 和 node CSV 原始结果。
- [x] 7.5 运行 `gofmt`、相关 `go test` 和 `openspec validate lab1-upgrade-latency`。

## 8. 升级交易阶段拆分

- [x] 8.1 明确控制端可观测阶段口径，区分交易 RPC 接受、sender receipt、所有节点 receipt 和所有节点算法可调用。
- [x] 8.2 在 round-level JSON/CSV 中输出阶段时间点和阶段耗时。
- [x] 8.3 在 node-level JSON/CSV 中输出每个节点相对交易返回和 sender receipt 的同步/完成耗时。
- [x] 8.4 更新单元测试和实验说明，解释“共识完成”和“同步完成”的观测口径限制。
- [x] 8.5 运行 `gofmt`、相关 `go test` 和 `openspec validate lab1-upgrade-latency`。

## 9. Add-only 10 节点验证

- [x] 9.1 确认 `local-10nodes.yaml` 和渲染后的 `network.yaml` 均为 `smokeTest: false`，避免 preflight 上传 Add 污染正式测量。
- [x] 9.2 修复不完整轮次中 `all-receipt-to-all-complete` 可能出现负值的问题，仅在全节点完成时输出该阶段汇总。
- [x] 9.3 修复 plugin artifact 写入/打开竞态：编译输出先写入同目录临时文件，再原子替换 `.so`，并缓存首次 `plugin.Open` 结果。
- [x] 9.4 使用重建后的 `cryptoupgrade-geth:lab` 镜像运行 Add-only 10 节点 Clique 实验，保存结果到 `cryptoupgrade/results/upgrade-latency/lab1-add-stage-fixed-20260812-145105/`。

## 10. 五点时间轴整理

- [x] 10.1 在 round-level JSON 中增加 `phaseTimeline`，显式输出客户端发起交易、获取 tx hash、获取 receipt、确认 `codeUploaded` 事件和算法可调用完成五个时间点。
- [x] 10.2 在 `rounds.csv` 中增加五个绝对时间列，并补充 `receipt_to_event_ms`、`event_to_all_complete_ms` 阶段耗时。
- [x] 10.3 在 `nodes.csv` 和 node-level JSON 中增加 `event_to_complete_ms`，保留每个节点从事件确认到算法可调用的一手耗时。
- [x] 10.4 更新实验说明，明确 `eventObservedAt` 是控制端从 receipt 确认事件的时间，不等同于 geth 内部 listener 日志时间。
- [x] 10.5 重新运行 Add 单节点 bind-event 路径实验，保存五点时间轴结果和节点日志到 `cryptoupgrade/results/upgrade-latency/lab1-add-1node-bind-event-timeline-20260812-222944/`。
