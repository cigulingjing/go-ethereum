# 第四章 安全性分析

本节对升级协议进行形式化，并在此基础上推导调用过程的安全保证。第 4.1 节给出了账本与注册表抽象、可信假设、攻击者模型，以及针对升级机制必须排除的两类失败模式的基于博弈的定义：其一是诚实节点在同一区块高度选择了不同版本；其二是诚实节点执行了不同于已准入 artifact 的字节码。第 4.2 节证明，EvoCrypt 在账本抽象下无条件地排除了第一类失败，并通过到 $\mathrm{Keccak256}$ 抗碰撞性的紧归约排除了第二类失败，进而将二者合并为一个升级可靠性定理。第 4.3 节将一致性扩展到调用结果；第 4.4 节则讨论模型之外仍然存在的剩余风险，以及用于应对这些风险的治理恢复路径。

## 4.1 安全模型

**符号与账本。** $\kappa$ 表示安全参数，$\mathsf{negl}(\kappa)$ 表示可忽略函数；除非另有说明，本文中的算法均为概率多项式时间算法（PPT）。$\mathcal{N}$ 表示诚实节点集合，其中每个节点都运行算法~\ref{alg:upgrade-protocol} 和算法~\ref{alg:invocation-protocol}。符号 $\mu_{a,v}$、$C_{a,v}$、$h_{a,v}$ 和 $H_{a,v}$ 的含义与定义~\ref{def:version-metadata} 中相同，$u_{a,v}$ 表示准入 $\mu_{a,v}$ 的 Upgrade Block 高度。我们将区块链抽象为一个交易账本：诚实节点 $n$ 持有一个本地视图 $\mathcal{L}_n$，它是一个有限的区块序列，其前缀表示为 $\mathcal{L}_n[1..b]$，高度为 $|\mathcal{L}_n|$。区块内部的交易顺序也是账本的一部分，因此一个执行点由区块高度和该区块内的位置共同确定；为简化表示，本文用 $b$ 同时表示二者。

**定义（注册表状态）。** 账本前缀 $P$ 的注册表 $\mathsf{Reg}(P)$，是从创世状态开始执行 $P$ 后得到的 Management Contract 存储状态；由于 EVM 的确定性，它是 $P$ 的函数。对于节点 $n$ 和高度 $b$，令

$$
\mathcal{V}^{n}_{a}(b)=\{v \mid \mu_{a,v}\in\mathsf{Reg}(\mathcal{L}_n[1..b])\}
$$

并令 $v^{*}_{n}(a,b)$ 表示在 $\mathcal{V}^{n}_{a}(b)$ 上按照式~\eqref{eq:version-select} 计算得到的值；当可行集合为空时，令 $v^{*}_{n}(a,b)=\bot$。

**可信假设。** 假设~\ref{asm:ledger}--\ref{asm:cr} 将在第 4.2 节中使用；假设~\ref{asm:runtime} 仅在第 4.3 节中使用。

**假设（规范账本）。** 对于所有诚实节点 $n,n'\in\mathcal{N}$，以及任意满足 $b\le\min(|\mathcal{L}_n|,|\mathcal{L}_{n'}|)$ 的 $b$，都有：

$$
\mathcal{L}_n[1..b]=\mathcal{L}_{n'}[1..b].
$$

这是健壮交易账本的持久性属性，应用于调用被评估时所处的账本前缀；仍可能发生重组的区块会按照普通 EVM 规则重新执行，因此不属于本模型讨论范围。

**假设（委员会门控的准入）。** Management Contract 只有在提交获得委员会授权时才准入 $\mu_{a,v}$，并且每条已准入记录都是格式良好的：(i) $h_{a,v}=\mathrm{Keccak256}(C_{a,v})$；(ii) $u_{a,v}<H_{a,v}$，即该记录必须严格早于其 Activation Block 被提交；(iii) $(a,v)$ 是唯一的，并且已准入记录不会被修改或删除；(iv) $C_{a,v}$ 通过配置的 WASM Runtime 验证。此外，委员会还被信任会在准入前审查实现及其 gas schedule。条件 (i)--(iii) 可以由合约本身强制执行；将其列为假设，是为了使安全分析不依赖具体的合约实现。

**假设（抗碰撞性）。** 对于任意 PPT 算法 $\mathcal{B}$，有：

$$
\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B})
=\Pr\bigl[(C,C')\leftarrow\mathcal{B}(1^{\kappa}) :
C\neq C' \wedge \mathrm{Keccak256}(C)=\mathrm{Keccak256}(C')\bigr]
\le\mathsf{negl}(\kappa).
$$

