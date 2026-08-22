# 实验结果图目录

本目录按论文实验划分结果图，根目录只保留生成脚本和总索引，具体图片放入三个 lab 子目录。

重新生成命令：

```bash
python paper/image/generate_experiment_figures.py
```

每张图均提供 `.svg`、`.pdf`、`.png`、`.tiff` 四种格式：

- `.svg` / `.pdf`：论文排版优先使用，文本可编辑；
- `.png`：快速预览；
- `.tiff`：高分辨率提交备用。

## Lab 1：升级延迟与可用性

目录：`lab1-upgrade-latency-availability/`

图片：

- `lab1_upgrade_latency_5nodes.*`：5 节点 Clique 网络中，不同密码算法的端到端升级延迟和节点级完成时间。
- `lab1_deployment_upgrade_cost.*`：Add 与 Blake2b-256 的动态升级上传成本和 Solidity 合约部署成本对比。

主要结论：

- 多节点升级延迟主要由 receipt 之后的 activation 观测阶段贡献。
- 同一算法下 5 个节点的完成时间聚集较紧，说明无故障局域实验环境中节点级可用时间差异较小。
- 动态升级上传耗时高于 Solidity 合约部署耗时，但 Blake2b-256 场景中上传 Gas 明显低于 Solidity 合约部署 Gas。

## Lab 2：密码算法执行效率

目录：`lab2-execution-efficiency/`

图片：

- `lab2_execution_efficiency_latency.*`：多算法 `eth_call` 延迟对比。
- `lab2_execution_efficiency_gas_estimate.*`：多算法 Gas 估算对比。
- `lab2_real_chain_precompile_comparison.*`：SHA-256 真实链升级方案与预编译合约的 setup、调用延迟和 Gas 对比。

主要结论：

- 动态升级方案在多数密码算法上的调用延迟接近预编译合约，并明显低于 Solidity 合约实现。
- 动态升级方案的 Gas 估算整体接近预编译合约，显著低于复杂 Solidity 密码算法实现。
- SHA-256 真实链实验中，动态升级方案调用 Gas 接近预编译合约，调用延迟高于预编译合约。

## Lab 3：升级过程状态一致性

目录：`lab3-state-consistency/`

图片：

- `lab3_upgrade_state_consistency_5nodes.*`：scheduled / immediate 升级模式下，各节点版本输出、状态一致性检查和升级交易 Gas。

主要结论：

- scheduled 模式在 activation block 前保持旧版本输出，在 activation block 返回新版本输出。
- immediate 模式在升级交易生效区块返回新版本输出。
- 4 轮实验、5 个节点的 chain view、receipt、event、version 和 output 一致性检查全部通过。

