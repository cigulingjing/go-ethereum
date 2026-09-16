## 1. 实验命令实现

- [ ] 1.1 新增 `experiments/cryptoupgrade/bench/cmd/benchtxgas/main.go`，定义 8 个算法 fixture（WASM 路径、合约源码、ABI 类型、测试输入、algoGas）。
- [ ] 1.2 实现 WASM 升级交易：压缩编码 WASM，`eth_sendTransaction` 调用 `CodeStorage.uploadCode`，等待 receipt 并记录 `gasUsed`。
- [ ] 1.3 实现 Solidity 升级交易：solc 编译合约，发送部署交易，等待 receipt 并记录 `gasUsed` 与合约地址。
- [ ] 1.4 实现执行交易测量：`eth_call` 校验两路径输出一致后，分别发送 `CodeStorage.callFunc` 与合约调用真实交易，记录 receipt `gasUsed`。
- [ ] 1.5 输出 JSON（含网络、链 ID、命令、每算法四项 gasUsed、txHash、blockNumber）与文本摘要，默认输出到 `experiments/cryptoupgrade/results/tx-gas/<timestamp>/result.json`。

## 2. 单节点网络

- [ ] 2.1 新增 `experiments/cryptoupgrade/deployments/local-1node.yaml`（nodeCount=1，独立 outputDir，暴露 8761）。
- [ ] 2.2 停止当前 20 节点 docker 网络，渲染并启动单节点，确认 RPC 可用、signer 已解锁。

## 3. 实验执行与产物

- [ ] 3.1 运行 8 算法实验，检查 JSON 字段完整与输出一致性校验通过。
- [ ] 3.2 将结果写入论文实验数据表（升级消耗Gas、执行消耗gas 两个 sheet 的真实交易数据），重新生成 `figure_upgrade_gas_comparison` 与 `figure_execution_gas_comparison`。
- [ ] 3.3 `gofmt`、聚焦构建检查、`openspec validate benchmark-tx-gas`。
