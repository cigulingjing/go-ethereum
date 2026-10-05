# 第四章安全性分析修改计划

## 1. 修改目标

针对审阅意见“安全分析过于朴素，需要加入理论论证、安全定义和安全定理”，将 `paper/Springer LNCS/evocrypt.tex` 的第四章从“攻击场景 + 工程防护措施”的描述性分析，提升为基于系统模型、对手模型、安全定义、定理和证明边界的形式化安全分析。

本次修改不改变 EvoCrypt 的核心设计、实验方案或已有实验结论，重点是：

- 将已有的 `Active version`、链上元数据、WASM 哈希校验、Activation Block 和确定性 Gas 计价提升为可引用的形式化对象；
- 证明 EvoCrypt 在给定信任假设下具备模块绑定安全、元数据一致性、激活一致性和调用确定性；
- 对 WASM 资源限制给出条件性安全结论，但不将其扩大为完整的网络级拒绝服务证明；
- 明确委员会安全、共识安全、密码学算法正确性和运行时实现正确性属于外部假设或分析边界。

## 2. 修改范围

### 2.1 主要文件

- 修改目标文件：`paper/Springer LNCS/evocrypt.tex`
- 当前计划文件：`paper/v2/plan.md`

### 2.2 章节范围

重点修改：

```text
\section{Security Analysis}
```

不修改以下内容的核心结论：

- EvoCrypt 的整体架构和协议流程；
- Section 5 的实验设计和已有测量结果；
- 升级一致性不设置为独立实验，而作为 Security Analysis 中的协议属性分析；
- 论文对 WASM Runtime、委员会和底层共识协议的安全边界说明。

## 3. 建议的第四章结构

当前第四章包含：

```text
4 Security Analysis
4.1 Attack Circumstances
4.2 Security Solutions
```

建议调整为：

```text
4 Security Analysis
4.1 System Model and Security Assumptions
4.2 Adversary Model
4.3 Security Definitions
4.4 Security Theorems
4.5 Security Limitations
```

如果版面限制较大，可合并为：

```text
4.1 Security Model and Adversary Model
4.2 Security Definitions and Theorems
4.3 Security Solutions and Limitations
```

推荐采用第一种结构，因为它能清楚区分“模型”“可证明性质”和“工程防护”。

## 4. 系统模型与安全假设

### 4.1 需要统一的形式化对象

沿用第三章已有符号，并在第四章中明确其安全含义：

- $\mathcal{C}$：canonical blockchain；
- $\mathcal{M}$：Management Contract 的链上状态；
- $\mu_{a,v}$：算法 $a$ 的版本 $v$ 的版本元数据；
- $C_{a,v}$：版本 $v$ 对应的编译后 WASM 字节码；
- $h_{a,v}$：模块哈希，满足 $h_{a,v}=\mathrm{Keccak256}(C_{a,v})$；
- $H_{a,v}$：版本 $v$ 的 Activation Block 高度；
- $v^{*}(a,b)$：算法 $a$ 在区块高度 $b$ 的活动版本；
- $n_i,n_j$：遵循协议的 honest nodes；
- $\mathcal{N}_{H}$：honest node 集合。

### 4.2 建议明确的安全假设

形式化证明至少依赖以下条件：

1. **底层共识安全**：honest nodes 对 canonical chain 达成一致；
2. **升级授权安全**：只有委员会授权路径可以登记版本元数据；
3. **哈希抗碰撞性**：Keccak256 在本文讨论的安全参数下具有抗碰撞性；
4. **元数据权威性**：honest node 从 canonical Management Contract 状态恢复元数据，而不是信任本地缓存或事件内容本身；
5. **运行时确定性**：相同 WASM 字节码、相同输入和相同协议环境产生相同结果或相同协议级错误；
6. **运行时资源约束**：WASM Runtime 强制执行线性内存上限、执行预算、导入函数检查和入口函数检查；
7. **ABI 规则确定性**：ABI 编码、解码和错误处理规则在节点之间一致。

需要说明：这些是假设或系统前提，不应写成 EvoCrypt 单独解决的安全问题。

建议在正文中使用如下边界表述：

> The analysis assumes canonical-chain agreement, correct committee authorization, collision resistance of Keccak256, and deterministic WASM execution. It does not model a compromised consensus protocol, a captured committee, a compromised Geth binary, or a cryptographic algorithm whose mathematical security assumption is false.

## 5. 对手模型

保留现有 `Attack Circumstances` 的内容，但将其改写为形式化的 adversary model。

### 5.1 对手能力

外部攻击者可以：

- 构造任意合约调用参数和交易输入；
- 观察升级事件并施加调用时序压力；
- 提交格式错误的调用请求；
- 尝试利用本地缓存、事件投影或未准备状态造成版本混淆；
- 诱导节点处理大模块、慢模块或资源消耗型模块。

