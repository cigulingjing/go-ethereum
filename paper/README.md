# paper

论文写作目录。实现与实验代码分别在 `cryptoupgrade/`、`experiments/cryptoupgrade/`；本目录只负责文稿、图表和写作约束。

## 本次论文范围

本次可以写：**已实现异步升级架构**。依据是实现（`uploadCode` 只做链上登记与事件，编译加载由 event/activation 异步完成）以及 Lab1 的阶段拆分（升级交易先获得 receipt，之后才观测到各节点 `callFunc` 成功）。这证明编译加载不在该升级交易的 EVM 执行关键路径上。

本次**不写**、也**不补做**：升级准备窗口内，编译/加载对其他 EVM 交易延迟或吞吐的损耗。该问题与 Lab1 同类，但需要在 `T_receipt`–`T_all_complete` 之间持续打无关交易，现有数据没有这一项。

## Future work

- **升级阶段对 EVM 的损耗**：在升级交易 receipt 之后、算法可调用之前，测量普通转账或其他合约调用的延迟与吞吐，并与无升级基线对比。用于论证异步激活是否扰动正在进行的 EVM 执行，而不是只论证升级交易本身不等待编译。
- 未就绪节点 / fail-stop、无 Activation Block 对照、State Root 采集：见 `experiments.md` Lab3 未做项。不作为本次正文结果。

## 目录

```shell
paper/
├── AGENTS.md
├── README.md
├── paper_plan.md # 全文具体逻辑
├── claims.md # 论文核心结论，并将结论进行编号
├── terminology.md # 术语表，限制英语表达问题
├── experiments.md # 论文数据源
├── references.md # 引用文档
├── related_work.md # 国内外研究综述
├── tables/ # 表格目录
├── image/ # 图片目录
├── sn-article-template
│   ├── bst
│   │   ├── sn-apacite.bst
│   │   ├── sn-aps.bst
│   │   ├── sn-basic.bst
│   │   ├── sn-chicago.bst
│   │   ├── sn-mathphys-ay.bst
│   │   ├── sn-mathphys-num.bst
│   │   ├── sn-nature.bst
│   │   ├── sn-vancouver-ay.bst
│   │   └── sn-vancouver-num.bst
│   ├── empty.eps
│   ├── fig.eps
│   ├── sn-article.pdf
│   ├── sn-article.tex  # 核心主文档
│   ├── sn-bibliography.bib
│   ├── sn-jnl.cls
│   └── user-manual.pdf
```

