# v1 修改计划：切换为 WASM 字节码升级路径，重构整体协议与实验论证

## 结论

需要修改，而且是**整体协议逻辑修改**，不是把现有 Go plugin 换成另一种本地产物。

`v1/review.md` 的 P0-3 / P0-4 不能再沿用“源码上传 + 各节点本地 `go build -buildmode=plugin`”来缓解。该路径把不可信 Native 代码和异构本地编译同时放进客户端地址空间，安全边界和确定性都比升级协议本身更大。后续主线改为：

```text
Solidity 上传 WASM 字节码
        ↓
链上登记 + 事件广播下发到各节点协处理器
        ↓
协处理器异步完成 WASM 校验 / 编译 / 实例化
        ↓
Activation Block 确定性激活
        ↓
调用仍从 Solidity 进入（callFunc），由协处理器在 WASM runtime 中执行
```

修改后的核心论证应收窄为：

> The Cryptographic Coprocessor treats a contract-facing cryptographic algorithm as a WASM module. Developers upload WASM bytecode through a Solidity management contract; the chain broadcasts the artifact to coprocessors; each coprocessor asynchronously validates and compiles the module outside the EVM critical path; and scheduled Activation Block selects the same version at the same height. Unprepared nodes must fail closed. Under the evaluated private-chain scenarios, Solidity remains the only contract-facing call entry, and compiled WASM modules provide a sandboxed execution path whose efficiency is compared with Precompiled Contracts and Solidity implementations.

不能再把 Activation Block 写成完整的 consensus fork 解决方案。它只解决“节点本地准备完成时间不同导致的版本选择不一致”。未准备节点、WASM runtime 差异、非法 host import 属于额外安全边界，必须用协议约束、实验补充或明确 limitation 处理。

相对 `review.md` 第 4 条“依然使用源码的思路”，本计划**明确改道**：上传物改为 WASM 字节码，不再以 Go 源码作为链上权威产物。源码只作为链下开发物，不进入升级协议。

## 新的整体逻辑

升级路径与调用路径必须分开写。升级路径换掉；调用入口保持 Solidity。

```text
                         升级路径（新）
  Developer
      │  链下将算法编译为 WASM bytecode
      ▼
  Solidity / Management Contract
      │  uploadWasmVersion(name, version, wasm, abi, gas, activationBlock)
      │  链上只登记 wasmHash、ABI、Gas、Activation Block
      │  发出 wasmVersionUploaded 事件（广播）
      ▼
  各节点 Cryptographic Coprocessor
      │  Event Listener 接收广播
      │  从交易 calldata / CodeStorage 取出同一份 WASM 字节码
      │  校验 wasmHash、size、exports、forbidden imports
      │  异步编译（AOT/JIT）并实例化到沙箱 runtime
      │  标记 prepared version
      ▼
  Activation Block 到达
      │  version selector 切换 active version
      │  本地缺失目标模块则 fail closed，不得回退旧语义

                         调用路径（保持）
  Solidity callFunc(name, input)
      ▼
  EVM dispatch / CodeStorage 入口
      ▼
  Coprocessor：按区块高度选择版本
      ▼
  WASM runtime 执行已编译模块
      ▼
  按 ABI 回写 bytes 给 Solidity
```

协处理器在该逻辑中同时承担三件事：

1. **接收广播**：在线节点听事件；离线节点按链上版本索引补齐 WASM artifact。
2. **完成编译任务**：校验、AOT/JIT、实例化、conformance vectors。编译不在升级交易的 EVM 关键路径上。
3. **承接 Solidity 调用**：EVM 不解释 WASM；合约只看见稳定 `callFunc` 接口。

## 与旧方案的差异

