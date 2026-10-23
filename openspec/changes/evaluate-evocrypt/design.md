## Context

**实施进度（2026-10-23）：** 用户随后授权执行单节点实验并将递增计费接入链上；下文是原始预注册协议的历史版本。实际结果以 `experiments/cryptoupgrade/results/chapter-five/20261023-single-node-billing/report.md` 为准；仅在创世配置中启用固定 `M(C)=1`，并通过中性自转账/空父块核查 `parentCalls=0`。通用父区块负载与重组/重启一致性仍属 `adjust-algorithm-gas-pricing` 未完成范围。原始协议中的“直接父块必须含自转账”遇到周期 0 出空块后改为核验父块为空或中性，逐块逆溯至分隔交易；档位、费率和迭代次数没有因此变化。

技术栈：Go/Geth、Solidity/Foundry、TinyGo/WASM、wazero、Python 绘图、OpenSpec；实验代码及原始数据放 `experiments/cryptoupgrade`。参见 `proposal.md` 和 `specs/evocrypt-chapter-five-evaluation/spec.md`。本文件是**待实施的实验协议**，不是实验结果。

当前 `benchexecutionefficiency` 已有 PBKDF2 三路径 fixture、10 次预热、100 次 `eth_call` 计时与 `eth_estimateGas`，但迭代次数固定为 2，且 fixture 指向已迁移的 `src/archive/Pbkdf2Sha256.sol`（实际文件在 `src/Pbkdf2Sha256.sol`）。Solidity 合约用 `sha256` 预编译实现 HMAC 内层哈希，仍由 Solidity 实现 PBKDF2/HMAC 的循环和缓冲处理；必须披露此依赖。`benchtxgas` 有单次 receipt 和固定 500 次的旧比较；`benchgascomplexity` 的 PBKDF2 100/500/1000/2000 档为 WASM 热运行时间，不含 RPC/交易；`benchcallgas`/`benchcomparegas` 只有离线动态报价和旧费率真实收据。`adjust-algorithm-gas-pricing` 的计价器已存在，但链上集成、激活配置、父区块计数与真实收据尚未完成（其 tasks 1.3–1.4、2.3、3.x 等未勾选）；不能直接开展 5.2.2 的 EvoCrypt 实测。

## Goals / Non-Goals

**Goals:** 逐样本可追溯的同任务三路径耗时与收据 Gas、可证实的交易内计费曲线、最少补测的功能证据、按 5.1–5.4 撰写的条件化结论。按用户后续约束，所有**新增链上实验只启动单节点**；历史多节点证据只读复核，不新建 20 节点链。

**Non-Goals:** 不测部署/升级成本；不把旧 `gasEstimate` 当收据、不把节点内部耗时同 RPC 耗时直接求比；本提案不实现计价共识变更。本阶段不修改 `paper/`（其现有结构约定与本次第五章结构不一致，正式写作前需单独确认）。

## Decisions

### 5.1 实验设置：固定实验契约

- 统一在同一 Geth 构建、链规则、EVM fork、节点硬件与负载、编译器与配置快照下执行；记录 CPU/内存/OS、Go/TinyGo/solc/Foundry 版本、commit、WASM/合约哈希、链 ID、区块 Gas 上限、目标区块 hash、Gas profile 哈希与激活高度。避免把不同日期/机器或不同模块哈希的结果相减。
- 两类自变量分别编码为 `iterations`（一次 PBKDF2 内部循环）和 `callsPerTx`（同一交易中重复调用）；固定 `password="password"`、`salt="salt"`、`keyLength=32`、迭代次数以外 ABI 输入，以及每档的有效输出。用独立 Go PBKDF2 参考向量和三路径解码结果逐档比对；只比较具有同一 HMAC-SHA256 语义的路径。
- 固定一次调用的比较边界：主指标为预热后 `eth_call` 从客户端 RPC 发送前到响应到达的墙钟延迟，包含 RPC/ABI/EVM/运行时，不含合约部署、上传、编译及预热；另采集同条件 `receipt.gasUsed`（成功真实交易）作为链上费用，**不能把该收据执行时间当作 `eth_call` 延迟**。已有 `coprocessor_exit.durationNs` 只作分层诊断，与 native 对照若无等价节点内边界不得直接求倍率。`eth_estimateGas` 仅作辅助。
- 每档每路径先预热 10 次，至少 3 个独立轮次、每轮至少 30 个成功 `eth_call` 样本；每轮交错或固定种子随机化路径/档位顺序，记录失败、超时和区块高度。记录每档每路径逐样本 Gas 收据至少 3 笔独立成功交易；若 Gas 随状态/计数变化则每笔满足相同父区块条件，不能混合。以轮内中位数、跨轮中位数及 p5/p95（明确样本级定义）汇总；保留样本而非只留平均值。独立轮次从相同链快照或等价干净状态开始。

