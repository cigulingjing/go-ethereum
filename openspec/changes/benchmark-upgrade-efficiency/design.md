## Context

当前动态升级路径已经存在，实验必须基于这套实现：

- 升级请求由外部脚本或控制台向 `common.CodeStorageAddress` (`0x0000000000000000000000000000000000000043`) 发送 `CodeStorage.uploadCode(name, base64Code, gasLimit, inputType, outputType)` 交易。
- 升级 payload 是 gzip 后再 base64 编码的 Go 算法源码，并作为 `uploadCode` ABI 参数的一部分进入交易 input。
- EVM 在 `core/vm` 中 special-case `CodeStorageAddress` 的 ABI selector，`uploadCode` 执行时调用 `cryptoupgrade.ActivateAlgorithm`，完成源码解压、Go plugin 编译、算法元数据写入和 `codeUploaded` log 生成。
- `cmd/geth` 启动后还会订阅 `codeUploaded` event；在当前单节点 dev 链上，upload 交易执行期间已经完成本节点激活，event listener 通常只做一致性检查或重复激活跳过。
- 升级后的算法通过 `CodeStorage.callFunc(name, encodedInput)` 调用，`callFunc` 根据算法名读取本地 `algorithm_info.json` / 内存 map，加载 `.so` plugin 并按 ABI 类型解码输入、编码输出。

`cryptoupgrade/docs/chain_setup.md:89-108` 指定的 dev 链启动命令是本实验唯一的链启动依据：在 `build/bin/chain` 下运行 `./geth --dev --datadir chain/node1 --password chain/password.txt --networkid 11223344 --http --http.addr 0.0.0.0 --http.port 8666 --http.corsdomain "*" --http.vhosts "*" --http.api "web3,eth,debug,net,admin"`。实验框架可以设置进程环境变量隔离 plugin 制品，但不创建额外私有链。

## Goals / Non-Goals

**Goals:**

- 提供一个 Add-only 升级效率实验入口，自动启动上述 `geth --dev`，等待 RPC ready，执行一次完整 `Add` 升级和验证。
- 分开记录 transaction submission、transaction confirmation、algorithm activation observation 和 upgraded algorithm validation 的时间点。
- 输出原始 JSON，包含升级交易 hash、receipt gas used、payload/input size、提交时 block、确认 block、推断的生效 block、验证调用 block、阶段延迟和 Add 输出。
- 保存本次 geth 日志，便于核对 upload、plugin compile 和 activation 相关日志。
- 明确记录无法从现有 RPC 精确观测的指标，尤其是 activation 与 confirmation 的关系。

**Non-Goals:**

- 不新增或改变动态升级机制，不绕过 `CodeStorage.uploadCode` / `CodeStorage.callFunc`。
- 不批量测试其他算法，不做重复轮次统计，不生成论文图表。
- 不搭建额外私有链，不修改 genesis、共识或 EVM 执行语义。
- 不把 `eth_call` 验证伪装成已上链验证交易；如果需要验证交易，应作为后续 change 明确设计。

## Decisions

1. 新增独立命令 `cryptoupgrade/cmd/benchupgradeefficiency`。

   该命令聚焦实验二的升级效率指标，避免把“自动启动 geth”和“分阶段升级测量”混入已有 `benchcandidate`、`benchrealchain` 和 `upgradeflow`。已有命令仍保留，用于部署/调用成本对比。

2. 使用 per-run artifact 目录隔离 plugin 状态。

   命令为 geth 进程设置 `GETH_CRYPTOUPGRADE_PLUGIN_DIR=<result-dir>/plugin`，避免上一次运行留下的 `algorithm_info.json` 让 Add 在提交升级前已经可用。该设置只影响 cryptoupgrade plugin 制品路径，不改变链 datadir、networkid 或 RPC 启动方式。

3. 以 receipt block 作为实际生效 block 的推断值，并保留观测字段。

   当前 `uploadCode` 在交易执行过程中同步调用 `ActivateAlgorithm`。因此 receipt 成功意味着执行该交易的 block 已完成激活；对同一节点而言，confirmation 和 activation 在语义上是同一执行点。公共 RPC 没有暴露“plugin 编译完成时间”或“内存 map 写入时间”的独立时间戳，所以 JSON 将包含：

   - `activationBlockNumber`: receipt block，来源标记为 `inferred_from_upload_receipt`；
   - `activationObservedAt`: receipt 后首次 `getInfo` 或 `callFunc` 成功的本地观测时间；
   - `activationObservationBlockNumber`: 观测时的 latest block。

4. 验证调用使用 `eth_call` 到 `CodeStorage.callFunc`。

   该方式与已有 benchmark 和文档中的调用路径一致，可以直接获得 Add 返回值。因为 `eth_call` 不产生 receipt，实验只记录调用时使用的 latest block，字段名明确为 validation call block，而不是 validation transaction block。

5. 同时记录多种 payload size。

   “升级 payload size”在当前实现中可指源码、gzip+base64 字符串或完整交易 input。实验 JSON 将分别记录 `sourceBytes`、`compressedGzipBytes`、`compressedBase64Bytes` 和 `uploadCalldataBytes`，避免把其中一个值误称为唯一 payload size。

## Risks / Trade-offs

- `geth --dev` 按需出块，空闲时不持续产块 → 交易确认延迟会包含 dev miner 触发和本地 plugin 编译时间，结果不代表固定出块间隔网络。
- `eth_estimateGas` 对 `uploadCode` 会执行同样的 special-case 路径，可能提前编译并污染 activation 观测 → 本实验默认使用固定 upload gas limit，不在提交前估算 upload gas。
- 如果 8666 端口已有节点占用，自动启动会失败 → 命令保留 `-reuse-rpc` 选项供人工复用现有节点，但正式单次实验默认启动新节点并保存日志。
- Add 代码很小，latency 可能主要由 dev 链调度和 plugin 编译开销决定 → 当前阶段只验证方法，后续批量实验再处理重复次数和统计显著性。
- `eth_call` 的 validation block 是调用时 latest head，不是链上交易 block → JSON 和报告必须明确该定义。