| 环节 | 旧逻辑（v1 现状） | 新逻辑 |
| --- | --- | --- |
| 用户上传物 | Go 源码包 | WASM bytecode |
| 链上权威身份 | 源码字节 / 弱约定 | `wasmHash` |
| 下发方式 | 事件通知后各节点拿源码 | 事件广播下发同一份 WASM 字节码 |
| 编译发生地 | 各节点 `go build -buildmode=plugin` | 协处理器对同一 WASM 做 AOT/JIT |
| 执行边界 | 进程内 Native plugin，共享 Geth 地址空间 | WASM sandbox，限制内存、fuel、host import |
| 调用入口 | Solidity `callFunc` | **不变**，仍为 Solidity `callFunc` |
| 激活规则 | Activation Block | **不变**，只保留 scheduled activation |
| 确定性主张 | “同源码本地编译” | “同一 `wasmHash` + 受控 runtime + conformance vectors” |
| Lab2 对照 | Upgrade Native ≈ Precompile | Upgrade WASM vs Precompile vs Solidity，不再默认 Native-like |

旧主线中的 `native-like execution efficiency` 不能原样保留。WASM 换来的是可分发字节码和沙箱，不是与 Go plugin / Precompile 等价的峰值性能。Lab2 必须重跑后再写效率边界。

## 当前材料中的主要冲突

| 问题 | 当前位置 | 冲突 |
| --- | --- | --- |
| 上传物仍是 Go 源码 / plugin | `paper_plan.md:24-26`、`sn-article.tex:173-203`、`cryptoupgrade/internal/activation` | 新协议的权威产物是 WASM bytecode，不是源码，也不是 `.so`。 |
| 调用与升级都按 Native plugin 叙述 | `claims.md` C1/C4/C5、`terminology.md` Native implementation | 执行单元改为 WASM module；Solidity 仍是入口，但不能再写“动态加载 Native 实现”。 |
| Lab3 写 State Root，正式数据没有 `stateRoot` | `paper_plan.md:57`、`claims.md:201-213`、`experiments.md:295, 325, 363` | `claims.md` 把 `state root` 列为 C8 证据，但正式数据没有该字段。 |
| immediate activation 仍在正文结果中 | `sn-article.tex:123, 219, 264-271, 295`、`experiments.md:266-277` | review 认为 immediate 不合法，应从正式实验和正文删除。 |
| 未就绪节点只作为设计约束出现 | `claims.md:243-263`、`sn-article.tex:210, 275` | 没有 fail-stop / recovery 实验。 |
| 安全问题只放在 future work | `sn-article.tex:297` | 旧文把 Go plugin 当 limitation；新文必须把 WASM sandbox 写成方法，而不是附录。 |
| 同源本地编译确定性没有协议化 | `sn-article.tex:199-203`、`review.md:21` | 源码路径应废弃；确定性改为 bytecode identity。 |
| Polkadot 对比会变钝 | `related_work.md:22`、原 modification_plan P1-1 | 双方都用 WASM 后，必须强调“EVM 保留 + 仅替换密码模块 + Solidity 入口”，而不是“我们也用 WASM”。 |
| cryptography 动机强于实验支撑 | `paper_plan.md:4`、`sn-article.tex:123,149,293` | 当前 7 个算法验证的是代表性密码计算，不是 post-quantum migration。 |

## P0 修改方案

### P0-1 Activation Block 不能单独解决共识分叉

**判定：需要修改核心协议表述，并补充实验。与 WASM 切换正交，但准备物改为 WASM artifact。**

论文应明确三层语义：

1. `Activation Block` 只定义版本选择规则：同一链高度选择同一目标版本。
2. 节点未完成目标 WASM 模块准备时，不能回退到旧版本继续执行；必须 fail closed。
3. 离线或落后节点恢复时，先通过链上索引补齐升级元数据和 WASM bytecode，再在协处理器中完成编译任务，然后继续执行对应高度的调用。

需要新增或强化的机制描述：

