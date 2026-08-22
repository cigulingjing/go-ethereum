# claims.md

本文档定义论文允许提出的核心结论及其证据边界。论文正文、摘要和结论中的重要陈述应能够映射到本文中的 Claim。

---

## C1 — 密码算法接口与 Native 实现解耦

**Claim**

密码学协处理器将面向智能合约的密码算法调用接口与客户端中的 Native 密码算法实现解耦，使具体算法实现可以独立于 EVM 核心执行逻辑进行管理和替换。

**English**

> The cryptographic coprocessor decouples contract-facing cryptographic algorithm interfaces from their native implementations.

**Evidence**

* 系统架构设计
* Cryptographic Coprocessor 实现
* 动态模块注册与调用路径

**Boundary**

不得声称完全解耦整个区块链客户端，也不得将动态库加载本身描述为主要创新。

---

## C2 — 支持密码算法运行时动态升级

**Claim**

密码学协处理器支持在客户端持续运行过程中更新面向智能合约的密码算法实现，而无需通过传统客户端软件重新构建和整体部署的方式完成密码算法更新。

**English**

> The proposed architecture enables runtime upgrades of contract-facing cryptographic algorithms without replacing the entire blockchain client software.

**Evidence**

* 动态编译与模块加载机制
* Upgrade Contract
* Lab1 升级实验

**Boundary**

不得写成“无需任何客户端修改”或“零成本升级”。

本文升级的是密码算法实现，而不是整个 blockchain client。

---

## C3 — 升级准备脱离交易执行关键路径

**Claim**

密码算法的编译与加载由客户端异步执行，不位于升级交易的 EVM 执行关键路径，因此链下升级准备过程不会直接阻塞该交易的正常执行。

**English**

> Cryptographic algorithm compilation and loading are performed asynchronously outside the transaction execution critical path.

**Evidence**

* Event Listener
* 后台异步升级任务
* Lab1 各阶段时间数据

**Boundary**

该结论表示“编译和加载不阻塞交易执行路径”，不等价于证明整个升级期间系统性能完全不受影响。

除非增加持续交易或吞吐实验，不使用：

> zero-downtime upgrade

或：

> no availability impact

---

## C4 — 动态升级后的算法保持 Native 执行效率

**Claim**

密码学协处理器中的动态加载算法仍以客户端 Native 代码执行，其单次调用性能与 Precompiled Contract 实现处于相近水平。

**English**

> Dynamically loaded cryptographic algorithms retain execution performance comparable to native precompiled contracts.

**Evidence**

* Lab2
* Upgrade vs. Precompile 执行时间
* 多种密码算法测试结果

**Boundary**

论文重点是：

> comparable performance

不得默认声称：

* faster than precompiled contracts
* significantly outperforms precompiled contracts
* universally better performance

除非数据与统计分析明确支持。

---

## C5 — 相比 Solidity 更适合复杂密码计算

**Claim**

对于部分计算复杂度较高的密码算法，Native 执行能够避免 Solidity/EVM 实现产生的大量指令执行和 Gas 开销。

**English**

> Native execution avoids the high EVM execution overhead incurred by contract-level implementations for computationally intensive cryptographic algorithms.

**Evidence**

* Lab2 Solidity / Precompile / Upgrade 对比
* 复杂密码算法 Gas 与执行时间结果

**Boundary**

不得声称所有密码算法都显著优于 Solidity。

Add 等简单操作可能不存在明显优势。

---

## C6 — 节点可以异步完成升级准备

**Claim**

不同区块链节点无需在相同物理时间完成新密码算法的编译与加载，可以独立完成升级准备。

**English**

> Blockchain nodes may complete cryptographic algorithm preparation asynchronously.

**Evidence**

* Lab1 多节点 Ready 时间
* 不同节点完成升级准备的时间差

**Boundary**

这里的“异步”仅指：

> upgrade preparation

不得描述为：

> asynchronous activation

算法正式生效由统一 Activation Block 决定。

---

## C7 — Activation Block 提供确定性版本切换

**Claim**

系统将升级准备时间与算法生效时间分离。节点依据统一的 Activation Block 决定密码算法版本，从而避免节点根据本地 Ready 时间独立切换算法。

**English**

> The activation block separates asynchronous upgrade preparation from deterministic algorithm activation.

固定表达：

> **Asynchronous Preparation, Deterministic Activation**

**Evidence**

* Upgrade Protocol
* Activation Block 设计
* Lab3

**Boundary**

不得声称所有节点在相同物理时间完成升级。

论文保证的是：

> 相同链状态下采用相同的版本选择规则。

---

## C8 — 升级过程中保持状态一致性

**Claim**

在本文测试环境中，即使节点完成升级准备的时间不同，基于 Activation Block 的版本切换机制仍能够使节点在相同区块高度执行相同版本，并保持交易执行结果和区块链状态一致。

**English**

> Under the evaluated upgrade scenarios, deterministic activation preserves version and state consistency across nodes despite asynchronous preparation.

**Evidence**

* Lab3
* algorithm version
* transaction result
* block hash
* state root

核心验证关系：

```text
StateRoot_1(H)
=
StateRoot_2(H)
=
...
=
StateRoot_N(H)
```

**Boundary**

必须使用：

> under the evaluated scenarios

或等价限制。

不得将有限规模实验描述为：

* formal proof
* universal consistency guarantee
* consensus safety proof

---

## C9 — 未完成升级的节点不得继续旧版本执行

**Claim**

当区块达到 Activation Block 后，未完成目标算法准备的节点不能继续使用旧算法处理需要新版本语义的调用，应拒绝相关执行或进入 fail-stop 状态。

**English**

> A node that is not ready at the activation block must not continue executing the obsolete algorithm version.

**Evidence**

* 升级协议
* 版本检查机制
* 客户端失败处理逻辑

**Boundary**

该结论属于设计约束。

只有在 Lab3 实际测试异常节点时，才能进一步声称该行为经过实验验证。

---

# Evaluation Claims

三个实验分别主要支持：

| Experiment                  | Primary Claims    |
| --------------------------- | ----------------- |
| Lab1 — Upgrade Latency      | C2, C3, C6        |
| Lab2 — Execution Efficiency | C4, C5            |
| Lab3 — State Consistency    | C7, C8, C9（若实际测试） |

---

# 核心论文结论

全文最终应收敛到以下逻辑：

```text
Cryptographic Coprocessor
        ↓
C1: Interface / Implementation Decoupling
        ↓
C2: Runtime Algorithm Upgrade
        ↓
C3 + C6: Asynchronous Preparation
        ↓
C7: Deterministic Activation
        ↓
C8: State Consistency

同时：

Cryptographic Coprocessor
        ↓
Native Execution
        ↓
C4 + C5: Execution Efficiency
```

最终核心结论可以概括为：

> The proposed cryptographic coprocessor enables runtime upgrades of contract-facing cryptographic algorithms while retaining native execution efficiency and preserving deterministic state consistency during asynchronous upgrade preparation.

该总结只能在 C1–C8 均得到设计或实验支撑后用于 Abstract、Introduction 和 Conclusion。