### 5.2.1 调用效率：同任务 PBKDF2

- 预注册候选档位 `1, 2, 5, 10, 20, 50, 100` 次，涵盖固定开销主导与迭代主导区间；选择并非依据期望曲线。先做三路径正确性、EVM 执行上限和 Gas 安全预检。Solidity 最大迭代 10000、预编译最大 1000000，但有效档位由三路径**共同成功**且预先约定的区块 Gas/时间预算决定。若某档失败，保留失败原因与该点，图上注明不可比较；若确需提高预算，重启全部路径并版本化协议，不仅更换一条曲线的参数。
- 用现有 fixture/`Pbkdf2Sha256.sol`、`cryptoupgrade/precompile.go` 的 native PBKDF2 和 WASM 模块；修正已迁移的 Solidity 源路径并确认 SHA-256 预编译调用在所有路径是同一加密任务。无套利解释：各路径 Gas 计价政策独立；按迭代档同时报告 `gasUsed`（含 21000 基础与 calldata、封装）、可用时的执行 Gas 差分、`eth_call` 时间及每迭代的增量/趋势。若 EvoCrypt 固定 metadata Gas 不随迭代增长，诚实报告不匹配，禁止借此称计算更快。
- 主图 x=`iterations`、y=跨轮中位 `eth_call` ms，三条曲线带波动区间及失败点；伴随表/辅图给出每档三个路径 receipt Gas，注明上下文、模块哈希、定价模型与计时边界。不得把 100–2000 次 WASM-only 历史标定数据冒充三路径新实验。

### 5.2.2 调用计费：同交易重复调用

- 固定上节审定的同一输入（初始预注册 10 次内部迭代，若预检不能满足所需档位则记录并在采样前统一重定版本），对照 Solidity 与 EvoCrypt 共享一份实验驱动合约/等价合约方法：一次外部交易循环 `n` 次，每次使用相同 ABI 输入、校验同一结果并防止编译器消除调用；EvoCrypt 经 `callFunc` 进入 WASM，Solidity 经 PBKDF2 合约逻辑进入循环。固定 `n=1,2,3,4,5,6,8,12,16` 作为预注册网格；实际覆盖必须含低次数和 profile `N₀` 阈值两侧（若该配置阈值超出网格，在采样前扩充并保存新协议版本）。超出 Gas 上限的档位记录 OOG/不可观测上界，不删去或调参伪造曲线。
- **先决条件**：`adjust-algorithm-gas-pricing` 的真实链上计数、固定共识 profile 和显式激活经测试且可从交易/trace 核验；记录 `g₀(iterations)`、`N₀`、`α`、`M(0)=1` 及 profile 哈希；这些参数来自独立标定与审阅，不为了贴合 Solidity 曲线反向设定。对每个目标区块以受控出块器确保恰好一笔目标交易且无其他交易，父区块的**算法调用计数为零**（不是目标块内调用数为零），核查父区块哈希、计数及中性 `m=1`；配置失败的样本剔除并留痕重跑，不声称该配置可以跨交易防滥用。
- 每档每路径分别预热 10 笔（不计入测量；也必须隔离或重置链上下文），至少 3 个独立轮次、每轮至少 10 笔成功真实交易，轮次间恢复相同状态；预检区块容量。逐笔记 `receipt.status`、`gasUsed`、块交易列表、父块哈希及计数、n 个调用的返回/输出 hash、trace 可获得的逐次报价。用总 `gasUsed` 作为累计 Gas 主图；另报告 n=0 基线封装（如果可执行）及超基线 Gas，不能误称为“算法纯费用”；不得将 `n × 单次 eth_estimateGas` 当真实累计值。
- 图 x=`callsPerTx`、y=跨轮中位 `receipt.gasUsed`，两路径和波动区间；标注 `N₀` 与实际低次数单次费用、有效最高 n/OOG，附每次 EvoCrypt 账单和总收据的可解释差异。结果只说明同交易、同算法的重复计费；跨交易、父区块负载或攻击抑制另需实验。

### 5.3 功能验证：复用审计矩阵