| 机制 | 论文应写清楚的内容 |
| --- | --- |
| Online path | 在线节点通过 `Event Listener` 接收 `wasmVersionUploaded` 广播，把同一份 WASM 字节码交给协处理器异步编译。 |
| Offline/recovery path | 离线节点启动或追块时查询 `Management Contract / CodeStorage` 的最新版本索引；若本地 prepared version 落后，则根据链上 `upgrade tx hash / event log / wasmHash` 重新获取字节码并编译。 |
| Activation readiness check | 高度到达 `Activation Block` 时选择新版本；若本地缺失已编译模块，返回 required-version-missing / fail closed，而不是执行旧版本。 |
| Signer behavior | 出块节点在 `Activation Block` 前未准备目标模块，应停止生产依赖新语义的区块或进入不健康状态。 |
| Safety/liveness trade-off | 协议牺牲未就绪节点 liveness，避免旧版本静默执行造成状态分歧。 |

需要补充实验：

| 实验 | 目的 | 最小设计 |
| --- | --- | --- |
| Lab3-F1：unprepared observer | 证明未就绪观察节点不会返回旧版本成功结果 | 阻塞一个 observer 的 WASM compile/instantiate；在 `Activation Block` 后执行新版本调用；记录错误状态、head 行为、恢复后追上 canonical chain。 |
| Lab3-F2：unprepared signer | 证明出块节点未就绪时不会用旧语义出块 | 在 Clique signer 上阻塞 WASM 准备；观察是否停止出块/报错/等待恢复。 |
| Lab3-F3：offline recovery | 回答“离线客户端如何解决” | 升级前停一个节点；升级后重启；节点按链上索引拉取 WASM bytecode，完成编译后再执行调用。 |

如果短期无法实现这些实验，正文必须降级为：

> The current prototype defines fail-closed behavior for unprepared nodes, but the formal evaluation does not yet validate unprepared-node recovery.

这只能缓解，不能完全解决 P0。

### P0-2 删除 immediate activation

**判定：必须修改实验设计和正文。与执行后端无关。**

后续论文只保留 scheduled activation。所有 upgrade proposal 必须指定未来区块作为 `Activation Block`，并满足最小准备窗口：

```text
activationBlock > proposalBlock
```

更稳妥的协议约束是：

```text
activationBlock >= proposalBlock + minActivationDelay
```

`minActivationDelay` 需要覆盖 WASM 广播、校验和编译的准备窗口，而不是 Go plugin 编译窗口。具体数值由 Lab1 的 event-to-all-complete 分布决定，正文不得拍一个未测量的秒数。

需要修改：

| 文件 | 修改 |
| --- | --- |
| `experiments.md` | Lab3 改为 `scheduled activation only`；immediate 标为 invalid/smoke。 |
| `claims.md` | C7/C8 不再引用 immediate；C9 是否纳入 Lab3 取决于 fail-stop 实验。 |
| `paper_plan.md` | 删除“immediate 模式”与 State Root 已验证的暗示。 |
| `sn-article.tex` | Abstract、Evaluation、Section 4.4、Conclusion 删除 `scheduled and immediate activation`。 |
| 图表 | 只展示 scheduled activation；若补做 fail-stop / recovery，再扩展。 |

禁止继续使用：

```text
immediate activation
immediate mode
scheduled and immediate activation preserved consistency
```

### P0-3 用 WASM sandbox 重写安全边界，而不是给 Go plugin 写 limitation

**判定：这是本次整体逻辑修改的主因。必须改协议，而不是只改讨论段落。**

旧计划把 Go plugin 共享地址空间写成 prototype limitation，并把 WASM 留作 future work。新计划把 WASM 提升为执行与分发载体。

应采用的安全假设：

1. 链上治理或授权账户决定“哪一份 WASM 可以被登记”，不表示任意用户提交的字节码都安全可执行。
2. 协处理器只加载通过校验的 WASM 模块；模块与 Geth 不共享任意地址空间，只能经受限 host API 交互。
3. Solidity 侧只提供登记与调用入口，不在 EVM 内解释 WASM。
4. 运行时仍必须计量：fuel/gas、memory limit、stack limit、compile timeout、artifact size limit。

应写入的防御面：

