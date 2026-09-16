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

### 5.1 Experimental Setup

实验在 20 节点 Clique 私有链（`period=0`，chain ID 11223344）上进行，客户端为修改版 Geth v1.17.5-unstable（commit df28d2ab8），Go 1.25.7 编译；Solidity 使用 solc 0.8.26，WASM 使用 TinyGo 0.42.0 构建。升级延迟为控制端可观测的 setup 时间，**不含共识与出块等待**；执行阶段指标为 `eth_call` 延迟与 `eth_estimateGas` 对 `callFunc` 返回的 `gasEstimate`（非 receipt `gasUsed`）。主机硬件规格待补全以便完全复现。

下表列出 8 个被测算法在三路径上的实现体量（均为源文件或模块字节数，单位 B），以及各自在评测中的**作用**。Go 大小指 `experiments/cryptoupgrade/algorithm/go/` 下 Native 参考源码；WASM 为上传的 TinyGo 模块；Solidity 为 `contracts/src/` 下基准合约 `.sol` 源文件大小。Dh2048Secret、PedersenCommit 的 Solidity 列为薄封装合约体积，模运算逻辑在共享库 `BigMod.sol`（8524 B）中；部署 bytecode 与 Gas 以链接后完整实现为准。

| 算法名字 | Go (B) | WASM (B) | Solidity (B) | 作用 |
|---|---:|---:|---:|---|
| Add | 156 | 11287 | 181 | 非密码基线 |
| Sha256 | 210 | 68288 | 219 | 密码哈希 |
| Blake2bSum256 | 4911 | 8724 | 5340 | 密码哈希 |
| Pbkdf2Sha256 | 1377 | 72580 | 2269 | 密钥派生 |
| Dh2048Secret | 1595 | 45919 | 473* | 密钥交换 |
| PedersenCommit | 1871 | 95780 | 645* | 承诺方案 |
| SchnorrVerify | 2171 | 95960 | 962 | 数字签名验证 |
| PolynomialMul | 790 | 7987 | 953 | 代数运算 |

\* 仅统计算法合约 `.sol` 源文件；模运算见共享 `BigMod.sol`。

对比方案：升级效率比较 EvoCrypt（WASM 升级上传）与 Solidity 合约部署的 **setup 延迟**及对应 setup 交易的 **`gasEstimate`**（普通 EVM 交易计价，非协议层升级 Gas 机制）；执行效率在 setup 完成后比较 EvoCrypt、**Native algorithm** 与 Solidity Contract 的 `callFunc` 延迟与调用 Gas。

### 5.2 Upgrade Efficiency

EvoCrypt 与 Solidity 实现密码算法可调用化过程的 setup 延迟对比，以及 upload/部署交易的 `gasEstimate` 对比（Figure upgrade-latency / upgrade-gas）；协议设计不对升级交易单独定义协处理器 Gas 规则，仅调用阶段使用链上 invocation gas 元数据。

### 5.3 Execution Efficiency

EvoCrypt 与 Solidity 升级后，以及与 Native 方案，密码算法执行耗时与调用 Gas 对比


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
