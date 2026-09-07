# cryptoupgrade 多节点升级延迟实验

本文档说明 `lab1-upgrade-latency` 实验入口的使用方式。该实验测量从控制端发起 `CodeStorage.uploadCode` 升级交易，到所有目标节点都能通过本地 `CodeStorage.callFunc` 成功调用升级算法的可观测端到端耗时。

## 前置条件

先启动一个由 `experiments/cryptoupgrade/bench/cmd/multinode` 渲染的 Clique 多节点网络。建议先执行不带 `-crypto-smoke` 的网络验证：

```bash
go run ./experiments/cryptoupgrade/bench/cmd/multinode \
  -mode validate \
  -config experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml \
  -output-json build/cryptoupgrade-networks/local-2nodes/validate-result.json
```

不要在同一网络、同一算法名上先运行 `-crypto-smoke`，因为 smoke test 会上传 `Add`，导致后续 `Add` 升级延迟实验失去未升级基线。

## 单轮实验

```bash
go run ./experiments/cryptoupgrade/bench/cmd/benchupgradelatency \
  -config experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml \
  -out experiments/cryptoupgrade/results/upgrade-latency/local-2nodes \
  -rounds 1 \
  -timeout 2m \
  -poll-interval 200ms
```

默认行为：

- sender 使用第一个 `role: signer` 节点；
- 目标节点使用 YAML 中的所有节点；
- 单轮默认上传 `cryptoupgrade/algorithm/wasm/add.wasm` 中的 `Add`；
- 节点完成条件是该节点 RPC 上 `CodeStorage.callFunc("Add", encodedInput)` 返回 `a + b`；
- preflight 检查 RPC、chain ID、peer count 和区块高度增长，但不会执行 cryptoupgrade smoke test。

## 多轮实验

```bash
go run ./experiments/cryptoupgrade/bench/cmd/benchupgradelatency \
  -config experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml \
  -out experiments/cryptoupgrade/results/upgrade-latency/local-2nodes-3rounds \
  -rounds 3
```

多轮模式默认复用 `.wasm` 制品并使用 `AddLatency001`、`AddLatency002` 等唯一上传名，避免上一轮同名算法已经可调用而污染下一轮测量。仅当手动传入 Go 源码时，工具才会在结果目录的 `sources/` 下生成重命名后的 TinyGo 构建输入。

## 多算法实验

```bash
go run ./experiments/cryptoupgrade/bench/cmd/benchupgradelatency \
  -config experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml \
  -out experiments/cryptoupgrade/results/upgrade-latency/local-2nodes-all \
  -algorithms all \
  -rounds 1 \
  -timeout 3m
```

`-algorithms` 使用内置 fixture，当前覆盖 `Add`、`Sha256`、`Blake2bSum256`、`Pbkdf2Sha256`、`Dh2048Secret`、`PedersenCommit` 和 `SchnorrVerify`。批量模式默认给每个上传函数名追加 `LatencyNNN` 后缀，例如 `Sha256Latency001`，以避免同名算法、builtin 或之前实验残留污染测量。JSON/CSV 会同时记录算法展示名 `algorithm`、实际上传函数名 `upgradeName`、ABI 类型、输入摘要和期望输出。

## 10 节点实验

```bash
go run ./experiments/cryptoupgrade/bench/cmd/multinode \
  -mode render \
  -config experiments/cryptoupgrade/deployments/networks/local-10nodes.yaml \
  -out build/cryptoupgrade-networks/local-10nodes

docker compose -f build/cryptoupgrade-networks/local-10nodes/docker-compose.yml up -d

go run ./experiments/cryptoupgrade/bench/cmd/benchupgradelatency \
  -config build/cryptoupgrade-networks/local-10nodes/network.yaml \
  -out experiments/cryptoupgrade/results/upgrade-latency/local-10nodes-all \
  -algorithms all \
  -rounds 1 \
  -timeout 5m \
  -preflight-timeout 2m
```

10 节点配置会让每个节点使用独立 `datadir`、`pluginDir`、node key 和 HTTP RPC 端口。建议在 Docker Compose 启动后等待 peer count 稳定，再开始测量；preflight 不计入升级延迟。

正式多算法采集建议每个算法使用一套干净 10 节点网络，分别保存该算法的 `result.json`、`rounds.csv` 和 `nodes.csv`。这样可以避免同一 geth 进程连续加载多个 WASM 模块时产生的运行时干扰，同时保证每个算法的节点级数据都是从未升级状态开始采集的一手数据。

## 指定节点

```bash
go run ./experiments/cryptoupgrade/bench/cmd/benchupgradelatency \
  -config experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml \
  -sender node1 \
  -nodes node1,node2 \
  -from 0xF5F871aA6Bd253914705898c66251f994aa426FA
```

`-nodes` 只影响本次统计的目标节点。未纳入的节点会记录在 JSON 的 `config.excludedNodeIds` 中。

## 输出文件