$\mathrm{Keccak256}$ 是无密钥哈希函数，因此式~\eqref{eq:cr-adv} 按照 Rogaway 的具体安全意义来理解：下文中的每个归约都会显式给出一个碰撞寻找器 $\mathcal{B}$，并用 $\mathcal{B}$ 的优势来界定协议攻击者的优势。

**假设（确定性运行时）。** WASM Runtime 暴露一个验证谓词 $\mathsf{Validate}(C)\in\{0,1\}$ 和一个执行函数 $\mathsf{Exec}(C,x)$。其输出，包括错误结果在内，只依赖于 $(C,x)$ 以及固定的、白名单化的宿主接口。它会强制执行有界线性内存、导入验证和配置的资源限制，并且不会暴露该接口之外的任何宿主状态或能力。

**攻击者。** PPT 攻击者 $\mathcal{A}$ 可以：(a) 提交任意交易，包括调用输入和候选升级提案；其中候选升级提案只有在满足假设~\ref{asm:committee} 时才会被准入；(b) 调度升级事件的传递，延迟或中断节点本地的准备过程，并使节点崩溃和重启，从而触发准备过程重试；以及 (c) 篡改节点本地的 artifact 路径，即替换节点在解码、持久化或重新加载 artifact 时获得的字节序列 $\widehat{C}$，以建模被破坏的缓存、持久化副本以及不可信的检索通道。攻击者不能违反假设~\ref{asm:ledger}--\ref{asm:cr}，不能伪造委员会授权，也不能修改客户端二进制文件。因此，该模型排除了以下情况：被攻陷的 Geth 二进制文件、失效的共识协议、被控制的委员会、失败的实现审查，以及底层假设本身不成立的密码学原语。

**安全性定义。** 下面的每个实验都会让 $\mathcal{A}$ 在该模型中与诚实节点交互；当 $\mathcal{A}$ 造成不希望发生的事件时，实验输出 $1$。$\mathcal{A}$ 的优势定义为实验在所有随机性上的输出 $1$ 的概率。

**定义（版本分歧）。** 在 $\mathsf{Exp}^{\mathrm{div}}_{\mathcal{A}}(\kappa)$ 中，攻击者最终输出一个元组 $(a,b,n,n')$，其中 $n,n'\in\mathcal{N}$，且 $b\le\min(|\mathcal{L}_n|,|\mathcal{L}_{n'}|)$。当且仅当 $v^{*}_{n}(a,b)\neq v^{*}_{n'}(a,b)$ 时，实验输出 $1$；其中 $\bot$ 与任意版本都不同。定义：

$$
\mathsf{Adv}^{\mathrm{div}}_{\mathcal{A}}(\kappa)=\Pr[\mathsf{Exp}^{\mathrm{div}}_{\mathcal{A}}(\kappa)=1].
$$

**定义（模块替换）。** 当且仅当存在某个诚实节点 $n$，它将 $(a,v)$ 标记为已准备，或针对 $(a,v)$ 执行了某个字节序列 $\widehat{C}$，并且满足 $\mu_{a,v}\in\mathsf{Reg}(\mathcal{L}_n)$ 且 $\widehat{C}\neq C_{a,v}$ 时，实验 $\mathsf{Exp}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)$ 输出 $1$。定义：

$$
\mathsf{Adv}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)=\Pr[\mathsf{Exp}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)=1].
$$

**定义（升级可靠性）。** 当且仅当存在某个诚实节点在高度 $b$ 调用算法 $a$ 时，执行了一个字节序列 $\widehat{C}\neq C_{a,v^{*}(a,b)}$，实验 $\mathsf{Exp}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)$ 输出 $1$；其中 $v^{*}(a,b)$ 是在高度为 $b$ 的规范链前缀上计算得到的。如果在 $v^{*}(a,b)=\bot$ 的情况下仍然执行了任何模块，也同样计为输出 $1$。如果对于任意 PPT 攻击者 $\mathcal{A}$，都有

