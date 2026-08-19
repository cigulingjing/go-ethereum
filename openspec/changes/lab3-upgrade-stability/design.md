## Context

技术栈为 Go、Solidity、ethereum/go-ethereum、Clique 私有链、JSON/CSV 实验结果输出。

实验 2 已经形成升级后执行效率基准，`lab1-upgrade-latency` 已经提供多节点私链中从升级交易提交到各节点 `callFunc` 成功的观测方式。当前运行时已经把 `uploadCode` 的链上提交与本地 `ActivateAlgorithm` 解耦：`uploadCode` 写入待激活 metadata 并发出 `codeUploaded` event，节点事件监听线程再执行本地激活。

本 change 关注另一个问题：升级交易作为链上交易进入私有链后，是否会破坏或扰动区块链一致性。现有 `CodeStorage` 合约和 `common.CodeStorageABI_json` 仍是单版本模型，只包含 `uploadCode(name, code, gas, itype, otype)`、`getInfo(name)` 和 `callFunc(name, input)`；事件也只有 `codeUploaded(string name)`。如果要验证“指定区块启用新算法”或“切换到新算法使用”，需要在合约 metadata、EVM dispatcher、事件解析和实验命令中补齐版本号与生效区块的最小语义。

## Goals / Non-Goals

**Goals:**

- 验证当前 `CodeStorage` 合约、ABI、EVM dispatcher、event activation 和实验工具是否支持合约触发升级、链上版本管理、指定区块生效和立即切换。
- 在不改变 Geth 共识规则的前提下，为动态升级路径补齐必要的版本 metadata、事件字段和按区块选择版本逻辑。
- 基于已有私有链多节点环境，设计并实现升级稳定性实验入口。
- 证明升级交易被所有目标节点导入后，节点间 head、block hash、receipt、event、版本视图和算法输出保持一致。
- 输出可复现的 JSON/CSV 原始结果和摘要，用于论文中的“升级完整性/一致性”论证。

**Non-Goals:**

- 不重新设计实验 2 的执行效率对比，也不把执行时间和 Gas 作为本实验主结论。
- 不实现 rollback、升级权限治理、多签审核、链重组恢复或故障注入。
- 不修改 Clique 共识规则、交易排序规则、precompile 行为或既有算法 ABI。
- 不要求所有节点在同一物理时刻完成本地 plugin 编译；实验只要求在同一区块高度语义下观察到一致版本和一致输出。

## Decisions

1. 先做功能审计，再做最小补齐

   实现阶段先检查 `cryptoupgrade/contracts/code_storage.sol`、`common.CodeStorageABI_json`、`cryptoupgrade/internal/evm/code_storage.go`、`cryptoupgrade/internal/event/service.go` 和 `experiments/cryptoupgrade/bench/cmd/benchupgradelatency`。审计结论必须明确记录哪些能力已存在、哪些能力缺失。当前预期缺口是合约侧版本号、生效区块、版本查询 API、事件字段和 `callFunc` 的按区块版本选择。

   Alternative considered: 直接编写实验脚本假定版本管理已存在。该方案会把实现缺口误解释为实验失败，不利于定位一致性问题。

2. 合约侧作为升级触发器和版本管理模块

   `CodeStorage` 保留链上提交入口，同时新增版本化 metadata。建议将每个算法扩展为按版本保存实现，记录 `version`、`code`、`gas`、`itype`、`otype`、`activationBlock` 和 `status`。升级交易必须发出包含算法名、版本号和生效区块的事件。查询 API 应能返回指定版本和当前区块应使用的版本。

   Alternative considered: 只在 Geth 本地 repository 管理版本。该方案无法让多节点从同一链上状态推导同一版本，不能用于一致性论证。

3. 支持 scheduled 和 immediate 两种切换策略

   `scheduled` 模式规定从 `activationBlock` 开始使用新版本；在该区块之前，`callFunc` 仍返回旧版本结果或在无旧版本时保持不可调用。`immediate` 模式可建模为 `activationBlock <= 当前区块号`，升级交易被包含后后续调用立即选择新版本。两种策略都通过同一链上 metadata 表达，避免引入两套调用路径。

   Alternative considered: 为立即切换新增独立 ABI 方法。该方案会扩大 ABI 面，且本质仍是生效区块为当前或下一可执行区块。

