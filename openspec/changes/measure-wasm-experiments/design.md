## Context

技术栈：Go、ethereum/go-ethereum、Docker Compose、Clique 私有链、wazero WASM runtime、JSON/CSV 实验结果。

现有 Lab1 `benchupgradelatency` 已能上传 `.wasm` 并以 `callFunc` 成功作为完成判据，但只记录控制端 RPC 轮询时间；节点内 “收到事件、开始激活、WASM 编译完成” 只出现在普通日志中且不便机器读取。现有 Lab2 `benchexecutionefficiency` 会比较 upgrade/precompile/contract 并统计 Gas，本次需要 WASM-only 单节点运行耗时和容器资源利用率。

## Goals / Non-Goals

**Goals:**

- 为 WASM 激活路径增加机器可读节点内 trace，记录事件处理开始、WASM 持久化完成、编译完成和激活完成。
- 扩展 Docker Compose 渲染，使每个节点拥有相同 CPU/内存限制，并将该限制写入实验结果。
- 生成 20、30、40 节点配置，复用已有 5、10 节点配置完成 SchnorrProof 规模实验。
- 新增 WASM-only Lab2 命令或模式，只走动态升级路径，采集 `eth_call` 耗时和 Docker CPU/内存样本。
- 用一个 orchestration/report 入口按用户指定顺序运行三个实验并生成审计报告。

**Non-Goals:**

- 不修改 CodeStorage ABI、EVM 调用形状、WASM guest ABI 或算法 fixture 输出。
- 不把节点内 wall-clock 直接用于跨节点总延迟排序；全网延迟仍以控制端单调时钟观测为准。
- 不在 WASM-only Lab2 中输出 Gas 结论。

## Decisions

1. 节点内阶段采用 JSONL trace 文件，而不是解析普通日志。

   每个 geth 节点在 plugin 基础目录下写 `activation_trace.jsonl`。事件服务在收到 log 并准备调用 activator 前写 `event_received`/`activation_started`；activation 服务在 WASM decode/persist 后写 `wasm_persisted`；wasmruntime 在 `CompileModule` 返回后写 `wasm_compiled`，实例化完成后写 `wasm_instantiated`；activation 成功后写 `activation_completed`，失败时写 `activation_failed`。每条记录包含 node 可通过容器/文件路径关联的 name、version、txHash、blockNumber、wasmHash、timestamp 和可用 duration。

   备选方案是解析 `docker logs`。普通日志对机器读取和阶段关联不稳定，并且难以保证后续 CSV/JSON 聚合。

2. benchupgradelatency 在轮次结束后合并 trace。

   命令根据渲染后的 `network.yaml` 所在目录推导 `nodes/<id>/plugin/activation_trace.jsonl`，筛选本轮 `upgradeName`、version 和 tx hash 对应记录，写入 round/node JSON 与 CSV。控制端时间继续使用 `time.Time` 单调差值；节点内阶段只作为辅助阶段数据和编译耗时。

   备选方案是新增 RPC API 查询 trace。该方案侵入 geth API 面，不符合最小侵入。

3. Docker 资源限制放在网络 YAML 的 `docker` 段。

   新增 `cpus` 和 `memory` 字段，Compose 为每个 service 输出同一组 `cpus`、`mem_limit` 和必要标签。未配置时保持现有行为。正式实验配置统一使用同一资源值，报告记录这些字段。

4. 大规模网络配置用可复用生成工具维护。

   为避免手写 20/30/40 节点 YAML 出错，新增本地配置生成命令或脚本，根据节点数、端口基数、网络名和资源限制生成规范 YAML 与确定性 node key。生成结果作为实验输入保存到 `experiments/cryptoupgrade/results/<run-id>/configs/`，必要的固定示例可保留在 `deployments/networks`。

5. WASM-only Lab2 独立于三方对比命令。

   新增 `benchwasmexecution`，复用 Lab2 fixture 与动态升级 setup，只调用 `CodeStorage.callFunc`。命令在每个算法测量期间并发采样 `docker stats --no-stream` 或 Docker Engine stats API，输出 CPU percent、memory usage/limit、memory percent 的 min/mean/max/p95。若 Docker stats 不可用，实验失败而不是静默缺失资源指标。

6. 报告由统一 runner 生成。

   新增 `runwasmexperiments` 负责按顺序构建/启动网络、运行 Lab1/Lab2、保存日志和 Docker 状态、生成 `report.md`。报告最后读取 `paper/outline.md`，按实验内容对照 JSON 字段给出“已支撑/部分支撑/未支撑”的审计结论。

## Risks / Trade-offs

- [Risk] 40 节点本地 Docker 网络可能受机器 CPU/内存限制影响而超时。→ Mitigation: 每个节点使用相同资源限制，报告记录机器和容器资源；失败时保留部分结果和日志。
- [Risk] 节点内 trace 使用 wall-clock timestamp，跨容器时钟仍可能有偏差。→ Mitigation: 跨节点总耗时仍使用控制端观测；节点内 trace 只用于单节点内阶段顺序和编译耗时。
- [Risk] 事件处理和 benchmark `eth_call` 可能触发同一个 WASM 模块的惰性加载。→ Mitigation: trace 区分 activation path 和 call path；完成判据仍为 `callFunc` 返回期望值。
- [Risk] Docker stats 采样会增加少量控制端开销。→ Mitigation: 采样频率固定并记录在结果中，Lab2 只比较 WASM 算法间执行资源，不与三方对照混合。

## Migration Plan

1. 增加 OpenSpec delta、trace 写入和相关单元测试。
2. 扩展 Compose 资源限制和大规模配置生成，并验证 1/5 节点 smoke。
3. 扩展 Lab1 输出 trace 阶段字段。
4. 新增 WASM-only Lab2 命令与 Docker stats 采集。
5. 新增 runner/report，按 20 节点全算法、SchnorrProof 多规模、单节点执行效率顺序运行。
6. 运行聚焦 `go test`、`openspec validate`，再执行正式实验。

回滚：删除新增 runner/benchmark/trace 聚合代码即可；trace 文件是附加实验制品，不影响链状态和 CodeStorage 行为。