$$
\mathsf{Adv}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)=\Pr[\mathsf{Exp}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)=1]
$$

是可忽略的，则称 EvoCrypt 满足升级可靠性。

定义~\ref{def:sound} 是本文的目标性质：无论 $\mathcal{A}$ 如何操纵调度和节点本地存储，每个响应调用请求的诚实节点，都必须恰好运行规范链在该高度指定的 artifact。定义~\ref{def:div} 和定义~\ref{def:sub} 分别刻画了该性质可能失败的两种方式。

## 4.2 升级安全性

**引理（激活及时性）。** 在假设~\ref{asm:committee} 下，对于每个诚实节点 $n$ 和每个 $\mu_{a,v}\in\mathsf{Reg}(\mathcal{L}_n)$：(i) 对任意 $b<H_{a,v}$，都有 $v^{*}_{n}(a,b)\neq v$；并且 (ii) 对每个满足 $H_{a,v}\le b\le|\mathcal{L}_n|$ 的 $b$，$v$ 都位于式~\eqref{eq:version-select} 的可行集合中。

**证明。** (i) 可行集合只包含满足 $H_{a,v}\le b$ 的版本。(ii) 由假设~\ref{asm:committee}(ii) 可知，$u_{a,v}<H_{a,v}\le b$，因此准入交易位于 $\mathcal{L}_n[1..b]$ 中；由 (iii) 可知，该记录不会被移除。因此 $\mu_{a,v}\in\mathsf{Reg}(\mathcal{L}_n[1..b])$，且 $H_{a,v}\le b$ 使得 $v$ 位于可行集合中。证毕。

该引理将 $(a,v)$ 的准备窗口限制在 $u_{a,v}<b<H_{a,v}$ 的高度范围内，并排除了追溯激活：追加新区块永远不会改变已经执行过的高度所指定的版本。这也是为什么像第 5.3 节中的后加入节点在重放历史时，会与实时执行该历史的节点保持一致。

**定理（激活一致性）。** 在假设~\ref{asm:ledger} 和假设~\ref{asm:committee} 下，对于每个攻击者 $\mathcal{A}$，即使它计算能力无界，也有 $\mathsf{Adv}^{\mathrm{div}}_{\mathcal{A}}(\kappa)=0$。

**证明。** 固定 $\mathcal{A}$ 的任意输出 $(a,b,n,n')$。由假设~\ref{asm:ledger} 可知，$P=\mathcal{L}_n[1..b]=\mathcal{L}_{n'}[1..b]$；由定义~\ref{def:registry} 可知，$\mathsf{Reg}(P)$ 是 $P$ 的函数，因此两个节点持有相同的记录和相同的可行集合 $\{(H_{a,v},v): H_{a,v}\le b\}$。由假设~\ref{asm:committee}(iii) 可知，其元素两两不同，因此当集合非空时，式~\eqref{eq:version-select} 中的字典序最大值是唯一的；当集合为空时，两个节点都返回 $\bot$。因此 $v^{*}_{n}(a,b)=v^{*}_{n'}(a,b)$。没有任何攻击者能力会参与该计算：调度、准备时机、崩溃和 artifact 篡改作用于节点本地状态，而算法~\ref{alg:invocation-protocol} 第~\ref{line:resolve} 行会在查询任何节点本地状态之前，先基于注册表评估式~\eqref{eq:version-select}。因此，无论 $\mathcal{A}$ 的运行时间如何，该实验都不会输出 $1$。证毕。

定理~\ref{thm:activation} 是信息论意义上的。下一个定理表明，将已执行字节码绑定到注册表，只需要付出一个抗碰撞性假设的代价。

**定理（模块绑定）。** 在假设~\ref{asm:committee} 下，对于每个 PPT 攻击者 $\mathcal{A}$，都存在一个 PPT 算法 $\mathcal{B}$，其运行时间为 $\mathcal{A}$ 的运行时间加上模拟诚实节点的开销，并且满足：

$$
\mathsf{Adv}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)\le\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B}).
$$