| 子项 | 直接复用（保留原口径） | 需要调整/核对 | 缺失与补测门槛 |
| --- | --- | --- | --- |
| 5.3.1 激活块前后与多节点 | `output/results/upgrade-consistency-period5-20261007-095348/analysis/summary.json`：20 节点、窗口覆盖率 1.0、版本/输出/相同交易与 state root 检查均真；同日 `upgrade-join-scenario-20261007-103613` 摘要亦报告覆盖率 1.0 | 查原始 `run.json`、节点/区块哈希、激活高度、版本和 status；旧的 `upgrade-consistency-final` 与 `...093059` 覆盖率 0，不采信；区分机制测试与环境中的人工延迟屏障 | 加入后同步：join 记录没有 late-start/同步时间字段，node20 在区块 11 已有观测，不能仅凭目录名和摘要断言“激活后加入”；需 docker 启动/停机与同步高度证据；确实无证据才补跑 late-start 场景并比较同一区块的已在线节点 |
| 5.3.2 基础算法 | `benchexecutionefficiency` 的八算法候选及历史 Lab2/`tx-gas` 数据可作 fixture 与方法参考；`benchgascomplexity` 保留 WASM 热执行曲线 | 核对每种算法实际成功的 WASM/统一接口、native 输入/输出、运行时和原始样本、`eth_call`/节点内时间口径；路径过时的 PBKDF2 Solidity 源地址需修复；旧 Go plugin 结果不作为 WASM 证据 | 对未同时具备同向量正确性和 native/WASM 同边界时间的**具体算法**补测；不能拿独立 WASM 热运行同 RPC 时间相除 |
| 5.3.2 后量子/零知识验证 | `results/{aigis_sig2_verify,dilithium3_verify,ml_dsa_65_verify,slh_dsa_shake_192f_verify,groth16_bls12381_verify,groth16_bn254_verify}/20261005T022045Z/result.json` 已有向量 hash、两路径记录及对比；`benchprecompile` 可复用 | 按每份原始 `comparison`、输出真值/一致性、模块/输入 hash、预热/样本与口径逐项审核；同一文件 node 内 vs RPC 倍率不同，BLS12-381 的 RPC 倍率约 25.26，不得套用旧文 1.3–9 倍的表述 | 未通过核对的算法单独补正确性或匹配边界的计时；无负例不得声称拒绝伪造签名/证明的全面安全性 |

5.3.1 新增单节点运行只核对 `H-1/H/H+1` 的版本和输出；历史多节点摘要只读核对同块 hash 与输出。**单节点无法证明后加入节点从另一节点同步**；若历史记录缺乏可核验启动/同步时间线，须写成“尚未验证”，不在单节点数据上伪称验证。5.3.2 用一张按算法分类的表列每项输入/输出向量、模块 hash、原生参考、计时边界、中位数、开销比和证据路径；适用时直接复用无需重跑。

### 5.4 本章小结与数据组织

输出 `experiments/cryptoupgrade/results/chapter-five/<run-id>/` 下的 `manifest.json`（预注册档位与原因、commit/版本/profile/环境/计时边界）、`samples.jsonl`（一行一个 `runId,section,round,path,iterations,callsPerTx,measurementKind,durationNs,gasUsed,txHash,blockHash,parentHash,parentCalls,receiptStatus,outputHash,error`，不适用字段为 null）、`summary.csv`（按独立变量与 path 汇总样本数、中位数/区间、失败数）、`evidence-matrix.md`、`figures/*.svg`/`*.png` 和生成脚本。发布前从原始记录重算摘要及图点；目录标识原始/历史复用和新增数据的来源，不覆盖旧结果。第五章小结只引用已过验收项、报告失败/限制，不补写升级成本。

## Risks / Trade-offs

- [Risk] 动态报价器已实现但链上规则未接入。→ 保持 5.2.2 EvoCrypt 为阻塞，先完成独立计费 change 的共识集成及验证；绝不用反事实估算冒充收据。
- [Risk] Solidity `sha256` 使用原生哈希原语且 Gas 与 WASM 计价模式不同。→ 明示任务语义、原语依赖及成本构成，时间与收费分开下结论。
- [Risk] 已保存的 join 摘要不足以证明激活后加入，旧实验/论文版式与本需求矛盾。→ 查证时间线；无证据才补测；正式改写 `paper/` 前由作者确认目标稿件结构，不影响当前实验规范。
- [Risk] 人工调价、区块拥挤、Gas 上限、异步出块或 RPC 波动产生假曲线。→ 采样前锁定网格/配置，逐样本验证父块、状态与输出，独立重复、保留 OOG 与异常点。

## Migration Plan

先审核复用材料及依赖状态，再补实验夹具/采样与绘图；5.2.2 只有在计费变更验证完成后才能采链上结果；最后合成第五章。新脚本/结果独立命名、旧目录只读，回退时删除新实验入口而不改变 Geth 既有语义。
