
terminology.md文档核心是统一英文与中文用词。

| 英文 | 中文 | 解释|
| -- | -- | -- |
| Non‑halt redeployment | 不停机更新 | -- |
| contract-facing cryptographic algorithms  | 面向智能合约的密码算法 | 智能合约侧调用但是实现在客户端的算法 |
| blockchain client | 区块链客户端 | 负责接收、验证区块并维护本地链状态的软件 |
| execution client | 执行客户端 | 负责执行交易、维护状态并支持 EVM 的 Ethereum client |
| consensus client | 共识客户端 | 负责区块选择和共识相关逻辑的 Ethereum client |
| externally-owned account (EOA) | 外部拥有账户（EOA） | 由私钥控制、可发起交易的账户 |
| contract account | 合约账户 | 由智能合约代码控制、在收到消息调用时执行字节码的账户 |
| post-quantum cryptographic algorithms | 后量子密码算法 | 能够抵抗已知量子攻击模型威胁的密码算法族 |
| post-quantum migration | 后量子转型 | 区块链系统从传统易受量子攻击的密码算法逐步迁移到后量子密码算法的过程 |
| Cryptographic Coprocessor | 密码学协处理器 | 与 EVM 侧密码算法调用并列协作的客户端 WASM 计算与升级管理单元 |
| Execution Engine | 执行引擎 | 执行普通 EVM 交易并在识别到管理合约调用时进入协处理器边界的执行组件 |
| Identity Hooks | 身份识别钩子 | 原型中识别 Management Contract / CodeStorage 调用并将特定调用路由到 Cryptographic Coprocessor 的 EVM 集成点 |
| contract-facing interface | 面向合约的接口 | 智能合约用于访问密码算法能力的稳定 EVM/ABI 调用边界 |
| CodeStorage | 代码存储合约 / CodeStorage | 原型中用于提交算法元数据、触发事件并提供 `callFunc` 调用入口的 EVM-facing interface |
| Management Contract | 管理合约 | 链上发布升级提案、维护版本元数据并提供调用入口的管理组件，本文原型中由 CodeStorage 承担 |
| Upgrade Contract | 升级合约 | 合约侧统一发布升级提案和算法元数据的链上入口，本文原型中由 CodeStorage 承担 |
| upgrade proposal | 升级提案 | 合约侧提交的新算法版本、源码、Gas、ABI 类型与激活区块等元数据 |
| Event Listener | 事件监听器 | 客户端链下订阅 CodeStorage 升级事件并触发本地准备任务的组件 |
| Upgrade Handler | 升级处理器 | 客户端内负责接收升级事件、读取链上元数据并协调本地 WASM module 准备流程的组件 |
| activation service | 激活服务 | 客户端内负责解码源码、编译 WASM module、验证导出接口和持久化本地元数据的组件 |
| prepared version | 已准备版本 | 节点本地已经完成编译、加载和符号验证但尚不一定生效的算法版本 |
| active version | 活动版本 | 给定区块高度下由版本选择规则确定并用于执行的算法版本 |
| version selector | 版本选择器 | 根据算法名称与区块高度选择应使用算法版本的逻辑 |
| runtime upgrade | 运行时升级 | 客户端持续运行期间更新面向智能合约的密码算法实现 |
| dynamic software update (DSU) | 动态软件升级 | 程序或服务持续运行期间更新代码、组件或执行逻辑的通用软件升级技术 |
| asynchronous upgrade preparation | 异步升级准备 | 节点在链下异步完成算法编译、加载和本地准备的过程 |
| deterministic activation | 确定性激活 | 节点依据统一链上规则在指定区块高度切换算法版本 |
| Activation Block | 激活区块 | 新算法版本开始生效的链上区块高度 |
| WASM module | WASM 模块 | 编译为 WebAssembly 字节码并在受控运行时中执行的算法模块 |
| WASM bytecode | WASM 字节码 | WASM module 的可序列化字节载荷，用于上传、哈希校验、编译和实例化 |
| WASM runtime | WASM 运行时 | 执行 WASM module 并提供导入、内存与调用边界的宿主运行时 |
| sandbox boundary | 沙盒边界 | WASM 运行时对模块可访问能力施加的限制 |
| hard fork | 硬分叉 | 区块链协议规则发生不向后兼容变更时形成的升级方式 |
| soft fork | 软分叉 | 区块链协议规则通过向后兼容约束实现的升级方式 |
| on-chain governance | 链上治理 | 通过链上提案、投票或状态规则协调协议变更的治理机制 |
| Native implementation | Native 实现 | 在客户端原生代码中实现的密码算法逻辑 |
| Precompiled Contract | 预编译合约 | 客户端内置的 EVM 特殊地址 Native 计算接口 |
| Solidity contract implementation | Solidity 合约实现 | 完全由 Solidity/EVM 指令执行的算法实现 |
| dynamic loading | 动态加载 | 程序运行期间加载外部构建的共享库、plugin 或 WASM module 并解析其导出符号或接口 |
| shared library | 共享库 | 可在运行时被宿主程序加载的二进制产物，如 `.so` |
| plugin | 插件 | 由宿主程序在运行时加载的独立模块；在本文中仅作为 Go plugin 的历史对照术语使用 |
| upgrade latency | 升级延迟 | 从提交升级交易到目标节点可调用新算法的控制端可观测时间 |
| execution efficiency | 执行效率 | 算法调用的 `eth_call` 延迟与 `gasEstimate` 表现 |
| upgrade consistency | 升级一致性 | `Security Analysis` 中讨论的协议属性，指相同链视图下节点依据相同链上元数据和 Activation Block 选择算法版本；当前论文不设置一致性实验 |
| version consistency | 版本一致性 | 节点在相同链视图和相同区块高度下应选择相同算法版本的协议要求；当前论文不作为实验指标 |
| execution consistency | 执行一致性 | 相同输入和相同算法版本下应返回相同输出的协议要求；当前论文不作为实验指标 |
| chain-view consistency | 链视图一致性 | 节点观察到相同 canonical block 和相关 receipt 信息时的一致视图要求；当前论文不作为实验指标 |
| gasEstimate | Gas 估算 | `eth_estimateGas` 返回的调用成本估算，不等同于 receipt `gasUsed` |
| eth_call latency | `eth_call` 延迟 | 通过 RPC 触发 EVM 调用路径的观测耗时，不是纯算法核函数时间 |
| gas pricing mechanism | Gas 定价机制 | 为执行、存储、calldata 和数据可用性等资源设定链上费用的规则 |
| resource accounting | 资源计费 | 基于算法类型和确定性输入参数计算协处理器调用 Gas 的规则，不依赖节点本地 CPU 时间 |
| denial-of-service (DoS) attack | 拒绝服务攻击（DoS） | 攻击者通过大量请求、交易或资源占用降低系统可用性的攻击 |
| data availability saturation attack | 数据可用性饱和攻击 | 通过过量数据提交或传播需求占满区块链数据可用性资源的攻击风险 |
| implementation-language compatibility | 实现语言兼容性 | 支持不同语言编写或编译的算法模块在统一调用边界下被加载、验证和执行 |
