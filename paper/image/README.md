# 实验结果图目录

本目录按论文实验划分结果图，根目录只保留生成脚本和总索引。当前论文实验内容只设置 Lab1 和 Lab2；历史一致性图不进入正文实验内容。

重新生成命令：

```bash
python paper/image/generate_experiment_figures.py              # 全部图
python paper/image/generate_experiment_figures.py upgrade-gas  # 只生成指定图
```

可选图名：`upgrade-gas`、`upgrade-latency`、`execution-gas`、`execution-latency`。

每张图均提供 `.svg`、`.pdf`、`.png`、`.tiff` 四种格式：

- `.svg` / `.pdf`：论文排版优先使用，文本可编辑；
- `.png`：快速预览；
- `.tiff`：高分辨率提交备用。

## WASM rerun：当前正文图源

目录：`wasm-rerun/`

图片：

- `figure_wasm_evaluation.*`：当前正文使用的三面板 Evaluation 图，仅覆盖 Lab1 升级延迟、Lab2 三方案执行效率和 Gas 估算；源数据为 `figure_wasm_evaluation_source_data.csv`。
- `figure_wasm_experiments.*`：历史四面板合并图产物；其中一致性 panel 不进入论文实验内容。

主要结论：

- WASM 升级交易完成与节点本地 WASM 准备分离，当前 5 节点 rerun 的端到端可调用延迟处于约一个 5 s block interval。
- WASM-backed Upgrade 在多数算法上接近 Precompiled Contract，但 SchnorrVerify 是明确的尾部延迟样本。
- Upgrade Consistency 只在安全分析中作为协议属性讨论，不设置一致性实验，不引用 scheduled / immediate harness 结果作为正文实验 baseline。

## Lab 1：升级延迟与上传成本

目录：`lab1-upgrade-latency-availability/`

图片：

- `lab1_upgrade_latency_algorithms_20nodes.*`：20 节点全算法实验中，不同密码算法的升级时延和节点级完成离散度。
- `lab1_upgrade_latency_schnorr_scale_5to40nodes.*`：SchnorrProof/SchnorrVerify 在 5/10/20/30/40 节点下的升级时延和 WASM 编译时间。

主要结论：

- 多节点升级延迟主要由 receipt 之后的 activation 观测阶段贡献。
- 同一算法下节点完成时间聚集较紧，说明无故障局域实验环境中节点级可用时间差异较小。
- SchnorrProof/SchnorrVerify 的升级时延随节点规模增加仍保持在秒级区间。

## Lab 2：密码算法执行效率

目录：`lab2-execution-efficiency/`

图片：

- `lab2_execution_efficiency_latency.*`：多算法 `eth_call` 延迟对比。
- `lab2_execution_efficiency_latency_20nodes.*`：当前 20 节点 Lab2 三方案平均 `eth_call` 延迟分组柱状图，Y 轴为对数刻度。
- `lab2_execution_efficiency_gas_estimate.*`：多算法 Gas 估算对比。
- `lab2_real_chain_precompile_comparison.*`：SHA-256 真实链升级方案与预编译合约的 setup、调用延迟和 Gas 对比。

主要结论：

- 动态升级方案在多数密码算法上的调用延迟接近预编译合约，并明显低于 Solidity 合约实现。
- 动态升级方案的 Gas 估算整体接近预编译合约，显著低于复杂 Solidity 密码算法实现。
- SHA-256 真实链实验中，动态升级方案调用 Gas 接近预编译合约，调用延迟高于预编译合约。

## 历史图：升级过程状态一致性

目录：`lab3-state-consistency/`

图片：

- `lab3_upgrade_state_consistency_5nodes.*`：旧稿生成的升级一致性图文件，当前正文不引用。

当前处理：

- 该目录保留为历史产物，不作为当前论文实验图目录。
- 正文不得设置 Lab3 一致性实验，不得引用该图证明节点版本、输出、receipt/event、chain-view 或 State Root 一致。
- 如需讨论 Upgrade Consistency，只能在 `Security Analysis / Security Solutions` 中从协议机制和设计边界展开。
