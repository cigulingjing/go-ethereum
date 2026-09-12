## Context

See proposal.md for motivation. 现有 Lab1 `benchupgradelatency` 已能在多节点网络中观测本文方案的升级交易、各节点 receipt 和 `CodeStorage.callFunc` 完成时间；现有 Lab2 `benchexecutionefficiency` 已定义 7 个算法的 Solidity 合约源码、ABI 和测试输入。缺口是把 Solidity 合约部署也按 Lab1 的 20 节点口径采集。

技术栈：Go、ethereum/go-ethereum RPC client、go-ethereum ABI package、solc、现有 `experiments/cryptoupgrade/network` YAML 网络配置。

## Goals / Non-Goals

**Goals:**

- 新增独立命令连接已经启动的多节点网络，按算法部署 Solidity 合约。
- 记录部署交易从提交到所有目标节点 receipt 可见并完成一次合约校验调用的控制端观测耗时。
- 使用 receipt `GasUsed` 作为 Solidity 组升级 Gas。
- 输出字段尽量对齐 Lab1 WASM 的 `rounds.csv` 和 `nodes.csv`。

**Non-Goals:**

- 不修改 Geth 运行时、合约源码或动态升级机制。
- 不实现代理合约升级模式；Solidity 传统方式用“部署新合约”表示算法升级成本。
- 不把部署前的 solc 编译耗时计入链上升级耗时，编译输入和字节码信息只作为可复现元数据。

## Decisions

1. 新增独立命令 `experiments/cryptoupgrade/bench/cmd/benchsolidityupgradelatency`

   独立入口避免改变现有 Lab1 WASM 结果格式和 Lab2 调用效率逻辑。命令复用多节点 YAML、sender/targets 选择、preflight 和 RPC 轮询口径。

   Alternative considered: 扩展 `benchexecutionefficiency`。该命令偏向已部署后的调用效率，只有单 RPC 视角，不能产生 20 节点全网完成时间。

2. Solidity 完成判据为 receipt 成功后合约校验调用成功

   对传统 Solidity 方案，部署交易被所有节点看到后合约代码已进入链状态；再执行一次等价函数调用能排除部署失败、ABI 错配或错误合约地址。`submit_to_all_complete_ms` 因此可直接与本文方案的全节点可调用完成时间对齐。

   Alternative considered: 只等待 receipt。该口径较弱，无法证明 deployed contract 在目标节点实际可调用。

3. 编译不计入升级时间

   实验主指标从部署交易 `eth_sendTransaction` 开始计时，与本文方案从 `uploadCode` 交易提交开始计时一致。solc 编译发生在本地实验控制端，属于准备阶段，不计入链上升级耗时。

   Alternative considered: 把 solc 编译加入耗时。该口径会把链外准备成本和链上升级成本混在一起，不利于和 `CodeStorage.uploadCode` 的交易耗时比较。

## Risks / Trade-offs

- [Risk] 轮询间隔和 RPC 往返会放大控制端观测耗时。→ Mitigation: 输出 poll interval、timeout，并沿用 Lab1 已注明的控制端观测口径。
- [Risk] `solc` 版本不同会影响合约字节码和部署 Gas。→ Mitigation: 结果记录 solc 路径、evm version、合约源码路径、bytecode bytes 和 deployment calldata hash。
- [Risk] 大型 Solidity 合约部署可能超过网络 gas limit。→ Mitigation: 提供 `-tx-gas` 覆盖参数，默认让节点估算 gas；失败时保留 receipt/error。

## Migration Plan

1. 新增命令和 focused tests，不影响现有实验入口。
2. 使用当前 20 节点网络配置运行 `-algorithms all -rounds 1` 生成 Solidity 原始数据。
3. 后续论文出图脚本可读取该命令输出，与现有 WASM `lab1-20nodes-all/bench/rounds.csv` 按算法合并。
