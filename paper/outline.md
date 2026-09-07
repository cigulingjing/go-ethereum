# An Upgradable Cryptographic Coprocessor for the Ethereum Virtual Machine

面向以太坊虚拟机的可升级密码协处理器。

## Abstract

说明智能合约通过合约代码调用密码算法存在执行效率低、Gas 开销高的问题，而预编译合约依赖客户端升级，缺少灵活性。本文提出一种面向 EVM 的密码协处理器，通过调用旁路执行 WASM 密码模块，并利用链上提案和指定块高完成动态升级。实验从升级效率和执行效率两个方面进行评估，验证方案能够在不中断节点服务的情况下更新密码算法，并获得接近预编译合约的执行效率。

## 1 Introduction

介绍智能合约对密码算法的需求，以及后量子迁移等场景对密码敏捷性的要求。

分析现有方案的不足：

- Solidity 实现执行效率低、Gas 开销高；
- 预编译合约效率较高，但新增算法需要升级客户端；
- 节点独立升级可能造成服务中断和版本不一致。

概述本文方案：

- 在 EVM 中引入密码协处理器；
- 通过统一接口调用 WASM 密码模块；
- 通过链上提案发布升级信息；
- 通过指定区块高度完成版本切换。

总结本文贡献：

1. 设计面向 EVM 的密码协处理器架构；
2. 设计链上协调的 WASM 密码算法升级协议；
3. 实现原型并评估升级开销、执行时间和 Gas 消耗。

## 2 Technical Background

### 2.1 Cryptographic Execution

介绍 Solidity 合约、EVM 操作码和预编译合约等密码算法执行方式，分析其在效率、Gas 成本和可升级性方面的差异。

### 2.2 Runtime Upgrade

介绍区块链客户端升级过程，以及节点重启、升级协调和版本切换问题。

### 2.3 WebAssembly Runtime

介绍 WASM 的跨平台字节码、沙箱隔离、线性内存和确定性执行特征，说明其作为密码模块载体的适用性。

## 3 Coprocessor Architecture

### 3.1 System Overview

介绍系统整体架构，包括 Execution Engine、Identity Hooks、Cryptographic Coprocessor、Upgrade Handler、WASM Runtime 和 WASM Modules。说明普通交易仍由 EVM 执行，只有识别到特定管理合约调用时才进入协处理器。

### 3.2 Invocation Abstraction

定义统一调用接口：

```solidity
callFunc(string algorithm, bytes input)
```

说明调用过程：智能合约构造算法名称和 ABI 参数，EVM 通过 Identity Hooks 识别调用，密码协处理器执行对应 WASM 模块，并将结果编码后返回 EVM。

### 3.3 Upgrade Protocol

定义升级提案内容：

```text
moduleID
moduleVersion
wasmHash
wasmBytes
activationHeight
```

说明升级流程：开发者提交升级提案，节点监听链上事件，Upgrade Handler 获取并校验 WASM 模块，WASM Runtime 完成加载后等待指定区块激活。

### 3.4 Deterministic Activation

根据区块高度选择算法版本：

$$
V(m,b)=
\begin{cases}
V_{\mathrm{old}}, & b < H,\\
V_{\mathrm{new}}, & b \ge H.
\end{cases}
$$

说明未完成模块校验或加载的节点在激活后停止相关执行，避免继续使用旧版本。

### 3.5 Resource Accounting

采用类似预编译合约的 Gas 规则：

$$
G_a(x)=G_{\mathrm{base},a}
+\left\lceil\frac{L(x)}{W}\right\rceil G_{\mathrm{unit},a}.
$$

说明 Gas 由算法类型和确定性输入参数计算，不能根据节点实际 CPU 时间动态计费。

### 3.6 Prototype Implementation

介绍基于 Geth 和 WASM Runtime 的原型实现，包括 EVM 调用识别、WASM 模块加载、ABI 参数转换、版本状态管理、块高激活和 Gas 扣除。

## 4 Security Analysis