特别地，在假设~\ref{asm:cr} 下，$\mathsf{Adv}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)$ 是可忽略的。

**证明。** 在输入 $1^{\kappa}$ 时，$\mathcal{B}$ 在内部运行 $\mathsf{Exp}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)$：它模拟账本和每个诚实节点，代表它们执行算法~\ref{alg:upgrade-protocol} 和算法~\ref{alg:invocation-protocol}，并且像真实实验一样响应 $\mathcal{A}$ 的调度请求和 artifact 篡改请求。由于模型中没有秘密状态，该模拟是完美的。每当某个被模拟的诚实节点即将用 $\widehat{C}$ 将 $(a,v)$ 标记为已准备（算法~\ref{alg:upgrade-protocol} 第~\ref{line:mark-prepared} 行），或即将针对 $(a,v)$ 执行 $\widehat{C}$（算法~\ref{alg:invocation-protocol} 第~\ref{line:execute} 行）时，$\mathcal{B}$ 从被模拟的注册表中读取 $C_{a,v}$；如果 $\widehat{C}\neq C_{a,v}$，它就输出 $(\widehat{C},C_{a,v})$ 并停止；否则最终输出 $\bot$。

令 $E$ 为 $\mathsf{Exp}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)$ 输出 $1$ 的事件；由于模拟是完美的，在 $\mathcal{B}$ 的运行中有 $\Pr[E]=\mathsf{Adv}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)$。在事件 $E$ 上，某个诚实节点接受了用于 $(a,v)$ 的 $\widehat{C}$；这只有在它根据规范记录验证 $\mathrm{Keccak256}(\widehat{C})=h_{a,v}$ 后才会发生（算法~\ref{alg:upgrade-protocol} 第~\ref{line:hash-check} 行；已准备集合以 $h_{a,v}$ 为键，因此执行时会在同一检查下重新加载 artifact）。由假设~\ref{asm:committee}(i) 可知，$h_{a,v}=\mathrm{Keccak256}(C_{a,v})$，因此 $(\widehat{C},C_{a,v})$ 构成一个碰撞，并且 $\Pr[E]\le\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B})$。该归约是紧的：它没有损失任何优势，也不需要猜测目标记录。证毕。

哈希只绑定字节码；ABI 描述符、gas 参数和 Activation Block 都直接从规范注册表读取，而定理~\ref{thm:activation} 已经保证诚实节点之间的规范注册表是一致的。

**定理（升级可靠性）。** 在假设~\ref{asm:ledger}--\ref{asm:cr} 下，EvoCrypt 满足升级可靠性：对于每个 PPT 攻击者 $\mathcal{A}$，都存在一个如定理~\ref{thm:binding} 中所述的 PPT 算法 $\mathcal{B}$，使得：

$$
\mathsf{Adv}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)\le\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B}).
$$

**证明。** 令 $W$ 表示 $\mathsf{Exp}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)$ 输出 $1$ 的事件：某个诚实节点 $n$ 在高度 $b$ 调用算法 $a$，并使用 $\widehat{C}$ 执行。令 $v=v^{*}(a,b)$ 为规范选择结果，$v_n$ 为节点 $n$ 在算法~\ref{alg:invocation-protocol} 第~\ref{line:resolve} 行解析得到的版本，$\hat v$ 为 $\widehat{C}$ 被准备时对应的版本。则 $W\subseteq E_1\cup E_2\cup E_3$，其中

$$
E_1 : v_n\neq v,\qquad
E_2 : v_n=v \wedge \hat v\neq v_n,\qquad
E_3 : \hat v=v_n=v \wedge \widehat{C}\neq C_{a,v}.
$$

