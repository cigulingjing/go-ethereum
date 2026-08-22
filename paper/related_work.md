# 国内外研究现状
本章首先从区块链拓展性出发，介绍相关研究发展，并从区块链架构优化、区块链虚拟机优化两个角度综述当前已有的优化方案。随后，从手续费计价调整和二层网络两方面介绍针对区块链手续费过高问题的主要解决思路。接着，对软件热更新领域研究进行综述，包括传统动态软件升级、分布式系统热更新和区块链热升级机制。最后，在上述研究的基础上，分析当前存在的问题及未来的发展趋势。
## 区块链拓展性
### 区块链架构优化
区块链作为融合分布式系统、密码学与经济学原理的复杂协同系统。随着区块链应用场景的不断延申，传统单体架构在灵活性、扩展性与效率上的瓶颈日益凸显。当前研究普遍聚焦于架构层面的重构与升级，通过模块解耦、层级优化、功能扩展等方式，为区块链系统适配多样化需求提供基础，并为迭代预留标准化接口。
许多研究着眼于区块链的功能模块化，通过定义接口构建分层架构，提升区块链系统的功能扩展性。Xu等人[13]提出了一个通用的模块化区块链分析框架。该框架将区块链系统分解为相互作用的模块，并系统总结了智能合约以及零知识证明在区块链系统的应用。表明了区块链模块化、模块接口标准化是未来技术演进的关键方向。其研究成果为后续的区块链虚拟机架构升级提供了理论基础。Fabric项目[14]支持共识协议、身份认证与策略的可插拔替换，从而能够适应不同的信任模型与业务场景。Fabric设计为分层架构，通过定义统一接口，支持各个组件的替换以及扩展。此外利用容器化技术屏蔽各个模块的实现细节。
许多研究着眼于区块链底层实现技术，通过重构区块链实现方式或者网络组织方式来提升功能扩展性或执行效率。Li等人[15]将微服务引入区块链系统，通过设计基于微服务的架构提升高并发与复杂业务环境系统的可用性与适应性。其主要有三个贡献，第一是基于微服务重构了区块链系统，将各个模块通过微服务的方式连接起来，提升了系统的灵活性、可用性以及性能。第二是，基于多子链结构，支持每个节点根据实际需求部署新的微服务。第三是针对智能合约的执行，设计了并行执行引擎，来缓解区块链系统在处理并发交易以及复杂合约执行的效率。Liu[16]等聚焦于智能合约的执行效率，将节点分为共识节点和执行节点，执行节点能够并行执行交易，共识节点能够异步地排序交易并处理执行节点的执行结果。通过优化共识和执行层的方式提升了复杂智能合约的执行效率。
上述方案从区块链的网络拓扑组织、模块职责划分等维度切入，研究区块链架构优化。其核心思路在于通过功能抽象，将区块链的复杂功能拆解为可独立演化的单元，并借助分层架构明确各模块的权责边界与功能定位。这种设计不仅显著提升了系统的整体运行效率，更增强了区块链的兼容性与可扩展性，为跨链交互与多生态融合奠定了坚实的底层架构基础。
### 区块链虚拟机优化
区块链虚拟机按照预定义的规则，执行交易，并修改区块链状态。区块链虚拟机提供了安全、可控的沙箱执行环，但是同样的也带来了灵活性受限的问题。随着区块链在许多领域的应用，虚拟机架构的限制导致了区块链生态扩展困难。当前许多研究聚焦于虚拟机架构上灵活性不足的问题进行研究。
许多研究着眼于区块链虚拟机的功能扩展，Han[17]等提出了VM-Studio，其将区块链虚拟机层扩展为了多虚拟机架构，以满足跨链智能合约验证以及执行需求。通过设计迁移以及装载协议，能够实现其他链虚拟机的部署以及执行，进而在不影响原始区块链交易执行性能的情况下实现了广泛的通用性。这样的设计使得区块链虚拟机的功能大大扩展，灵活性有了更大提升。HyperService[18]同样着眼于区块链系统的跨链场景，提出了一套统一的状态模型，将不同区块链抽象为带公共变量状态与函数的对象，屏蔽底层实现细节。而后基于此模型设计了HSL编程语言，允许开发者直接定义跨链实体、操作以及依赖关系。但是其交易执行依赖于静态分析，动态交易场景会引发状态不一致问题。Glimpse[19]通过轻量化客户端机制，降低了跨链通信的存储和开销，同时保持了高兼容性。针对当前轻客户端验证需要拉取全部区块头验证的问题，Glimpse提出的客户端采用按需验证模式，当有交易验证需求时，才会拉取特定的区块头。通过优化虚拟机外部接口的方式，提升了区块链系统的吞吐量。
此外将零知识证明引入虚拟机，降低虚拟机的验证成本也是一个常见思路。Zhang等人[20]进行了测量研究，针对WASM和EVM在执行智能合约性能，进行了综述，揭示当前以WASM作为执行引擎的区块链平台存在问题，并指出了当前WASM虚拟机部署面临的挑战。ZKSMT[21]一个用于在零知识环境下证明可满足性模理论中公式的有效性。并听过实例化两种常用理论验证了虚拟机的完备性以及可靠性。RISC Zero[22]提出了基于RISC-VISA的虚拟机，能够对运行的二进制生成零知识证明。支持开发者使用常规RISC-V工具链或Rust语言写程序，然后在零知识证明虚拟机中运行、生成证明、验证，从而极大降低零知识证明系统的开发门槛。Ceno框架[23]利用简洁、非交互式的零知识证明对任何代码进行可验证的计算。我们的方法将程序执行的证明分为两个阶段。在第一阶段，该过程将程序执行分解为多个部分，识别相同的部分并将其分组。然后通过允许不同数量的重复的数据并行电路证明这些段。在后续阶段，验证者检查这些段证明，根据段的重复数和原始程序重建程序的控制和数据流。第二阶段可以通过统一递归证明进一步证明。
总结来看，这些研究主要围绕区块链虚拟机的效率提升、功能扩展、安全增强等方向开展，这些研究共同推动区块链虚拟机向更灵活、高效、安全且易于开发的方向演进，为构建可扩展的下一代区块链基础设施奠定了坚实基础。
## 区块链升级机制
### 动态软件升级
动态软件升级（DSU）是一种在不停止主程序的情况下，完成代码的更新。软件动态升级领域按照实现方式可以划分为两类：内存地址修改以及虚拟化容器方案。内存地址修改将升级功能封装为若干插件，主程序通过预留的程序接口动态调用插件文件；虚拟化方案将可升级功能装载入虚拟机内，主程序作为稳定的宿主。软件升级时将待升级文件装载到虚拟机中并重新加载虚拟机，完成热升级过程。Zhang等人[34]提出的KylinX系统，通过页级动态映射实现类进程虚拟机（pVM）的创建与通信，利用库级动态映射支持运行时库更新与pVM回收，并以互信pVM家族、版本库权限管理两个机制保障安全，在保留静态Unikernel强隔离性的同时，解决了其动态管理能力缺失的问题，为云虚拟化场景下的实例动态升级提供了高效解决方案。Pina等人[35]提出了多版本执行框架Mvedsua，利用多版本执行与动态软件升级结合起来，实现了可靠、低延迟更新的更新方法，并可以容忍更新过程中的错误。Huang等人[4]利用Python程序语言解释型编译器的特性，开发了PYLIVE系统框架，支持安全且可移植的动态代码变更。但是解释性语言会牺牲执行效率，并且会限制与其他系统的兼容性。
### 分布式系统热更新
分布式场景下的动态升级对于一致性有了更加严苛的要求，网络的不确定性以及不同节点之间软件环境的差别，对于系统的一致性带来了巨大挑战。目前一些研究着眼于分布式系统的一致性进行研究。Ajmani等人[36]首次系统提出了分布式系统中的自动化动态升级机制，利用中心化服务器提供升级以及存储服务。Luciano等人[37]首次将动态依赖与版本一致性结合，利用拓扑学寻找最适合升级的窗口，兼顾了一致性以及升级效率问题。
### 区块链系统热更新
区块链系统的热更新相较于上述更新场景，额外需要保证合约更新前后状态一致性与交易可验证性。这是因为区块链的历史数据和交易验证逻辑依赖于旧版本的合约与执行规则。若更新后的合约改变了状态结构或验证逻辑，导致旧交易记录无法被新逻辑正确验证，从而破坏系统的可追溯性与共识安全。
一些研究不直接解决硬分叉问题，而是通过安全防御机制、区块链状态同步器等方式，降低硬分叉带来的影响。Cardano[43][40]项目提出了硬分叉组合器（HFC）的概念。当区块链高度高达指定的升级点时，硬分叉组合器自动切换到新的协议规则。但是要让不同协议共存于同一条链，必须设计复杂的抽象接口；此外，多协议拼接容易引入边界状态问题，例如在切换点处的状态迁移或规则兼容性。Ciampi[38]等人研究了区块链硬分叉后的同步问题，设计了在新链和旧链区块互相转化的编译器，让硬分叉式升级在去中心化场景下安全可控，消除分叉带来的不一致风险。
另一些研究着眼于避免硬分叉的发生，通过代理合约、动态软件升级等思路，实现不停机状态下区块链系统的热更新。Zindrors[39]深入研究了通过软分叉机制调整区块链的参数和算法的方法，该方案通过将区块链宏观经济政策相关的算法与参数抽象至智能合约中实现，从而保障了区块链状态的一致性。不过该方案存在适用范围局限：其核心设计适配于宏观经济政策调整这类逻辑相对简单的升级场景；若面临密码算法迭代等深度升级需求，会显著增加开发与部署成本，难以高效适配。Wang[11]等人设计并实现了Phoenix区块链客户端，将升级代码嵌入区块链交易，并通过即时编译实现动态更新，额外引入并行引擎提升交易执行效率。但是每次调用升级逻辑都需要从存储区提取代码，并加载到解释器中而后实现编译部署，相较于编译效率较低。Polkadot[41]采用链上治理和可升级运行时机制互相配合解决升级问题，其将链相关逻辑使用WASM编写并放置在链上。社区通过治理模块决定是否升级链逻辑。一旦通过提案，节点在同步新区块时，会读取链上存储的最新代码逻辑，自动切换执行逻辑。EVMPatch[42]框架针对智能合约的自动化即时补丁修复问题展开研究，通过设计字节码重写引擎，在最小化补丁的前提下，确保修补后的新合约与旧合约兼容。并且会通过原合约的历史交易测试新合约的正确性，识别潜在的安全漏洞。利用代理合约模式，确保新合约能够自动化部署，无需开发人员的手动干预，确保了区块链服务的稳定运行。
 