### 4.1 Threat Model

定义恶意 WASM 模块、资源耗尽攻击、错误升级构件和节点升级延迟等威胁。

### 4.2 Execution Isolation

分析 WASM 沙箱、线性内存和受限宿主接口如何降低虚拟机逃逸风险，说明 WASM 模块不能直接访问 Geth 进程内存、文件系统、系统网络和未授权宿主函数。

### 4.3 Resource Abuse

分析低 Gas 定价可能造成的计算资源滥用问题。说明通过确定性 Gas 计量、内存上限、调用深度限制和执行预算防止拒绝服务攻击。

### 4.4 Upgrade Consistency

从协议设计论证升级一致性：模块哈希确定唯一执行构件，链上提案确定目标版本，区块高度确定统一切换边界，节点本地时间不参与版本选择，未就绪节点不会继续执行旧模块。一致性作为安全属性分析，不再单独设置实验。

## 5 Evaluation

### 5.1 Experimental Setup

说明 Geth 版本、共识配置、硬件环境、WASM Runtime 和节点配置。

| WASM 模块 | 导出入口 | Solidity 文件 | Solidity 入口 | 状态 |
| --- | --- | --- | --- | --- |
| `add.wasm` | `Add` | `Add.sol` | `Add` | 待按 WASM 路径重做 |
| `blake2b.wasm` | `Sum256` | `Blake2b.sol` | `Sum256` | 待按 WASM 路径重做 |
| `dh2048.wasm` | `Dh2048Secret` | `Dh2048.sol` | `Dh2048Secret` | 待按 WASM 路径重做 |
| `pbkdf2_sha256.wasm` | `Pbkdf2Sha256` | `Pbkdf2Sha256.sol` | `Pbkdf2Sha256` | 待按 WASM 路径重做 |
| `pedersen_commit.wasm` | `PedersenCommit` | `PedersenCommit.sol` | `PedersenCommit` | 待按 WASM 路径重做 |
| `schnorr_proof.wasm` | `SchnorrVerify` | `SchnorrProof.sol` | `SchnorrVerify` | 待按 WASM 路径重做 |
| `sha256.wasm` | `Sha256` | `Sha256.sol` | `Sha256` | 待按 WASM 路径重做 |

### 5.2 Upgrade Efficiency

评估不同节点规模和密码模块下的升级开销。

测量指标包括：

- 提案确认时间；
- 模块获取时间；
- 模块校验时间；
- WASM 加载时间；
- 单节点准备时间；
- 全网升级完成时间；
- 升级期间交易成功率；
- 最长出块间隔。

验证升级开销是否可控，以及升级期间是否能够继续提供服务。

### 5.3 Execution Efficiency

比较 WASM 密码协处理器、预编译合约和 Solidity 合约的执行效率。

测量指标包括：

- 算法执行时间；
- P95 执行时间；
- Gas 消耗；
- 相对性能提升；
- 相对 Gas 变化。

验证密码协处理器是否具有接近预编译合约、优于 Solidity 合约的执行效率。

## 6 Related Work

### 6.1 EVM Cryptographic Extensions

总结 EVM 操作码、预编译合约和密码算法扩展研究。

### 6.2 Blockchain Runtime Upgrades

总结硬分叉、链上治理、运行时升级和密码敏捷性研究。

### 6.3 WebAssembly Execution

总结 WASM 在区块链中的确定性执行、沙箱隔离和资源计量研究，并说明本文与现有工作的区别。

## 7 Conclusion

总结本文提出的 EVM 密码协处理器及其升级协议，概括以下实验结论：

- 密码算法可以在节点不停机的情况下完成更新；
- 升级开销随节点规模增长保持可控；
- WASM 密码模块具有接近预编译合约的执行效率；
- 相比 Solidity 实现，方案能够降低密码算法的执行开销。

最后说明当前限制及未来工作，包括更完善的 WASM 确定性规范、Gas 参数校准、升级治理和更多密码算法支持。