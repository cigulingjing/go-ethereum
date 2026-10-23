# EvoCrypt: An Upgradable Cryptographic Coprocessor for the Ethereum Virtual Machine

面向以太坊虚拟机的可升级密码协处理器。

## Abstract

智能合约调用密码算法时，Solidity 实现开发成本高、执行延迟与 Gas 开销大。预编译合约效率较高，但算法逻辑固化在客户端中，链上运行期缺少可替换、可演进的动态更新机制，通常只能依赖客户端重编译与协调发布。本文提出面向 EVM 的 EvoCrypt 密码协处理器：在稳定合约接口下加载可替换的 WASM 模块，并通过链上元数据与激活区块完成版本切换，从而补上预编译路径所不具备的动态升级能力。我们在 Geth 上实现原型，在本地 20 节点 Clique 私有网络（`period=0`）中测量不含共识等待的升级延迟，以及执行阶段的延迟与调用 Gas 估算。

## 1 Introduction

介绍智能合约对密码算法的需求，以及后量子迁移等场景对密码敏捷性的要求。

分析现有方案的不足：

- Solidity 开发密码算法难度大，并且实现以后密码算法执行效率低、Gas 开销高；
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

### 2.1 Smart contract

介绍 Solidity 合约、EVM 操作码和预编译合约等密码算法执行方式，分析其在效率、Gas 成本和可升级性方面的差异。

### 2.2 Live Upgrade

介绍区块链客户端升级过程，以及节点重启、升级协调和版本切换问题。

### 2.3 WebAssembly Runtime

介绍 WASM 的跨平台字节码、沙箱隔离、线性内存与可移植执行语义。说明协处理器以同一 WASM 模块字节码作为权威算法载荷，在兼容运行时上不同 OS/架构节点对相同输入产生相同输出，从而支撑跨节点执行一致性与区块链状态重放要求（区别于进程内 native plugin 的平台相关二进制）。

## 3 Coprocessor Architecture

### 3.1 System Overview

介绍系统整体架构，包括 Execution Engine、Identity Hooks、Cryptographic Coprocessor、Upgrade Handler、WASM Runtime 和 WASM Modules。说明普通交易仍由 EVM 执行，只有识别到特定管理合约调用时才进入协处理器。

### 3.2 Upgrade Protocol

定义升级提案内容：

```text
moduleID
moduleVersion
wasmHash
wasmBytes
activationHeight
```

说明升级流程：开发者提交升级提案，节点监听链上事件，Upgrade Handler 从 Management Contract 获取并校验 WASM 模块，WASM Runtime 完成加载；链上事件仅携带算法名、版本与激活高度，完整元数据以合约存储为准。流程见 **Algorithm 1（Activation-Block-based WASM upgrade protocol）**。

版本选择：区块高度低于 Activation Block 时使用旧版本，达到或超过 Activation Block 时使用新版本；若存在多个已调度版本，则在链上记录中选择激活高度不超过当前区块高度的最新版本。

说明采用强制升级策略：当达到 Activation Block 且链上已选中新版本时，节点不得回退旧实现；若本地尚未 prepared，则从 Management Contract 强制拉取 WASM 并完成校验、编译与实例化后再执行，而非直接返回 execution error。

### 3.3 Invocation Protocol

定义统一调用接口：合约构造算法名称与 ABI 参数，经 `callFunc` 发起 `STATICCALL`；Identity Hooks 识别 CodeStorage 调用，协处理器按当前块高选版、必要时强制准备、校验调用 Gas、执行 WASM 并返回 ABI 编码结果。流程见 **Algorithm 2（Contract-to-coprocessor invocation protocol）**。

**调用 Gas 费用计算**：采用类似预编译合约的确定性规则——基础 Gas 加与输入长度相关的线性项；Gas 由算法类型与输入数据长度计算，不能按节点 CPU 时间动态计费；各算法版本在链上元数据中注册对应参数。

### 3.6 Prototype Implementation

介绍基于 Geth 和 WASM Runtime 的原型实现，包括 EVM 调用识别、WASM 模块加载、ABI 参数转换、版本状态管理、块高激活和 Gas 扣除。