在委员会授权路径被攻陷或委员会错误批准的情况下，admitted proposer 还可能提交：

- 畸形 WASM 模块；
- 缺失入口函数的模块；
- 使用未授权 host imports 的模块；
- 与 ABI、Gas 参数或 Activation Block 不一致的元数据；
- 语义错误或恶意的密码学算法实现。

### 5.2 不纳入模型的能力

以下攻击不纳入本章定理：

- 破坏底层共识或制造 canonical-chain 分歧；
- 伪造 Management Contract 状态；
- 绕过委员会授权并直接写入链上升级记录；
- 攻陷所有 honest nodes 的 Geth 二进制；
- 让 Keccak256 出现实际碰撞；
- 证明或否定上传密码学算法本身的数学安全性。

## 6. 安全定义

至少加入以下四个定义；如篇幅允许，再加入资源有界性定义。

### 6.1 Module-binding security

目标：节点实际执行的模块必须与链上元数据承诺的模块一致。

建议定义：

```latex
\begin{definition}[Module-binding security]
EvoCrypt satisfies module-binding security if, for every honest node accepting
$(a,v)$ as prepared, the loaded module $C'_{a,v}$ satisfies
\[
\mathrm{Keccak256}(C'_{a,v})=h_{a,v},
\]
where $h_{a,v}$ is the module hash committed in the canonical Management
Contract state.
\end{definition}
```

### 6.2 Metadata consistency

目标：算法名称、版本号、模块哈希、ABI 描述、Gas 参数和激活高度必须属于同一个权威元数据记录。

建议定义：

```latex
\begin{definition}[Metadata consistency]
A version record $\mu_{a,v}$ is metadata-consistent if its algorithm name,
version number, module hash, ABI descriptors, gas parameters, and activation
height are identical in the Management Contract state, the emitted event, and
the locally prepared record.
\end{definition}
```

需要强调：event 只作为通知和索引，Management Contract state 才是 authoritative source。

### 6.3 Activation consistency

目标：观察到同一条 canonical chain 的 honest nodes 在同一执行高度选择同一算法版本。

建议定义：

```latex
\begin{definition}[Activation consistency]
EvoCrypt satisfies activation consistency if any two honest nodes observing the
same canonical chain and executing a call at the same block height $b$ select
the same active version:
\[
v_i^{*}(a,b)=v_j^{*}(a,b).
\]
\end{definition}
```

### 6.4 Invocation determinism

目标：相同版本和相同输入必须产生相同的 ABI 编码结果或相同的协议级错误。

建议定义：

```latex
\begin{definition}[Invocation determinism]
For an admissible module version $(a,v)$, invocation determinism requires that
for every valid input $x$, all honest nodes produce the same encoded result or
the same protocol-level error:
\[
\mathrm{Execute}_i(a,v,x)=\mathrm{Execute}_j(a,v,x).
\]
\end{definition}
```

### 6.5 Resource-bounded execution（可选）

如果篇幅允许，加入：

```latex
\begin{definition}[Resource-bounded execution]
A module execution is resource-bounded if it consumes at most $M_{\max}$ units
of linear memory and $T_{\max}$ units of execution budget, as enforced by the
WASM Runtime.
\end{definition}
```

该定义只能支撑 runtime-level 的资源上界结论，不能直接推出整个网络的拒绝服务安全。

## 7. 安全定理与证明计划

### 7.1 Theorem 1：Module-binding security

建议表述：

```latex
\begin{theorem}[Module-binding security]
Assume that Keccak256 is collision resistant and that an honest node obtains
version metadata from the canonical Management Contract state. Then an
adversary cannot cause the node to prepare a module different from the module
committed by $\mu_{a,v}$, except with negligible probability.
\end{theorem}
```

证明要点：

