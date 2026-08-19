# lab3-upgrade-stability 实验结论

## 实验配置

- 私链配置：`experiments/cryptoupgrade/deployments/networks/local-5nodes.yaml`
- 节点规模：5 个 Clique 节点，1 个 signer，4 个 observer
- 共识周期：5 秒
- 升级模式：`scheduled` 与 `immediate`
- 轮次：每种模式 2 轮，共 4 轮
- 采样数据：`result.json`、`samples.csv`、`summary.txt`

## 结果摘要

- 网络前置检查通过：5 个节点 chain ID 均为 `11223344`，peer count 均为 4，区块高度正常增长。
- 4/4 轮升级稳定性实验通过。
- 2 轮 `scheduled` 模式在 `activationBlock - 1` 仍返回旧版本输出 `200`，在 `activationBlock` 返回新版本输出 `205`。
- 2 轮 `immediate` 模式在升级交易进入链后的生效区块返回新版本输出 `205`。
- 所有节点在同一采样区块下的 block hash、receipt block hash、版本号、activationBlock、metadata hash 和算法输出均一致。

## 结论

在本次 5 节点 Clique 私有链实验中，升级交易未导致链视图分叉、receipt 不一致、事件字段不一致、版本选择不一致或算法输出不一致。

合约侧版本计划与 Geth 本地事件激活解耦后，链上 `activationBlock` 决定版本生效语义；本地 activation 只负责提前准备对应版本 artifact。实验结果支持以下判断：在无故障私有链环境中，scheduled 与 immediate 两类升级交易均能保持链一致性、版本一致性和算法输出一致性。