| 风险 | 需要的边界/防御 |
| --- | --- |
| 恶意 WASM | governance authorization、`wasmHash` 校验、导出符号白名单。 |
| 逃逸到宿主 | 禁止任意 host import；只允许 ABI codec / 受控 crypto host 函数（若有）。 |
| DoS via 大型字节码或编译炸弹 | artifact size limit、compile timeout、instantiation timeout、rate limit、calldata/gas pricing。 |
| 执行死循环 / 内存膨胀 | fuel metering、memory pages limit；超出则调用失败，不得拖垮 Geth。 |
| ABI 不兼容 | input/output type hash、versioned ABI、export signature check。 |
| 非确定性 host 行为 | 禁止时间、随机、网络、文件系统、线程；conformance vectors。 |
| runtime 实现差异 | 固定 WASM runtime 与编译策略；记录 runtime version / compile options。 |

正文可以声称：

```text
WASM modules execute behind a sandboxed coprocessor boundary rather than as in-process native plugins.
```

正文不能声称：

```text
dynamic loading is safe for untrusted code
on-chain governance alone guarantees code safety
the coprocessor has the same sandbox boundary as EVM bytecode
WASM is as fast as native precompiles
```

### P0-4 确定性改为 bytecode identity，不再走源码本地编译

**判定：需要改协议，而不是给 Go 源码补 `go.mod` / build manifest。**

权威产物是用户上传的 WASM bytecode，不是源码，也不是各节点 AOT 后的机器码。

| 约束 | 内容 |
| --- | --- |
| Bytecode identity | 链上记录 `wasmHash`；广播和恢复路径都必须拿到哈希一致的同一份字节码。 |
| Runtime manifest | 记录 WASM runtime 名称与版本、AOT/JIT 策略、允许的 host import 集合、内存/fuel 上限。 |
| Local preparation gate | 只有 `wasmHash`、ABI hash、export signature、test vectors 全部通过，节点才标记 prepared。 |
| Conformance vectors | 每个算法版本携带固定输入输出向量；编译并实例化后必须本地执行并记录 hash。 |
| Determinism claim boundary | 论文只声称“在同一 WASM bytecode、受控 runtime 和测试向量通过的条件下，本实验观察到一致输出”，不声称任意 WASM 引擎天然给出一致语义，也不声称 AOT 机器码字节级相同。 |

建议在 Lab3 新增字段：

```text
wasm_hash
runtime_manifest_hash
compiled_artifact_hash
test_vector_hash
conformance_ok
state_root
receipts_root
storage_value
```

`compiled_artifact_hash` 只用于审计本地编译产物，不能单独作为共识正确性证明。真正进入主结论的应是 same block height 下的 version、output、state root / storage result 一致。

## P1 修改方案

### P1-1 强化与 Phoenix / Polkadot / Precompile 的差异

**判定：需要重写 Introduction 与 Related Work。双方都用 WASM 后，差异必须写在作用域和调用边界上。**

建议对比表：

| 机制 | 升级范围 | 执行边界 | 协调方式 | 激活语义 | 本文差异 |
| --- | --- | --- | --- | --- | --- |
| Precompiled Contract | 固定 Native 函数 | 客户端内置地址 | 客户端发布 / 协议升级 | 随客户端版本 | 高效，但不能把用户提交的算法模块运行时下发到节点。 |
| Phoenix | live-upgradable blockchain client | 客户端级动态升级 | 链上代码 + JIT | 客户端升级语义 | 范围是客户端本身；本文只升级 contract-facing cryptographic algorithms。 |
| Polkadot runtime upgrade | 整条链 runtime | WASM runtime 替换链逻辑 | on-chain governance | runtime version | 同样用 WASM，但替换的是广义 runtime；本文保留 EVM，Solidity 仍是调用入口，只把密码模块下发到协处理器。 |
| EVMPatch / proxy | 合约字节码 / 代理 | contract layer | patch / deploy | 合约调用路径 | 不替换客户端侧密码执行引擎。 |
| Cryptographic Coprocessor | 面向合约的密码算法 | 稳定 Solidity 接口 + WASM coprocessor | 链上 WASM 广播 + 链下编译 | scheduled Activation Block | 窄作用域、字节码分发、异步编译、确定性激活、调用入口不变。 |

Introduction 的贡献应改成：