默认输出到 `experiments/cryptoupgrade/results/upgrade-latency/run-<timestamp>/`：

- `result.json`：完整原始结果；
- `rounds.csv`：轮次级汇总；
- `nodes.csv`：节点级耗时；
- `sources/`：仅在手动传入 Go 源码时生成的 TinyGo 构建输入。

关键 JSON 字段：

- `rounds[].phaseTimeline.transactionSubmittedAt`：时间点 1，控制端开始发送升级交易；
- `rounds[].phaseTimeline.txHashObservedAt`：时间点 2，sender RPC 返回交易 hash，表示交易已被本地节点接受并进入网络传播路径；
- `rounds[].phaseTimeline.receiptObservedAt`：时间点 3，sender RPC 返回升级交易 receipt，表示控制端已观察到该交易被打包执行；
- `rounds[].phaseTimeline.eventObservedAt`：时间点 4，控制端从升级交易 receipt 中观察到 `codeUploaded` 事件；
- `rounds[].phaseTimeline.activationObservedAt`：时间点 5，所有目标节点都能通过 `CodeStorage.callFunc` 返回期望结果的时间；
- `rounds[].submissionStartedAt`：控制端开始发送升级交易的时间；
- `rounds[].submissionReturnedAt`：sender RPC 返回交易 hash 的时间，表示 RPC 已接受本次升级交易；
- `rounds[].receiptObservedAt`：sender RPC 看到升级交易 receipt 的时间；
- `rounds[].nodes[].receiptObservedAt`：目标节点首次看到升级交易 receipt 的时间；
- `rounds[].nodes[].completedAt`：目标节点首次 `callFunc` 返回期望结果的时间；
- `rounds[].summary.submitToTxHashMillis`：从开始发送交易到返回交易 hash；
- `rounds[].summary.txHashToSenderReceiptMillis`：从交易 hash 返回到 sender 节点看到 receipt，可作为本实验中“交易被打包执行/共识完成”的控制端观测耗时；
- `rounds[].summary.senderReceiptToAllReceiptMillis`：从 sender 看到 receipt 到所有目标节点都能看到该 receipt，可作为“所有节点同步到升级交易所在区块”的控制端观测耗时；
- `rounds[].summary.receiptToEventMillis`：从 sender receipt 可见到控制端从该 receipt 中确认 `codeUploaded` 事件的耗时；当前实现中二者来自同一次 receipt 观测，通常为 0；
- `rounds[].summary.eventToAllCompleteMillis`：从控制端确认 `codeUploaded` 事件到所有目标节点可调用算法；
- `rounds[].summary.allReceiptToAllCompleteMillis`：从所有目标节点看到 receipt 到所有目标节点 `callFunc` 验证成功；
- `rounds[].summary.submitToAllReceiptMillis`：从交易发送开始到所有目标节点看到 receipt；
- `rounds[].summary.submitToAllCompleteMillis`：从交易发送开始到所有目标节点完成本地升级并可调用；
- `rounds[].summary.meanNodeSubmitToReceiptMillis`：所有已看到 receipt 节点的平均 submit-to-receipt 耗时；
- `rounds[].summary.meanNodeTxHashToReceiptMillis`：所有已看到 receipt 节点从交易 hash 返回到本节点 receipt 可见的平均耗时；
- `rounds[].summary.meanNodeSenderReceiptToCompleteMillis`：所有 completed 节点从 sender receipt 到本节点算法可调用的平均耗时；
- `rounds[].summary.meanNodeReceiptToCompleteMillis`：所有 completed 节点的平均 receipt-to-complete 耗时；
- `rounds[].summary.meanNodeSubmitToCompleteMillis`：所有 completed 节点的平均 submit-to-complete 耗时；
- `rounds[].summary.slowestNodeId`：决定全网完成时间的最慢节点。

`nodes.csv` 是节点级一手数据，包含每个节点的 submit-to-receipt、tx-hash-to-receipt、sender-receipt-to-receipt、receipt-to-complete、submit-to-complete、sender-receipt-to-complete、event-to-complete 和 validation call latency；`rounds.csv` 中的平均值由这些节点级观测数据计算得到。

## 观测口径

该实验使用控制端单调时钟计算耗时。结果包含 RPC 往返和 `-poll-interval` 带来的观测误差，因此应解释为“控制端可观测升级延迟”。`phaseTimeline.eventObservedAt` 是控制端从 receipt 中确认事件的时间，不等同于 geth 内部 `BindCodeUploaded` goroutine 打印 `Catch codeUploaded event` 的日志时间；节点日志时间戳只用于排查，不作为跨节点精确耗时来源。

本文档中的“共识完成”指 sender RPC 首次返回成功 receipt，即控制端观测到升级交易已经进入区块并执行完成；这不是 BFT finality，也不是 Clique 内部状态。本文档中的“同步完成”指所有目标节点都能通过 RPC 返回该升级交易 receipt；“算法升级完成”指所有目标节点都能通过 `CodeStorage.callFunc` 返回期望结果。