1. honest node 从链上权威状态恢复 $C_{a,v}$ 和 $h_{a,v}$；
2. 节点重新计算恢复字节码的 Keccak256；
3. 只有哈希匹配时才将版本标记为 prepared；
4. 若不同字节码 $C'_{a,v}\neq C_{a,v}$ 仍然通过检查，则攻击者必须构造：

   \[
   \mathrm{Keccak256}(C'_{a,v})=
   \mathrm{Keccak256}(C_{a,v}),
   \]

   这与 Keccak256 的抗碰撞性矛盾。

证明边界：该定理证明的是模块身份绑定，不证明模块的密码学功能正确，也不防止委员会合法批准一个恶意模块。

### 7.2 Proposition 1：Metadata consistency

建议表述：

```latex
\begin{proposition}[Metadata consistency]
If an honest node accepts a version only after checking the event projection
against the authoritative Management Contract state, then an inconsistent
event cannot cause the node to prepare a different algorithm version, ABI
description, activation height, or gas schedule.
\end{proposition}
```

证明要点：

- event 与 Management Contract state 不一致时拒绝；
- 节点不把 event 作为独立安全根；
- ABI、Gas 和 Activation Block 都从同一条 $\mu_{a,v}$ 中恢复；
- 因此本地准备记录只能绑定到经校验的权威元数据。

### 7.3 Theorem 2：Deterministic activation

建议表述：

```latex
\begin{theorem}[Deterministic activation]
Assume that two honest nodes observe the same canonical chain and use the same
version-selection rule in~\eqref{eq:version-select}. Then, for any algorithm
$a$ and execution height $b$, both nodes select the same active version.
\end{theorem}
```

证明要点：

定义两个节点在高度 $b$ 上的候选集合：

\[
S_i=\{(H_{a,v},v)\mid H_{a,v}\leq b\},
\]

\[
S_j=\{(H_{a,v},v)\mid H_{a,v}\leq b\}.
\]

由于两个节点观察到同一条 canonical chain，因此 $S_i=S_j$。二者使用相同的 lexicographic $\arg\max$ 规则，所以：

\[
v_i^{*}(a,b)=v_j^{*}(a,b).
\]

需要明确说明：本定理不依赖节点何时完成本地编译和实例化；它依赖的是 Activation Block 选择规则，以及未准备完成时强制恢复和准备目标版本的策略。

### 7.4 Theorem 3：Invocation consistency

建议表述：

```latex
\begin{theorem}[Invocation consistency]
Suppose that (i) honest nodes select the same active version, (ii) the selected
WASM module is byte-identical, (iii) ABI encoding and decoding are deterministic,
and (iv) the WASM Runtime exposes only deterministic host functions. Then two
honest nodes executing the same request at the same block height produce the
same encoded result or the same protocol-level error.
\end{theorem}
```

证明要点：

1. 由 Theorem 2，两个节点选择同一 $v^{*}(a,b)$；
2. 由 Theorem 1，两个节点加载同一 $C_{a,v}$；
3. 输入 $x$、ABI 描述和 Gas 参数来自同一链上状态；
4. 运行时不暴露时间、随机数、文件系统、网络或未声明 host imports；
5. 因此执行轨迹、返回值和错误类型一致。

该定理是第四章最重要的组合结论，说明动态加载机制不会因为本地实现替换而自动破坏区块链执行确定性。

### 7.5 Theorem 4：Bounded module execution（可选）

建议表述：

```latex
\begin{theorem}[Bounded module execution]
If the WASM Runtime enforces a memory limit and an execution budget, then a
single invocation cannot consume more than the configured memory and execution
budgets, except for costs outside the runtime boundary such as artifact
retrieval, compilation, storage, and network propagation.
\end{theorem}
```

证明要点：

- Runtime 在模块执行过程中强制检查线性内存上限；
- 超出执行预算时终止执行并返回 runtime error；
- 因此单次模块调用的运行时资源消耗具有显式上界。

限制：该定理不等价于完整的网络级 DoS 安全证明，也不覆盖大量升级交易、超大模块传播、编译阶段 CPU 消耗或委员会恶意提交造成的系统级压力。

## 8. 安全解决方案与工程机制的重新组织

现有 `Security Solutions` 内容不删除，而是放在定理之后，作为“实现如何满足假设和证明前提”的工程支撑：

### 8.1 Committee-gated upgrade admission

说明其作用是阻止未授权外部账户登记模块和元数据，而不是证明委员会本身诚实。应将委员会被捕获列为 trust assumption 或 limitation。

### 8.2 Sandboxed WASM execution

说明 linear memory、import validation、entry-point validation 和 runtime budget 如何支撑执行边界和资源有界性。不要宣称 WASM sandbox 已经提供形式化完备的隔离证明。

### 8.3 Authoritative metadata recovery

说明 Management Contract state 是权威源，event 只用于 notification/index；本地重新计算模块哈希并将其绑定到版本记录。

### 8.4 Deterministic resource accounting

保留公式 $G(a,v,x)=g^{(0)}_{a,v}+\lambda_{a,v}\cdot |x|$，说明相同链上状态、版本和输入会得到相同 Gas 费用。明确当前 Gas 系数是实验参数，不把它表述为已经完成的经济最优或完整 DoS 防护。

### 8.5 Forced activation

说明 Activation Block 到达时不允许因为本地尚未准备完成而静默回退到旧版本；本地模块未准备完成时，节点不得回退旧版本，也不得将本地准备失败转换为影响共识状态的调用错误；应暂停相关区块执行，完成恢复与准备后再继续。

## 9. 安全边界与不能声称的结论

第四章结尾增加 `Security Limitations`，明确以下边界：

1. **不证明委员会治理安全**：若委员会被攻陷，可以合法批准恶意或错误模块；
2. **不证明密码学算法本身安全**：EvoCrypt 证明的是模块加载、版本选择和调用路径属性，不证明上传算法满足安全性定义；
3. **不证明 Geth 二进制安全**：不考虑宿主客户端已被攻陷的情形；
4. **不证明底层共识安全**：定理以 canonical-chain agreement 为前提；
5. **不证明完整网络级 DoS 安全**：资源定理只覆盖单次 WASM 执行边界；
6. **不保证任意故障下的可用性**：若模块始终无法编译、实例化或加载，协议可以阻止错误版本执行，但不保证调用成功；
7. **不将安全分析实验化**：当前论文没有单独的 state-root、一致性压力或 fail-stop 实验，因此这些内容只能作为协议分析和 future work。

建议使用以下总结句：

> The formal results establish protocol-level binding, activation consistency, and deterministic invocation under the stated assumptions. They do not constitute a complete proof of cryptographic algorithm security, committee governance security, or network-wide denial-of-service resistance.

## 10. LaTeX 实施步骤

### 阶段一：准备与术语统一

- 检查第四章使用的符号是否与第三章的 `$\mu_{a,v}$`、`$h_{a,v}$`、`$H_{a,v}$`、`$v^{*}(a,b)$` 一致；
- 不新增与 `terminology.md` 冲突的英文术语；
- 保留已有 `Active version` 和 `Algorithm-level gas schedule` 定义，避免重复定义；
- 确认 `llncs` 类已提供 `definition`、`theorem`、`lemma`、`proposition` 和 `proof` 环境。

### 阶段二：重组第四章

- 将现有 `Attack Circumstances` 改写为 `Adversary Model`；
- 在其前加入 `System Model and Security Assumptions`；
- 加入四个核心安全定义；
- 将现有安全解决方案段落移动到定理之后，作为实现支撑；
- 增加 `Security Limitations`；
- 将“未进行独立一致性实验”的说明保留在安全边界中，并确保与 Section 5 的表述一致。

### 阶段三：证明与语言检查

- 每个定理必须紧跟一个简短的 `proof`；
- 证明只使用论文已经声明的机制和假设，不引入未实现的签名、证明系统或额外共识机制；
- 使用 `except with negligible probability` 时，明确其来源是 Keccak 抗碰撞性或明确的密码学假设；
- 对确定性定理使用“same canonical chain + same rule + deterministic runtime”的条件表达，避免无条件安全声称；
- 将工程性表述和形式化结论分开，避免把“风险降低”写成“攻击必然被阻止”。

### 阶段四：编译和一致性检查

完成修改后执行：

```bash
cd 'paper/Springer LNCS'
pdflatex -interaction=nonstopmode evocrypt.tex
bibtex evocrypt
pdflatex -interaction=nonstopmode evocrypt.tex
pdflatex -interaction=nonstopmode evocrypt.tex
```

检查项目：

- theorem、definition、proposition 和 proof 环境是否正常编译；
- 公式编号、交叉引用和 `\eqref{eq:version-select}` 是否正确；
- 是否出现 overfull/underfull box 或定理环境导致的版面溢出；
- 第四章与第三章符号是否一致；
- 第四章与 Section 5 是否一致地说明没有独立升级一致性实验；
- 是否误将委员会、WASM sandbox 或算法本身描述为已经获得完全安全保证；
- 是否保留所有原有实验数据和结论边界。

## 11. 推荐的最小交付版本

如果需要控制篇幅，第一版只实现以下内容：

1. `System Model and Security Assumptions`；
2. `Adversary Model`；
3. `Module-binding security` 定义与定理；
4. `Activation consistency` 定义与定理；
5. `Invocation determinism` 定义与定理；
6. `Security Limitations`；
7. 保留现有元数据校验、WASM sandbox、Gas 确定性和 forced activation 作为实现支撑。

该最小版本已经能够回应“安全分析过于朴素”的主要意见，并且与 EvoCrypt 当前实现和实验边界相容。

## 12. 不建议在本轮加入的内容

为避免无依据扩展，本轮不建议加入：

- 未实现的数字签名、零知识证明或远程证明机制；
- 对委员会诚实性的无条件假设之外的治理协议设计；
- 没有实验数据支持的节点一致性测试结果；
- 完整的形式化 WASM 语义证明；
- “对所有恶意密码学算法都安全”的表述；
- 将 Gas 参数直接称为经过实证校准的经济模型；
- 将 runtime memory limit 直接等同于完整的网络级 DoS 防御。

