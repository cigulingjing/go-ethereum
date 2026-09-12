# Geth 阶段时间日志

独立 Logrus 实例追加 JSONL，每行一个时间事件；保留原 `activation_trace.jsonl`。

## 文件位置

已有网络设置了 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`，重新构建 Geth/镜像后自动写入各节点的 `plugin/stage_timing.jsonl`。默认本地网络的宿主机路径为：

```text
experiments/cryptoupgrade/deployments/nodes/node1/plugin/stage_timing.jsonl
experiments/cryptoupgrade/deployments/nodes/node2/plugin/stage_timing.jsonl
...
```

| 环境变量 | 含义 |
| --- | --- |
| `GETH_CRYPTOUPGRADE_STAGE_LOG` | 显式指定日志文件路径 |
| `GETH_CRYPTOUPGRADE_PLUGIN_DIR` | 未指定文件时，使用该目录的 `stage_timing.jsonl` |
| `GETH_CRYPTOUPGRADE_STAGE_LOG_DISABLE=true` | 禁用阶段日志 |
| `GETH_CRYPTOUPGRADE_NODE_ID` | 可选节点标签，默认 hostname |

没有显式文件路径且没有 plugin 环境变量时不创建日志。容器内指定路径需位于已有挂载目录，如 `/plugin/stage_timing.jsonl`。日志打开失败只告警；写入失败由 Logrus 报告，不改变业务返回或退出节点。

## 事件边界

| 链路 | stage | 时间点 |
| --- | --- | --- |
| 调用/升级 | `rpc_received` | 已解码请求进入 RPC handler，尚未排队执行 |
| 调用 | `rpc_call` | eth_call 进入 API，补充算法/方法元数据 |
| 调用/升级 | `rpc_transaction` | 已取得签名交易哈希，关联 requestId 与 txHash，尚未验证入池 |
| WASM 调用 | `coprocessor_enter` | 版本/元数据已解析，进入 WASM 执行函数 |
| WASM 调用 | `coprocessor_exit` | 执行函数返回，记录 success、error、durationNs |
| 调用/升级交易 | `transaction_included` | 区块主链索引与内存链头更新完成，先于升级事件通知 |
| 升级 | `wasm_upgrade_started` | 实际升级开始，准备解码并持久化 WASM |
| 升级 | `wasm_loaded` | 模块准备、编译、实例化和缓存成功完成 |
| 升级失败 | `wasm_upgrade_failed` | 升级返回错误，该次任务不会发出 wasm_loaded |

`rpc_received` 覆盖 eth_call、eth_sendTransaction、eth_sendRawTransaction、eth_sendRawTransactionSync，包括 batch 和 notification。batch 成员共享接收时间，各自有独立 requestId。这里是服务端解码后的接收时间，不是网卡收到首字节时间，也不表示交易已经通过验证。不能解码/提交的交易可能只有接收事件。

WASM 进入/退出区间包含函数内 Gas 检查、懒加载、模块锁等待与执行；builtin 不记录 WASM 阶段，版本解析失败而未进入 WASM 函数也没有进入事件。

eth_call 不产生打包事件。transaction_included 是本节点主链采纳时间：矿工记录本地产块写入，其他节点记录导入后的写入；不是区块头秒级 timestamp，也不是最终性或交易成功证明。执行状态需结合 receipt.status。

预约升级的 wasm_loaded 表示本地模块准备好，但仍须达到 activationBlock 才会被版本选择逻辑选中；不能将提前加载时间视为已生效。即时升级在完成加载后即可按当前高度调用。相同且已经准备好的版本被跳过时，不产生新的升级任务事件。

## 关联和计算

- nodeId/sessionId 标识节点与进程启动；重启更换 sessionId。
- requestId 是服务端生成的唯一标识；rpcId 是客户端原始 JSON-RPC ID，不能单独作为全局标识。
- rpc_transaction 将 requestId 关联到 txHash；直接 CodeStorage 调用还提供 flow（call/upgrade）、contractMethod、algorithm、升级 version。经中间合约调用时顶层 flow 为 transaction，WASM 阶段仍继承外层 txHash。
- 配对 coprocessor_enter/exit 使用 executionId。矿工候选执行、区块导入和重放可能多次执行同一交易，不能只用 txHash 配对。
- phase=simulation 表示 eth_call 等模拟；transaction_execution 是交易执行尝试；canonical 才表示落块。候选执行可能没有 blockHash。
- 升级用 txHash、algorithm、version、upgradeId 关联。blockNumber 为来源升级事件区块号，activationBlock 为版本生效高度。
- time 是 UTC RFC3339Nano；timestampUnixNano 是十进制字符串，转整数计算以避免 JSON 浮点精度损失；durationNs 使用进程内单调时钟，单位纳秒。

调用链：通过 requestId 找接收时间和配对执行事件；交易通过 rpc_transaction 得到 txHash，再找 transaction_included。升级链：按升级 txHash 找落块、升级开始和载入完成时间，各节点分别计算。

事件文件顺序可能与时间顺序不同，应按关联标识和时间分析。跨机器差值依赖时钟同步。同步日志有观测开销，进入日志的写入计入执行区间，实验需注明开启状态。文件不自动轮转，实验结束后归档；重新 render 不清空旧日志，使用 sessionId 区分运行。