4. EVM 调用边界显式传入 block number

   版本选择必须基于 EVM 正在执行的区块号，而不是节点本地 wall-clock 或事件到达时间。`core/vm/contracts_cryptoupgrade.go` 调用 `cryptoupgrade.RunCodeStorageCall` 时需要把当前 `BlockContext.BlockNumber` 传入 dispatcher；dispatcher 再用该高度解析 `getActiveVersion` 和 `callFunc`。这样同一个区块在所有节点重放时会选择同一版本。

   Alternative considered: 在事件监听线程激活完成后直接覆盖本地 active map。该方案可能导致不同节点因监听速度不同而在同一高度选择不同版本。

5. 本地 activation 只准备可调用 artifact，不决定链上生效语义

   事件监听线程收到升级事件后，应按版本保存本地源码、`.so` 和 metadata。它可以在生效区块前提前编译新版本，但 `callFunc` 是否使用该版本仍由链上 `activationBlock` 和当前 block number 决定。若某节点在生效区块到达时尚未完成编译，该节点应报告本地激活失败或不可调用，不能静默回退到旧版本并声称成功。

   Alternative considered: 只有到生效区块才开始编译。该方案会把编译耗时推迟到用户调用路径，降低实验稳定性。

6. 稳定性实验以链一致性和版本一致性为主指标

   新增或扩展实验命令，建议命名为 `benchupgradestability`，复用 `experiments/cryptoupgrade/network` 配置和 RPC 观测。每轮实验提交一次升级交易，然后在所有目标节点记录：

   - head number/head hash 是否最终一致；
   - 升级交易 receipt 是否一致；
   - receipt 中升级事件字段是否一致；
   - 指定区块前后 `getActiveVersion` 或等价查询结果是否一致；
   - 生效区块前后 `callFunc` 输出是否与预期版本一致；
   - 节点本地 activation 成功/失败状态和错误原因。

   Alternative considered: 只检查所有节点能否最终调用新算法。该标准无法发现生效区块之前提前切换、事件字段不一致或部分节点链视图分叉。

7. 实验输出保留实验 2 可复现实践

   JSON 记录完整配置快照、节点列表、算法 fixture、版本计划、交易 hash、receipt、事件字段、每个观测高度的版本和输出；CSV 按 node/height 展开，便于绘制一致性表。文本摘要只报告通过/失败、失败维度、最早/最晚稳定区块和涉及节点。

   Alternative considered: 只输出人工日志。该方案不适合论文复核，也难以和实验 1、实验 2 的结果组织保持一致。

## Risks / Trade-offs

- [Risk] 修改 `CodeStorage` ABI 会影响已有实验命令。→ Mitigation: 优先保留现有 `uploadCode` 作为兼容入口，新增版本化入口或 wrapper；必要时同时更新 `common.CodeStorageABI_json` 和相关测试。
- [Risk] `eth_call` 默认 latest 时不同节点高度可能短暂不同。→ Mitigation: 实验在指定高度或等待所有节点 head hash 一致后再采样，并在结果中记录实际 block number/hash。
- [Risk] 本地 plugin 编译速度差异可能被误解为链不一致。→ Mitigation: 把链上 receipt/event 一致性和本地 activation/callFunc 可用性拆开报告。
- [Risk] 同名算法历史版本或 plugin cache 污染多轮实验。→ Mitigation: 每轮使用唯一版本号和结果目录，提交前检查目标版本不存在。
- [Risk] 版本切换语义与现有单版本 repository 不兼容。→ Mitigation: 先增加最小版本 metadata 层，并让无版本调用默认解析为当前 active version，降低对旧调用方的影响。

## Migration Plan

1. 完成功能审计，记录当前合约、ABI、EVM dispatcher、event listener 和实验命令对目标能力的支持情况。
2. 最小扩展 `CodeStorage` 合约和 `common.CodeStorageABI_json`，增加版本号、生效区块、版本查询和事件字段。
3. 扩展 EVM dispatcher 与 repository，使上传、查询、事件激活和 `callFunc` 能按版本和 block number 工作。
4. 新增升级稳定性实验命令，复用多节点私链配置和已有算法 fixture。
5. 在本地 2 节点私链上完成 scheduled/immediate smoke test，再扩展到实验 2 使用的私链规模。
6. 若出现不可接受回归，保留新增实验文档，回滚 ABI/dispatcher 改动到单版本语义，并将版本管理拆成后续独立 change。

## Open Questions

- 版本号由用户显式传入，还是由合约按算法名递增生成。
- `activationBlock` 默认应为当前交易所在区块、下一块，还是由调用方强制指定未来高度。
- 正式实验是否需要把短暂分叉或节点重启纳入稳定性边界，还是先限定为无故障私有链。
