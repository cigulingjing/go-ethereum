# Design: benchmark-tx-gas

## Context

现有 `benchexecutionefficiency` 通过 `eth_estimateGas` + `eth_call` 测量调用成本，数值是估算而非真实交易消耗；`benchupgradelatency` 只记录升级交易 gasUsed，不覆盖 Solidity 部署与执行交易。论文升级/执行 Gas 对比图需要真实交易 receipt 的 `gasUsed` 数据。本设计新增独立命令 `benchtxgas`，复用 `benchexecutionefficiency` 的算法 fixture 定义，在单节点网络中完成测量。

## Goals / Non-Goals

Goals:

- 单节点 Clique 网络（period=0，立即出块）中测量 8 个算法的四类真实交易 gasUsed。
- 执行交易测量前用 `eth_call` 校验 WASM 与 Solidity 路径输出一致。
- 结果 JSON + 文本摘要写入 `experiments/cryptoupgrade/results/tx-gas/<timestamp>/`。

Non-Goals:

- 不测量延迟指标（已有 Lab1/Lab2 覆盖）。
- 不涉及 precompile 方案（precompile 无升级交易，且 Lab2 已覆盖其执行估算）。
- 不修改 Geth 核心、`cryptoupgrade` 运行时或既有 bench 命令行为。

## Decisions

1. **新建 `benchtxgas` 命令而非扩展 `benchexecutionefficiency`**：后者面向 `eth_call` 延迟采样（warmup+n 次），与真实交易测量流程差异大；独立命令保持最小侵入，fixture 定义代码直接复制并裁剪（去掉 precompile 字段），避免改动既有实验入口。
   - 备选：给 `benchexecutionefficiency` 加 `-real-tx` 模式。否决理由：会让该命令同时承担两套测量语义，回归风险高。
2. **执行交易用真实交易而非 estimateGas**：`callFunc` 虽为 view，但真实交易仍会执行并产生 receipt gasUsed；返回值通过前置 `eth_call` 校验。Solidity 侧调用同一合约函数的真实交易。两侧均为普通 EVM 交易定价，口径一致。
3. **WASM 上传使用 `uploadCode`（异步激活）**：与论文升级路径一致；上传后轮询 `eth_call callFunc` 直到激活完成再测执行交易。
4. **单节点配置独立文件 `local-1node.yaml`**：`nodeCount: 1`，`outputDir` 指向独立目录 `../nodes-1node`，不覆盖既有 20 节点制品；HTTP RPC 暴露 `8761`。
5. **每算法独立函数名后缀**：上传名使用 `<Name>TxGas001`，避免与节点既有版本冲突，保证重复运行结果可比。

## Risks / Trade-offs

- [单节点 period=0 出块快，交易回执几乎即时返回，gasUsed 与多节点一致（Gas 定价与网络规模无关）] → 在结果 JSON 中记录 nodeCount/chainId，论文中明确标注单节点环境。
- [Solidity 合约部署 gasUsed 依赖 solc 编译参数] → 固定 `solc 0.8.26 --optimize --via-ir --evm-version paris`，与既有实验一致，并在 JSON 中记录编译器参数。
- [Dh2048 等大 calldata 算法执行交易 gas 较高] → 交易 gas limit 统一设为 30M（链 gasLimit），失败即报错中止。

## Migration Plan

纯新增实验能力，无需迁移。验证方式：单节点网络运行 `benchtxgas`，检查 8 算法 JSON 字段完整、输出一致性校验通过。