因为如果上述事件都不发生，则 $\widehat{C}=C_{a,v^{*}(a,b)}$。$\Pr[E_1]=0$：节点 $n$ 在 $\mathsf{Reg}(\mathcal{L}_n[1..b])$ 上评估式~\eqref{eq:version-select}，而由假设~\ref{asm:ledger} 可知该前缀是规范前缀，因此定理~\ref{thm:activation} 的论证给出 $v_n=v$；如果 $v=\bot$，节点会返回缺失版本错误（第~\ref{line:missing} 行），并且不会执行任何内容。$\Pr[E_2]=0$：第~\ref{line:ensure} 行只允许为已解析出的二元组 $(a,v_n)$ 执行；在强制准备策略下，它会继续准备 $(a,v_n)$，而不是回退到另一个已准备版本，因此能够到达第~\ref{line:execute} 行的唯一 artifact 必然是为 $(a,v_n)$ 准备的。$E_3$ 是针对 $(a,v)$ 的模块替换，并且 $\mu_{a,v}\in\mathsf{Reg}(\mathcal{L}_n)$；后者由引理~\ref{lem:timeliness}(ii) 得到，因为 $H_{a,v}\le b$。因此，由定理~\ref{thm:binding} 可知，$\Pr[E_3]\le\mathsf{Adv}^{\mathrm{sub}}_{\mathcal{A}}(\kappa)\le\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B})$。由并合界可得上述结论。证毕。

**注（安全性与活性）。** 定理~\ref{thm:activation}--\ref{thm:soundness} 是安全性陈述：诚实节点永远不会选择或运行规范版本之外的版本。它们并不声称每个节点都能在高度 $H_{a,v}$ 给出响应；仍在准备中的节点，或处于算法~\ref{alg:upgrade-protocol} 重试循环中的节点，会选择延迟，而不是降级执行。在假设~\ref{asm:committee}(iv) 以及假设~\ref{asm:runtime} 的确定性验证下，一个已准入 artifact 会在每个诚实节点上得到相同的验证结果，因此不会出现某个节点准备成功而另一个节点永久失败的情况；其完成速度由第 5.2 节和第 5.3 节进行测量，而不是在此处证明。

## 4.3 调用安全性

第 4.2 节确定了诚实节点执行的是“哪个” artifact。本小节表明，一旦加入假设~\ref{asm:runtime}，artifact 上的一致性就可以提升为调用结果以及由此产生的 EVM 状态转换上的一致性。假设~\ref{asm:runtime} 是唯一新增的要素；它是关于已配置 WASM Runtime 的工程假设，而不是密码学假设，并且下面的陈述会明确体现它的作用。算法~\ref{alg:invocation-protocol} 的一个“结果”是二元组 $(o,\gamma)$，其中 $o$ 要么是 ABI 编码后的结果 $y$，要么是该算法返回的某个协议错误；$\gamma$ 是收取的 gas，当算法在第~\ref{line:execute} 行之前返回时，$\gamma=0$。

