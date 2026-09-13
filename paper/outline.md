# EvoCrypt: An Upgradable Cryptographic Coprocessor for the Ethereum Virtual Machine

面向以太坊虚拟机的可升级密码协处理器。

## Abstract

说明智能合约通过合约代码调用密码算法存在执行效率低、Gas 开销高的问题，而预编译合约依赖客户端升级，缺少灵活性。本文提出一种面向 EVM 的密码协处理器，通过调用旁路执行 WASM 密码模块，并利用链上提案和指定块高完成动态升级。实验从升级效率和执行效率两个方面进行评估，验证方案能够在不中断节点服务的情况下更新密码算法，并获得接近预编译合约的执行效率。

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

介绍 WASM 的跨平台字节码、沙箱隔离、线性内存和确定性执行特征，说明其作为密码模块载体的适用性。

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

说明升级流程：开发者提交升级提案，节点监听链上事件，Upgrade Handler 获取并校验 WASM 模块，WASM Runtime 完成加载后等待指定区块激活。


根据区块高度选择算法版本：

$$
V(m,b)=
\begin{cases}
V_{\mathrm{old}}, & b < H,\\
V_{\mathrm{new}}, & b \ge H.
\end{cases}
$$

说明未完成模块校验或加载的节点在激活后停止相关执行，避免继续使用旧版本。


### 3.3 Invocation Protocol

定义统一调用接口，用户将调用算法以及参数序列化为输入,在合约侧调用指定合约地址的指定方法：智能合约构造算法名称和 ABI 参数，EVM 通过 Identity Hooks 识别调用，密码协处理器执行对应 WASM 模块，并将结果编码后返回 EVM。


**调用Gas费用计算**

采用类似预编译合约的 Gas 规则：

$$
G_a(x)=G_{\mathrm{base},a}
+\left\lceil\frac{L(x)}{W}\right\rceil G_{\mathrm{unit},a}.
$$

说明 Gas 由算法类型和确定性输入参数计算，不能根据节点实际 CPU 时间动态计费。输入参数为统一接口的输入数据长度。不同算法可以定义不同的输入参数来调整gas费用与复杂度的关系。

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
3. 动态Gas计费策略，管理合约可以调整每一个算法的**输入gas计费参数**，通过调整参数能够让gas费匹配计算资源
4. 一致性保证机制，在 **Activation Block** 区块完成升级算法的启用，达成全网执行逻辑的共识，确保指定交易在指定区块计算结果是一直的。



## 5 Evaluation

### 5.1 Experimental Setup

| 算法名字 | Go 文件大小 | WASM 文件大小 | Solidity 文件大小 | 数学难题 / 计算类型 |
|---|---:|---:|---:|---|
| Add | 156 B | 11.0 KB (11,287 B) | 181 B | 非密码基线；整数加法 |
| Sha256 | 210 B | 66.7 KB (68,288 B) | 219 B | 密码哈希；SHA-256 单向压缩函数 |
| Blake2bSum256 | 4.8 KB (4,911 B) | 8.5 KB (8,724 B) | 5.2 KB (5,340 B) | 密码哈希；BLAKE2b 单向压缩函数 |
| Pbkdf2Sha256 | 1.3 KB (1,377 B) | 70.9 KB (72,580 B) | 2.2 KB (2,269 B) | 密钥派生；PBKDF2-HMAC-SHA256 迭代拉伸 |
| Dh2048Secret | 1.6 KB (1,595 B) | 44.8 KB (45,919 B) | 473 B* | 密钥交换；有限域离散对数（DH-2048, RFC 3526） |
| PedersenCommit | 1.8 KB (1,871 B) | 93.5 KB (95,780 B) | 645 B* | 承诺方案；有限域离散对数（Pedersen, mod p） |
| SchnorrVerify | 2.1 KB (2,171 B) | 93.7 KB (95,960 B) | 962 B | 数字签名验证；有限域离散对数（Schnorr, mod p） |
| PolynomialMul | 790 B | 7.8 KB (7,987 B) | 953 B | 代数运算；有限域多项式乘法（mod q） |


### 5.2 Upgrade Efficiency

EvoCrypt与Solidity实现的密码算法升级交易消耗执行时间对比
EvoCrypt与Solidity实现的密码算法升级交易消耗的Gas费用对比

### 5.3 Execution Efficiency


EvoCrypt与Solidity升级后，以及与原生方案，密码算法执行耗时对比
EvoCrypt与Solidity升级后，密码算法执行消耗Gas费用对比


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