1. A coprocessor boundary that keeps Solidity as the contract-facing entry while executing cryptographic algorithms as replaceable WASM modules.
2. An upgrade protocol that broadcasts uploaded WASM bytecode to coprocessors, compiles it asynchronously, and activates it at a scheduled block with fail-closed unprepared nodes.
3. An evaluation that separates upgrade latency, WASM vs Precompile vs Solidity execution cost, and scheduled activation consistency.

### P1-2 cryptography / migration 动机与实验不匹配

**判定：需要二选一，推荐补实验。**

推荐方案：新增密码迁移场景实验，作为 Lab2/Lab3 的扩展或独立 Lab4。

最小可行设计：

| 项目 | 设计 |
| --- | --- |
| 服务名 | 稳定 Solidity 接口，例如 `HashDigest(bytes) -> bytes32`。 |
| v1 WASM | `Sha256.wasm` |
| v2 WASM | `Blake2bSum256.wasm` |
| 过程 | 两次 `uploadWasmVersion`，从 v1 迁到 v2，设置未来 `Activation Block`。 |
| 指标 | upgrade latency、activation 前后输出变化、state root / storage result、Upgrade vs Precompile/Contract 调用开销。 |
| 结论边界 | representative cryptographic migration，不是 post-quantum migration。 |

如果不补实验，则必须削弱题目和动机：

1. `post-quantum migration` 只作为长期动机。
2. Abstract 和 Conclusion 不能暗示已经验证 post-quantum migration。
3. 实验写成 representative cryptographic algorithms compiled to WASM。

### P1-3 Lab3 更像功能测试，不足以叫 State Consistency

**判定：必须重做或重命名。推荐重做。**

若标题继续使用 State Consistency，需要改为 state-changing transaction 设计：

| 项目 | 新设计 |
| --- | --- |
| 调用方式 | 在 `Activation Block - 1`、`Activation Block`、`Activation Block + 1` 发送真实交易。 |
| 记录对象 | block hash、state root、receipts root、receipt status、logs、storage slot、selected version、output、`wasmHash`。 |
| 验证合约 | probe/verifier contract 将 `callFunc` 返回值和版本写入 storage。 |
| 节点比较 | 对同一 canonical block，比较所有节点的 `stateRoot`、`receiptsRoot`、storage value。 |
| 异常场景 | unprepared-node fail-closed 和 recovery。 |
| 模式 | 只保留 scheduled activation。 |

若无法重做，则论文必须把 Lab3 改名为：

```text
Version and Output Consistency During Scheduled Activation
```

并删除 State Root 一致的 claim。

### P1-4 私有链规模需要扩大，且全部实验必须按 WASM 路径重跑

**判定：需要补规模实验；WASM 切换使 Lab1/Lab2/Lab3 的旧 Native plugin 数据全部失效。**

| 实验 | 当前 | 修改 |
| --- | --- | --- |
| Lab1 | 5 节点，7 算法，Go source + plugin 编译 | 改为 WASM 上传 / 广播 / 编译；保留 5 节点并新增 10 节点；拆分 broadcast、validate、compile、instantiate。 |
| Lab2 | Upgrade Native vs Precompile vs Solidity | 改为 Upgrade WASM vs Precompile vs Solidity；不得沿用“接近 Native Precompile”的旧结论。 |
| Lab3 | 5 节点，scheduled + invalid immediate | 10 节点 scheduled state-consistency；加入 fail-stop/recovery。 |

图表建议：

1. Lab1：`5 nodes vs 10 nodes` 的 upgrade latency，并单独给出 WASM compile 时间。
2. Lab2：WASM / Precompile / Solidity 的 latency 与 `gasEstimate`。
3. Lab3：scheduled activation timeline，展示 version、`wasmHash`、stateRoot、output、fail-closed/recovery。
4. immediate 旧图和 Go plugin 旧图不再进入正文。

## 文件级修改计划

### 1. `paper_plan.md`

必须修改：