## 4 Security Analysis

### 4.1 Attack Circumstances

本文涉及到的攻击场景包括：

1. 虚拟机逃逸问题（Sandbox Escape），WASM执行逻辑跳出了EVM执行范围之外。
2. 升级元数据不一致，攻击者发布与代码不相符的提案，影响正确的升级合约数据
3. 资源耗尽攻击，
4. 异步更新带来的状态不一致问题，

### 4.2 Security Solutions

对应攻击场景的解决方案为：、
1. WASM隔离性，防止恶意代码访问宿主机信息
2. 合约数据是唯一正确来源，区块链不可篡改的性质，在合约记录升级metadata，所有节点依据打包完成的Event接受升级数据
3. 调用阶段确定性 Gas 计费：链上元数据为每个算法版本注册调用 Gas 参数，协处理器在 `callFunc` 执行时按输入长度等确定性规则扣费，而非按升级上传交易的 EVM Gas 机制建模
4. 一致性保证机制：在 **Activation Block** 按 \(b<H\) / \(b\ge H\) 统一选版；未 prepared 时**强制拉取并完成升级**，禁止静默回退旧版本。



## 5 Evaluation

本章以根目录 `evocrypt-old.tex` 第五章为基线。5.1–5.3 以旧稿为基础，Table 1 和 Fig. 3 右图已用 2026-10-23 当前源码复测数据更新；5.4 替换为本次 PBKDF2 专项调用实验；旧稿 complex-algorithm 三段论述移至新增 5.5，并同步更新数据。

### 5.1 Implementation

保留旧稿实验平台、工具链、八种基础算法的实现体量表和指标说明。该部分的旧实验设置与 5.4 的新增单节点实验分开表述。

### 5.2 Upgrade Efficiency

保留旧稿升级延迟；升级 Gas 使用 2026-10-23 当前源码复测的 receipt.gasUsed，七种算法比 Solidity 部署更低，Add 更高。Table 1 采用同次重建的源码和制品字节数。完整证据见 `experiments/cryptoupgrade/results/artifact-upgrade-recheck/20261023/report.md`。缺失 PDF 从 `/home/liuqi/project/go-ethereum-e72de4d/paper/image/` 复制到当前 `paper/image/`。

### 5.3 Upgrade Consistency

保留旧稿 20 节点升级一致性论述和 `consistence_scene.pdf`。

### 5.4 Execution Efficiency

仅讨论 PBKDF2：

- 调用效率：内部迭代次数 1–100，比较 Solidity、Precompiled Contract 和 EvoCrypt；指标为 `eth_call` RPC 延迟中位数，不能写成纯算法执行时间。
- 调用 Gas：内部迭代固定 10 次，同笔交易调用次数为 1、2、3、4、5、6、8、12、16；基线为 g₀=3000、α=1/2、N₀=2，使用真实收据总 Gas，并展示三个参数的敏感性。
- 图源为 `paper/image/experiments_result.xlsx`；基线 Gas 与 `experiments/cryptoupgrade/results/chapter-five/20261023-single-node-billing/report.md` 一致。正文明确费用优势仅适用于实验费率。

### 5.5 Functional Validation

保留旧稿 complex-algorithm 三段论述顺序：验证目标与指标、成功接入对扩展能力的支持、相对原生实现的性能代价。仅保留六种复杂算法，表中时间取自本次工作簿“复杂算法实验”，使用 WASM/native 比值列替代旧版模块大小列，避免将旧模块体积作为本次实验测量值。

Groth16/BLS12-381 的 WASM 时间采用工作簿的 24.809 ms，原生为 3.518 ms；整体比值范围为 1.91–7.05 倍。表格和正文使用独立验证微基准口径，不与 5.4 的 RPC 延迟混合。保留旧稿扩展能力论述，但不把六种算法的微基准解释为本次新增链上上传实验。


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
- WASM 密码模块具有接近原生代码实现的执行效率，在简单密码学操作中效率与solidity持平，在复杂密码学操作中好于solidity；
- 相比 Solidity 实现，方案能够显著降低密码算法的执行开销。