# 参考文献
[1]	Nakamoto S. Bitcoin: A Peer-to-peer Electronic Cash System[EB/OL]. https://bitcoin.org 
bitcoin.pdf. 2022-10
[2]	伍前红, 朱焱, 秦波等. 区块链密码学基础[M]. 北京: 科学出版社, 2024
[3]	国家密码管理局. 区块链密码应用技术要求: GM/T 0111—2021[S]. 北京: 中国标准出版社, 2021
[4]	Huang H, Xiang C, Zhong L, et al. PYLIVE: On-the-Fly Code Change for Python-based Online Services[A]. Proceedings of 40th USENIX Annual Technical Conference[C]. Online: USENIX Association, 2021: 349-363
[5]	SHOR P W. Polynomial-time Algorithms for Prime Factorization and Discrete Logarithms on a Quantum Computer[J]. Society for Industrial and Applied Mathematics Review, 1999, 41(2): 303–332
[6]	GROVER L K. A Fast Quantum Mechanical Algorithm for Database Search[A]. In: Proceedings of 28th Annual ACM Symposium on Theory of Computing[C]. Philadelphia Pennsylvania USA: ACM, 1996: 212–219
[7]	Abbasi M., Cardoso F., Vaz P., et al. A Practical Performance Benchmark of Post-Quantum Cryptography Across Heterogeneous Computing Environments[J]. Cryptography, 2025, 9:32
[8]	Zou W., Lo D., Kochhar P S, et al. Smart Contract Development: Challenges and Opportunities[J]. IEEE Transactions on Software Engineering, 2021, 47(10): 2084-2106
[9]	Ethereum vritual machine [EB/OL]. https://ethereum.org/zh/developers/docs/evm/, 2025-08-21
[10]	Ethereum percompiled contracts [EB/OL]. https://www.evm.codes/precompiled, 2025-10-23
[11]	Wang C., Li P., Fan X., et al. Phoenix: A Live Upgradable Blockchain Client[J]. IEEE Transactions on Sustainable Computing, 2023, 8: 703-714
[12]	Nave R, Palmer J F. A numeric data processor[A]. IEEE International Solid-State Circuits Conference. Digest of Technical Papers[C]. 1980: 108-109
[13]	Xu M., Guo Y., Liu C., et al. Exploring Blockchain Technology through a Modular Lens: A Survey[J]. ACM Computing Surveys, 2024, 56(9)
[14]	Androulaki E., Barger A., et al. Hyperledger Fabric: a Distributed Operating System For Permissioned Blockchains[A]. Proceedings of the 13th EuroSys Conference[C]. Porto, Portugal: ACM, 2018: Article 30
[15]	Li J. Research On Optimization Model of High Availability and Flexibility of Blockchain System Based on Microservice Architecture[J]. Procedia Computer Science, 2025, 261: 207-216
[16]	Liu J, Li P, Cheng R, et al. Parallel and Asynchronous Smart Contract Execution[J]. IEEE Transactions on Parallel and Distributed Systems, 2022, 33(5): 1097-1108
[17]	Han T., Mao J., Xie S., et al. VM-Studio: A Universal Crosschain Smart Contract Verification and Execution Scheme[J]. Security and Communication Networks, 2023, 2023(1): 2413532
[18]	Liu Z., Xiang Y., Shi J., et al. HyperService: Interoperability and Programmability Across Heterogeneous Blockchains[A]. Proceedings of 2019 ACM SIGSAC Conference on Computer and Communications Security[C]. New York, NY, USA: Association for Computing Machinery, 2019 549–566
[19]	Scaffino G., Aumayr L., Avarikioti Z., et al. Glimpse: On-Demand PoW Light Client with Constant-Size Storage for DeFi[A]. Proceedings of 2023 USENIX Annual Technical Conference[C]. Anaheim, CA, USA: USENIX Association, 2023: 733-750
[20]	Zhang Y., Zheng S., Wang H., et al. VM Matters: A Comparison of WASM VMs and EVMs in the Performance of Blockchain Smart Contracts[J]. ACM Transactions on Modeling and Performance Evaluation of Computing Systems, 2024, 9(2): 1-24
[21]	Luick D., Kolesar J., Antonopoulos T., et al. ZKSMT: A VM for Proving SMT Theorems in Zero Knowledge[A]. Proceedings of 2024 USENIX Annual Technical Conference[C]. Philadelphia, PA: USENIX Association, 2024: 3837-3845
[22]	Bruestle J., Gafni P. RISC Zero zkVM: Scalable, Transparent Arguments of RISC-V Integrity[EB/OL]. https://dev.risczero.com/proof-system-in-detail.pdf, 2023-08-11
[23]	Liu T., Zhang Z., Zhang Y., et al. Ceno: Non-uniform, Segment and Parallel Zero-Knowledge Virtual Machine[J]. Journal of Cryptology, 2017, 38:17
[24]	Chen T., Li X., Wang Y.,et al. An Adaptive Gas Cost Mechanism for Ethereum to Defend Against Under-Priced DoS Attacks[A]. Information Security Practice and Experience[C]. Cham: Springer, 2017: 3-24
[25]	Crapis D., Moallemi C., Wang S. Optimal Dynamic Fees for Blockchain Resources[A]. 28th International Conference on Financial Cryptography and Data Security[C]. Willemstad, Cham:Springer, 2024: 271-291
[26]	Chaliasos S., Swann C., Pilehchiha S.,et al. Unaligned Incentives: Pricing Attacks Against Blockchain Rollups[EB/OL]. https://arxiv.org/abs/2509.17126, 2025-9-21
[27]	Jourenko M., Larangeira M.. State Machines Across Isomoriphic Layer 2 Ledgers[A]. 27th International Conference on Financial Cryptography and Data Security[C]. Cham:Springer Nature Switzerland, 2023:75-91
[28]	Li Y., Weng J., Wu W., et al. PRI: PCH-based Privacy-preserving With Reusability and Interpoerability for Enhancing Blockchain Scalability[J]. Journal of Parallel and Distribubted Computing, 2023, 180:104721
[29]	Back S., Corallo M., Dashjr L.,et al. Enabling Blockchain Innovations with Pegged[EB/OL]. http://www.blockstream.com/sidechains.pdf, 2019-05-02
[30]	TrueBit[EB/OL]. https://whitepaper.io/document/0/truebit-whitepaper, 2017-11-16
[31]	Bearer J., Bunz B., Camacho P., et al. The Espresso Sequencing Network: HotShot Consensus, Tiramisu Data-Availability, and Builder-Exchange [EB/OL]. https://www.espressosys.com/, 2024-11-89
[32]	Polygon Zero whitepaper [EB/OL]. https://polygon.technology/papers/pol-whitepaper, 2025-09-13
[33]	Nil Foundatio [EB/OL]. https://docs.nil.foundation/, 2025-10-23
[34]	Zhang Y., Crowcroft J., Li D., et al. KylinX: A Dynamic Library Operating System for Simplified and Efficient Cloud Virtualization[A]. Proceedings of 2018 USENIX Annual Technical Conference[C]. Boston, MA: USENIX Association, 2018: 173-186
[35]	Pina L., Andronidis A., Hicks M.,et al. Mvedsua: Higher Availability Dynamic Software Updates via Multi-Version Execution[A]. Proceedings of 2019 Architectural Support for Programming Languages and Operating Systems. Providence, RI, USA, 2019: 573-585
[36]	Ajmani S., Liskov B., Shrira L.. Modular Software Upgrades for Distributed Systems[A]. Proceedings of 20th European Conference on Object-Oriented Programming[C]. Berlin, Heidelberg: Springer Berlin Heidelberg, 2006: 452-476
[37]	Baresi L., Ghezzi C., Ma X., et al. Efficient Dynamic Updates of Distributed Components Through Version Consistency[J]. IEEE Transactions on Software Engineering, 2017, 43(4): 340-358
[38]	Ciampi M., Karayannidis N., Kiayias A., et al. Updatable Blockchains[A]. Proceedings of 25th European Symposium on Research in Computer Security[C]. Guildford, UK, 2020: 590-609
[39]	Zindros D.. Soft Power: Upgrading Chain Macroeconomic Policy Through Soft Forks[A]. Proceedings of 25th Financial Cryptography and Data Security. Berlin, Heidelberg: Springer-Verlag, 2021: 467-481
[40]	Cardano hard-fork-combinator[EB/OL]. https://www.nmkr.io/glossary/cardano-hard-fork-combinator#:~:text=The, 2025-10-23
[41]	Polkadot Client Releases[EB/OL]. https://wiki.polkadot.network/docs/learn-runtime-upgrades#client-releases, 2025-08-15
[42]	Rodler M., Li W., et al. EVMPatch: Timely and Automated Patching of Ethereum Smart Contracts[A]. Proceedings of 30th USENIX Security Symposium. Online: USENIX Association, 2021: 1289-1306
[43]	Cardano Evolution[EB/OL]. https://docs.cardano.org/about-cardano/evolution/about-hard-forks, 2025-10-10
[44]	Vitalik Buterin. EIP-150: Gas Cost Changes for IO-heavy Operations [EB/OL]. https://eips.ethereum.org/EIPS/eip-150, 2016-09-24