1. 研究问题 1 改为：如何在保持 Solidity 调用入口的前提下，把密码算法实现替换为可广播、可编译的 WASM 模块。
2. 研究问题 3 改为：如何在协处理器异步编译和未就绪节点存在时，避免旧版本静默执行并保持可验证的一致状态。
3. 核心设计改为：WASM 字节码上传、事件广播下发、协处理器编译、Solidity 调用入口不变。
4. 算法准备表从 `.go` 入口改为 `.wasm` 模块导出；源语言只作为链下编译说明。
5. 实验三改为 scheduled activation + state-changing transaction + fail-stop/recovery。
6. 删除 Native plugin / 源码本地编译 / State Root 已验证的完成式表述。

### 2. `claims.md`

必须修改：

1. C1：解耦对象从 Native implementation 改为 WASM module；调用入口仍是 contract-facing Solidity interface。
2. C2：运行时升级的对象是 WASM bytecode，不是 plugin 共享库。
3. C3：异步准备写清为 WASM validate/compile/instantiate，且仍不在升级交易 EVM 关键路径上。
4. C4：删除“retain native precompile performance”；改为测量 WASM 路径相对 Precompile / Solidity 的开销，结论以新 Lab2 为准。
5. C5：保留“复杂密码计算优于 Solidity”的可能性，但证据改为 WASM 执行而非 Native plugin。
6. C7：Activation Block + fail-closed；准备完成以 WASM 模块 prepared 为准。
7. C8：补采 `stateRoot` 才保留 State Consistency，否则降级。
8. C9：完成 fail-stop 实验才能进入主结论。
9. 新增安全与确定性 claim：
   - bytecode-centered identity（`wasmHash`）
   - sandboxed WASM execution
   - no untrusted native plugin claim
   - no “same source compiles the same” claim

### 3. `experiments.md`

必须修改：

1. 升级交易字段从 encoded source 改为 WASM bytecode / `wasmHash`。
2. Lab1 完成判据仍可以是 `callFunc` 成功，但阶段拆分必须包含 compile/instantiate。
3. Lab2 对照路径改为 WASM Upgrade / Precompile / Solidity。
4. Lab3 删除 immediate；新增 `stateRoot`、`receiptsRoot`、storage、`wasmHash`。
5. Lab3 新增 fail-stop / offline recovery。
6. Lab1 增加 10 节点与重复轮次。
7. 增加 cryptographic migration scenario，或列为未完成且正文降级。
8. 明确旧 Go plugin JSON/CSV 不得写入新正文；新数值必须来自 WASM 路径结果文件。

### 4. `sn-article-template/sn-article.tex`

必须修改：

1. Abstract：删除 plugin / immediate / native-like 默认表述；改写为 WASM bytecode broadcast + coprocessor compilation + Solidity entry。
2. Introduction：动机可保留 cryptography，但贡献按新三点重写。
3. Section 3 按新数据流重写：
   - upload WASM bytecode
   - event broadcast to coprocessor
   - asynchronous compilation
   - Solidity `callFunc` invocation
   - scheduled activation
   - fail-closed / offline reconciliation
   - threat model：WASM sandbox，而不是 Go plugin limitation
4. Section 4：三个实验全部按 WASM 新数据重写。
5. Related Work：Polkadot 对比改为“同为 WASM、不同升级范围与调用边界”。
6. Conclusion：删除 immediate、universal consistency、untrusted native safety、post-quantum migration proof。

### 5. `terminology.md`

建议修订或新增：

| 英文 | 中文 |
| --- | --- |
| WASM bytecode | WASM 字节码 |
| WASM module | WASM 模块 |
| wasm hash | WASM 哈希 |
| bytecode broadcast | 字节码广播 |
| coprocessor compilation | 协处理器编译 |
| AOT compilation | 提前编译 |
| WASM runtime | WASM 运行时 |
| host import policy | 宿主导入策略 |
| fuel metering | 燃料计量 |
| sandboxed execution | 沙箱执行 |
| fail-closed activation | 失败关闭式激活 |
| upgrade reconciliation | 升级状态对账 |
| conformance test vectors | 一致性测试向量 |
| trusted upgrade governance | 可信升级治理 |
| contract-facing call entry | 面向合约的调用入口 |

