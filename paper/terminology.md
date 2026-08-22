
terminology.md文档核心是统一英文与中文用词。

| 英文 | 中文 | 解释|
| -- | -- | -- |
| contract-facing cryptographic algorithms  | 面向智能合约的密码算法 | 智能合约侧调用但是实现在客户端的算法 |
| blockchain client | 区块链客户端 | 负责接收、验证区块并维护本地链状态的软件 |
| execution client | 执行客户端 | 负责执行交易、维护状态并支持 EVM 的 Ethereum client |
| consensus client | 共识客户端 | 负责区块选择和共识相关逻辑的 Ethereum client |
| externally-owned account (EOA) | 外部拥有账户（EOA） | 由私钥控制、可发起交易的账户 |
| contract account | 合约账户 | 由智能合约代码控制、在收到消息调用时执行字节码的账户 |
| post-quantum cryptographic algorithms | 后量子密码算法 | 能够抵抗已知量子攻击模型威胁的密码算法族 |
| post-quantum migration | 后量子转型 | 区块链系统从传统易受量子攻击的密码算法逐步迁移到后量子密码算法的过程 |
| Cryptographic Coprocessor | 密码学协处理器 | 与 EVM 侧密码算法调用并列协作的客户端 Native 计算与升级管理单元 |
| contract-facing interface | 面向合约的接口 | 智能合约用于访问密码算法能力的稳定 EVM/ABI 调用边界 |
| CodeStorage | 代码存储合约 / CodeStorage | 原型中用于提交算法元数据、触发事件并提供 `callFunc` 调用入口的 EVM-facing interface |
| Management Contract | 管理合约 | 链上发布升级提案、维护版本元数据并提供调用入口的管理组件，本文原型中由 CodeStorage 承担 |
| Upgrade Contract | 升级合约 | 合约侧统一发布升级提案和算法元数据的链上入口，本文原型中由 CodeStorage 承担 |
| upgrade proposal | 升级提案 | 合约侧提交的新算法版本、源码、Gas、ABI 类型与激活区块等元数据 |
| Event Listener | 事件监听器 | 客户端链下订阅 CodeStorage 升级事件并触发本地准备任务的组件 |
| activation service | 激活服务 | 客户端内负责解码源码、编译 plugin、加载符号和持久化本地元数据的组件 |
| prepared version | 已准备版本 | 节点本地已经完成编译、加载和符号验证但尚不一定生效的算法版本 |
| active version | 活动版本 | 给定区块高度下由版本选择规则确定并用于执行的算法版本 |
| version selector | 版本选择器 | 根据算法名称与区块高度选择应使用算法版本的逻辑 |
| runtime upgrade | 运行时升级 | 客户端持续运行期间更新面向智能合约的密码算法实现 |
| dynamic software update (DSU) | 动态软件升级 | 程序或服务持续运行期间更新代码、组件或执行逻辑的通用软件升级技术 |
| asynchronous upgrade preparation | 异步升级准备 | 节点在链下异步完成算法编译、加载和本地准备的过程 |
| deterministic activation | 确定性激活 | 节点依据统一链上规则在指定区块高度切换算法版本 |
| Activation Block | 激活区块 | 新算法版本开始生效的链上区块高度 |
| hard fork | 硬分叉 | 区块链协议规则发生不向后兼容变更时形成的升级方式 |
| soft fork | 软分叉 | 区块链协议规则通过向后兼容约束实现的升级方式 |
| on-chain governance | 链上治理 | 通过链上提案、投票或状态规则协调协议变更的治理机制 |
| Native implementation | Native 实现 | 在客户端原生代码中实现的密码算法逻辑 |
| Precompiled Contract | 预编译合约 | 客户端内置的 EVM 特殊地址 Native 计算接口 |
| Solidity contract implementation | Solidity 合约实现 | 完全由 Solidity/EVM 指令执行的算法实现 |
| dynamic loading | 动态加载 | 程序运行期间加载外部构建的共享库或 plugin 并解析其导出符号 |
| shared library | 共享库 | 可在运行时被宿主程序加载的二进制产物，如 `.so` |
| plugin | 插件 | 由宿主程序在运行时加载的独立模块 |
| upgrade latency | 升级延迟 | 从提交升级交易到目标节点可调用新算法的控制端可观测时间 |
| execution efficiency | 执行效率 | 算法调用的 `eth_call` 延迟与 `gasEstimate` 表现 |
| state consistency | 状态一致性 | 已评估场景下各节点链视图、版本选择和算法输出保持一致 |
| version consistency | 版本一致性 | 同一采样区块上各节点选择相同且符合预期的算法版本 |
| execution consistency | 执行一致性 | 相同输入和采样区块下各节点返回相同算法输出 |
| chain-view consistency | 链视图一致性 | 同一 canonical 采样区块上各节点 `head_hash`、`block_hash` 与 receipt 可见性一致 |
| gasEstimate | Gas 估算 | `eth_estimateGas` 返回的调用成本估算，不等同于 receipt `gasUsed` |
| eth_call latency | `eth_call` 延迟 | 通过 RPC 触发 EVM 调用路径的观测耗时，不是纯算法核函数时间 |
| gas pricing mechanism | Gas 定价机制 | 为执行、存储、calldata 和数据可用性等资源设定链上费用的规则 |
| denial-of-service (DoS) attack | 拒绝服务攻击（DoS） | 攻击者通过大量请求、交易或资源占用降低系统可用性的攻击 |
| data availability saturation attack | 数据可用性饱和攻击 | 通过过量数据提交或传播需求占满区块链数据可用性资源的攻击风险 |
| implementation-language compatibility | 实现语言兼容性 | 支持不同语言编写或编译的算法模块在统一调用边界下被加载、验证和执行 |