**定义（调用分歧）。** 在 $\mathsf{Exp}^{\mathrm{inv}}_{\mathcal{A}}(\kappa)$ 中，攻击者最终输出 $(d,g,b,n,n')$，其中 $n,n'\in\mathcal{N}$，且 $b\le\min(|\mathcal{L}_n|,|\mathcal{L}_{n'}|)$；两个节点都在高度 $b$、以 gas 预算 $g$，针对请求 $d$ 运行算法~\ref{alg:invocation-protocol}。如果任一节点尚未产生结果，则实验输出 $0$；否则，当且仅当 $(o_n,\gamma_n)\neq(o_{n'},\gamma_{n'})$ 时，实验输出 $1$。定义：

$$
\mathsf{Adv}^{\mathrm{inv}}_{\mathcal{A}}(\kappa)=\Pr[\mathsf{Exp}^{\mathrm{inv}}_{\mathcal{A}}(\kappa)=1].
$$

当某个节点尚未响应时返回 $0$，使该定义保持为一个安全性性质，这与注~\ref{rem:liveness} 一致：在第~\ref{line:ensure} 行仍处于准备状态的节点会延迟响应，而不会被计为发生分歧。

**引理（执行前一致性）。** 在假设~\ref{asm:ledger} 和假设~\ref{asm:committee} 下，两个诚实节点在相同的 $(d,g,b)$ 上运行算法~\ref{alg:invocation-protocol} 时，在第~\ref{line:execute} 行之前的每一行都会走相同的分支，解析出相同的版本 $v=v^{*}(a,b)$，并计算出相同的 $G_{\mathrm{req}}=G(a,v,x)$。特别地，如果任一节点返回 ABI 错误、缺失版本错误或 gas 不足错误，则两个节点都会返回相同错误，且 $\gamma=0$。

**证明。** $\mathrm{Parse}$ 是 $d$ 的固定函数，因此两个节点会得到相同的 $(a,x)$ 或相同的 ABI 错误。定理~\ref{thm:activation} 给出 $v^{*}_{n}(a,b)=v^{*}_{n'}(a,b)$，因此缺失版本测试是一致的。如果解析出了某个版本，则两个节点都从 $\mathsf{Reg}(\mathcal{L}_n[1..b])=\mathsf{Reg}(\mathcal{L}_{n'}[1..b])$ 中读取 $\mu_{a,v}$（由假设~\ref{asm:ledger} 和定义~\ref{def:registry} 得到），因此读到相同的 $g^{(0)}_{a,v}$、$\lambda_{a,v}$、$\tau^{\mathrm{in}}_{a,v}$ 和 $\tau^{\mathrm{out}}_{a,v}$。由于 $|x|$ 由 $d$ 决定，式~\eqref{eq:gas-price} 会给出相同的 $G_{\mathrm{req}}$；再与共同预算 $g$ 比较，就会得到相同的 gas 不足判断。这些步骤中，除了已准备集合之外，没有任何步骤会读取节点本地状态；而已准备集合只决定第~\ref{line:ensure} 行是否必须先执行准备，并不决定走哪个分支。证毕。

**定理（调用一致性）。** 在假设~\ref{asm:ledger}--\ref{asm:runtime} 下，对于每个 PPT 攻击者 $\mathcal{A}$，都存在一个如定理~\ref{thm:binding} 中所述的 PPT 算法 $\mathcal{B}$，使得：

$$
\mathsf{Adv}^{\mathrm{inv}}_{\mathcal{A}}(\kappa)\le\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B}).
$$

此外，只要两个节点都给出响应，它们收取的 gas 就无条件相同：如果到达第~\ref{line:execute} 行，则 $\gamma_n=\gamma_{n'}=G(a,v^{*}(a,b),x)$；否则 $\gamma_n=\gamma_{n'}=0$。