删除或降级不再作为主术语的用法：`plugin`、`shared library`、`go build -buildmode=plugin`、`source hash` 作为链上权威身份。

`Native implementation` 不再描述升级后的执行单元；如需保留，只用于 Precompiled Contract 对照路径。

### 6. 实现与实验代码（需另开 OpenSpec change）

论文计划依赖实现切换。建议拆 change，不要混在论文润色里：

| change 名称 | 目标 |
| --- | --- |
| `replace-plugin-with-wasm` | 上传物、存储、事件、协处理器编译/运行时替换 Go plugin |
| `revise-upgrade-activation` | scheduled-only、fail-closed、offline reconciliation |
| `strengthen-consistency-evaluation` | Lab3 state-changing + stateRoot |
| `add-cryptographic-migration` | Lab4 代表性密码迁移 |

实现约束：

1. Solidity 入口保持 `callFunc(name, input)`，避免合约侧调用形状被升级路径绑死。
2. `uploadCodeVersion` 改为接收 WASM bytecode；链上事件必须能让所有协处理器拿到同一 `wasmHash`。
3. 编译任务只在 coprocessor 后台执行，升级交易的 EVM 路径仍然只做登记和事件。
4. 最小侵入 Geth：继续走现有 coprocessor / CodeStorage 边界，替换 `compiler` + `pluginruntime`，而不是改 EVM 解释器。

## 新实验设计草案

### Lab1-W：WASM 升级延迟与广播编译

目标：证明上传、广播、编译脱离升级交易执行关键路径，并给出规模趋势。

设计：

1. 网络：5 节点 + 10 节点。
2. 上传物：各算法的 WASM bytecode，而不是 `.go`。
3. 算法：至少 `Add`、`Sha256`、`Blake2bSum256`、`SchnorrVerify`；资源允许覆盖全部 7 个。
4. 轮次：每个算法每个规模至少 3 轮。
5. 指标：`submitToReceiptMillis`、`eventToAllCompleteMillis`、`compileMillis`、`instantiateMillis`、`submitToAllCompleteMillis`、node completion spread。
6. 结论：只能说明 evaluated private-chain scale 下的升级准备延迟，不能外推到公链；不能把 compile 时间写成交易执行阻塞时间。

### Lab2-W：WASM / Precompile / Solidity 执行效率

目标：重新标定效率，不再沿用 Native plugin ≈ Precompile。

设计：

1. 三条路径：WASM coprocessor、Precompiled Contract、Solidity contract。
2. 调用入口：WASM 与 Solidity 对照都从合约 ABI 进入；Precompile 走原地址。
3. 指标：`eth_call` latency、`gasEstimate`。
4. 允许结论：复杂算法上 WASM 路径低于 Solidity 开销；与 Precompile 的差距按数据写，不预设“接近 Native”。
5. 若 AOT 后某些算法接近 Precompile，只能写 observed comparable performance under the evaluated runtime，不能推广到所有 WASM 引擎。

### Lab3-S：Scheduled State Consistency

目标：把 Lab3 从功能测试升级为 state consistency 实验。

设计：

1. 只使用 scheduled activation。
2. 在 `H_activation - 1`、`H_activation`、`H_activation + 1` 发送 state-changing transaction。
3. 交易调用 `CodeStorage.callFunc`，并将返回值、版本和 `wasmHash` 写入 probe contract storage。
4. 每个节点采集 canonical block 的 `blockHash`、`stateRoot`、`receiptsRoot`、receipt status、logs、storage value。
5. 判定：

```text
stateRoot_1(H) = stateRoot_2(H) = ... = stateRoot_N(H)
storageValue_1(H) = storageValue_2(H) = ... = storageValue_N(H)
selectedVersion(H) = expectedVersion(H)
wasmHash(H) = expectedWasmHash(H)
```

### Lab3-F：Fail-Closed and Recovery

目标：回答“节点更新失败如何解决”。

设计：

