## Why

论文升级成本与执行成本对比图目前基于 `eth_estimateGas` 的估算值，缺少真实交易视角的证据。需要在单节点网络中，对 8 个算法测量 WASM 动态升级方案与 Solidity 合约方案在**升级交易**（`CodeStorage.uploadCode` vs 合约部署）和**执行交易**（`CodeStorage.callFunc` vs 合约调用）下的真实 `gasUsed`，形成可复现的链上 Gas 对比数据。

## What Changes

- 新增实验命令 `benchtxgas`：在单节点 Clique 网络中，对每个算法发送真实交易并读取 receipt 的 `gasUsed`。
- 每个算法采集四项指标：WASM 升级交易 gasUsed、Solidity 部署交易 gasUsed、WASM 执行交易 gasUsed、Solidity 执行交易 gasUsed。
- 执行交易发送前通过 `eth_call` 校验两条路径输出一致，输出不一致则该算法实验失败。
- 覆盖 8 个算法：Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit、SchnorrVerify、PolynomialMul。
- 输出机器可读 JSON 与简明文本摘要，结果写入 `experiments/cryptoupgrade/results/tx-gas/`。
- 新增单节点网络配置 `experiments/cryptoupgrade/deployments/local-1node.yaml`，与既有 20 节点配置互不干扰。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 增加真实交易 Gas 对比实验要求，覆盖单节点网络中 WASM 升级方案与 Solidity 合约方案在升级交易与执行交易下的 receipt `gasUsed` 对比。

## Impact

- Affected code: `experiments/cryptoupgrade/bench/cmd/` 下新增 `benchtxgas` 命令；`experiments/cryptoupgrade/deployments/` 下新增单节点网络 YAML。
- Affected inputs: 已有 WASM 算法文件（`experiments/cryptoupgrade/algorithm/go/wasm/*.wasm`）、Solidity 合约源码、测试输入。
- Affected artifacts: OpenSpec change、单节点实验 JSON 原始结果、文本摘要、论文实验数据表与对比图。
- No breaking changes to Geth consensus、EVM semantics、动态升级激活逻辑或既有实验命令。