**证明。** 令 $W$ 表示 $\mathsf{Exp}^{\mathrm{inv}}_{\mathcal{A}}(\kappa)$ 输出 $1$ 的事件，因此两个节点都给出了响应，且 $(o_n,\gamma_n)\neq(o_{n'},\gamma_{n'})$。令 $v=v^{*}(a,b)$，并令 $\widehat{C}_n,\widehat{C}_{n'}$ 表示两个节点在到达第~\ref{line:execute} 行时执行的字节序列。则 $W\subseteq F_1\cup F_2\cup F_3$，其中

$$
\begin{aligned}
F_1 &: \text{两个节点在第~\ref{line:execute} 行之前走了不同分支},\\
F_2 &: \text{两个节点都到达第~\ref{line:execute} 行，且 } \widehat{C}_n\neq C_{a,v}\ \vee\ \widehat{C}_{n'}\neq C_{a,v},\\
F_3 &: \widehat{C}_n=\widehat{C}_{n'}=C_{a,v}\ \wedge\ o_n\neq o_{n'}.
\end{aligned}
$$

因为如果这些事件均未发生，则两个节点要么返回相同的执行前错误且 $\gamma=0$，要么以相同的 $G_{\mathrm{req}}$ 到达第~\ref{line:execute} 行，执行相同字节码并返回相同的 $o$。由引理~\ref{lem:preexec} 可知，$\Pr[F_1]=0$。$F_2$ 表示某个诚实节点在高度 $b$ 调用算法 $a$ 时，执行了不同于 $C_{a,v^{*}(a,b)}$ 的字节码，这正是 $\mathsf{Exp}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)$ 的获胜事件；因此，由定理~\ref{thm:soundness} 可知，$\Pr[F_2]\le\mathsf{Adv}^{\mathrm{bind}}_{\mathcal{A}}(\kappa)\le\mathsf{Adv}^{\mathrm{cr}}_{\mathrm{Keccak256}}(\mathcal{B})$。该界限同时覆盖两个节点，因为 $\mathsf{Exp}^{\mathrm{bind}}$ 已经对所有诚实节点进行了量化。$\Pr[F_3]=0$：在 $F_3$ 上，两个节点以相同参数计算 $\mathsf{Exec}(C_{a,v},x)$；由假设~\ref{asm:runtime} 可知，其输出，包括任何运行时错误，都是 $(C_{a,v},x)$ 和固定宿主接口的函数；在共同的 $\tau^{\mathrm{out}}_{a,v}$ 下，对该输出进行 ABI 编码是确定性的，因此 $o_n=o_{n'}$。由并合界可得上述优势界限。关于 gas 的陈述来自引理~\ref{lem:preexec}：当到达第~\ref{line:execute} 行时，$\gamma$ 等于 $G_{\mathrm{req}}$；否则等于 $0$，且这两个量的一致性不依赖任何计算性假设。证毕。

该界限与定理~\ref{thm:soundness} 的界限一致：加入假设~\ref{asm:runtime} 不会引入额外的概率损失，并且收取的 gas 是无条件一致的。后者是定义~\ref{def:gas-schedule} 的形式化对应物：由于 $G$ 只依赖注册表字段和 $|x|$，没有任何节点本地测量进入费用计算；CPU 时间差异或执行的 WASM 指令数量差异，都不会导致诚实节点收取不同 gas。

## 4.4 剩余风险与治理恢复

在定理~\ref{thm:activation}--\ref{thm:invocation} 之后，第 4.1 节模型内仍然存在两类风险：已准入的实现之后可能被证明不合适或存在漏洞；已准入的 gas schedule 可能对一次调用消耗的资源定价过低或过高。二者都不能被假设~\ref{asm:committee} 中的委员会审查排除，因为委员会虽然被信任，但并非不会出错。EvoCrypt 并不阻止这类准入；它保证的是：一旦某个治理式修正被准入，该修正就会在每个诚实节点上生效，并使故障版本不可达。

**推论（治理式替换）。** 在假设~\ref{asm:ledger} 和假设~\ref{asm:committee} 下，设 $\mu_{a,v}$ 和 $\mu_{a,v'}$ 均已准入，且 $H_{a,v'}>H_{a,v}$。那么对于每个诚实节点 $n$ 和每个高度 $H_{a,v'}\le b\le|\mathcal{L}_n|$，都有 $v^{*}_{n}(a,b)\neq v$。进一步地，如果除 $v'$ 之外，算法 $a$ 的其他已准入版本都没有激活高度位于 $[H_{a,v'},b]$ 内，那么 $v^{*}_{n}(a,b)=v'$。

推论~\ref{cor:supersession} 覆盖了三种修正动作。**替换（Replacement）** 准入带有新字节码的 $\mu_{a,v'}$；从 $H_{a,v'}$ 开始，定理~\ref{thm:soundness} 保证每个诚实节点都会运行 $C_{a,v'}$，而不再运行 $C_{a,v}$。**重新定价（Repricing）** 准入带有修订后 $g^{(0)}_{a,v'}$、$\lambda_{a,v'}$ 的 $\mu_{a,v'}$，字节码可能保持不变；这是假设~\ref{asm:committee}(iii) 所允许的，因为唯一性约束作用于 $(a,v)$，而不是 $h_{a,v}$；随后，引理~\ref{lem:preexec} 会无条件地收取新费用。**弃用（Deprecation）** 准入一个对每个输入都返回协议错误的模块版本；因此由定理~\ref{thm:invocation} 可知，在 $H_{a,v'}$ 之后，对 $a$ 的所有调用都会确定性失败，而无需重新部署任何合约。在每种情况下，修正都会在准入后的第 $H_{a,v'}-u_{a,v'}$ 个区块精确生效；只要某个节点能在这个窗口内完成准备，它就在该高度处于就绪状态（注~\ref{rem:liveness}）。第 5.2 节测量了该窗口需要覆盖的准备时间。