1. 阻塞一个节点的 WASM compile/instantiate，使其在 `Activation Block` 到达时缺失目标模块。
2. 该节点执行或验证需要新版本语义的调用时，应返回 required-version-missing / unhealthy / fail closed。
3. 恢复后通过链上 version index 和 event metadata 补齐 WASM bytecode，完成编译并追上 canonical chain。
4. 记录 fail-closed 时间、recovery latency、恢复后的 state root / chain-view 一致性。

### Lab4-M：Representative Cryptographic Migration

目标：让 cryptography/migration 动机有实验支撑。

设计：

1. 定义稳定 Solidity 接口 `HashDigest(bytes) -> bytes32`。
2. 上传 v1 `Sha256.wasm`，再上传 v2 `Blake2bSum256.wasm`。
3. 通过 scheduled upgrade 从 v1 迁移到 v2。
4. 在 activation 前后采集输出变化、version、`wasmHash`、stateRoot、upgrade latency、steady-state latency / gasEstimate。
5. 结论写为 representative cryptographic migration，不写 post-quantum migration proof。

## 论文表达的硬性边界

必须删除或避免：

```text
Immediate activation is a valid mode.
Activation Block solves consensus forks by itself.
Developers upload source code and each node compiles a native plugin.
Same source code compiled locally naturally guarantees deterministic behavior.
Dynamically loaded Native implementations retain performance comparable to Precompiled Contracts.
The current Lab3 proves State Root consistency.
The current prototype validates fail-stop behavior.
The system provides zero-downtime upgrade or no availability impact.
The Go plugin mechanism is safe for untrusted code.
WASM execution is as safe as the EVM.
The experiments validate post-quantum migration.
```

可以保留但要加限定：

```text
Solidity remains the contract-facing call entry.
The chain broadcasts WASM bytecode to coprocessors.
Compilation and instantiation are outside the EVM execution path of the upgrade transaction.
Activation Block provides deterministic version selection under the same chain view.
Unprepared nodes must fail closed rather than execute obsolete semantics.
WASM modules execute in a sandboxed coprocessor runtime with explicit resource limits.
Execution efficiency is reported relative to Precompile and Solidity after the WASM path is re-evaluated.
State consistency is supported only after stateRoot/storage evidence is collected.
```

## 执行顺序

1. 先改 `paper_plan.md`、`claims.md`、`terminology.md`、`experiments.md`，把协议从“源码 + plugin”锁成“WASM 广播 + 协处理器编译 + Solidity 入口”。
2. 实现按 OpenSpec change 进行，优先 `replace-plugin-with-wasm`，再 `revise-upgrade-activation`。
3. 旧 Go plugin 实验结果全部降为历史数据，不进入新正文。
4. 实现可运行后按优先级重跑：
   - Lab1-W scheduled WASM 升级延迟（5 节点先跑通，再 10 节点）。
   - Lab2-W WASM / Precompile / Solidity。
   - 删除 immediate 后的 Lab3-S。
   - Lab3-F fail-closed / offline recovery。
   - Lab4-M representative cryptographic migration。
5. 新 JSON/CSV 生成后再更新图表与 `sn-article.tex`。
6. 最后执行 LaTeX 编译、引用检查、TODO 检查、术语一致性检查。

## 最小可接受版本

如果时间不足，最低限度也必须完成：

1. 协议和正文改为 WASM bytecode 上传、广播下发、协处理器编译；调用仍从 Solidity 进入。
2. 删除 Go plugin / 源码本地编译作为主机制的表述。
3. 删除 immediate activation。
4. 将 C4 从 Native-like 降级为“待 WASM Lab2 重测”；将 C8 从 State Root consistency 降级，或补采后再保留。
5. 在方法中写清 fail-closed activation、offline recovery，以及 WASM sandbox / host import / fuel 边界。
6. 把 post-quantum migration 降级为 motivation。

最低限度版本仍不能完全解决 P0-1 和 P1-3。要真正解决 `review.md` 中的核心问题，应完成 Lab3-S 与 Lab3-F，并且 Lab1/Lab2 必须使用 WASM 路径新数据。
