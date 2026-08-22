## Why

当前论文已经具备项目范围、允许使用的核心结论、实验边界、术语表，以及相对完整的国内外研究综述，但仍缺少一个可执行的整体编写计划，将这些材料转化为逐章节的论文写作流程。

本 change 的目标是在修改 `sn-article-template/sn-article.tex` 之前，先基于现有证据定义论文应如何组织和撰写，从而降低结论无证据支撑、术语漂移和写作顺序混乱的风险。

## What Changes

- 为密码学协处理器论文新增一个整体编写计划。
- 定义全文的章节顺序、各章节任务、所需证据来源和结论边界。
- 将写作流程与现有源文件对齐：`README.md`、`paper_plan.md`、`claims.md`、`experiments.md`、`terminology.md`、`references.md` 和 `related_work.md`。
- 要求后续正文写作保持当前论文范围：面向智能合约的密码算法运行时升级、异步升级准备、确定性激活、执行效率，以及已评估场景下的状态一致性。
- 建立写作前和写作后的检查项，覆盖术语、证据支撑、引用、TODO、图片、表格、标签和 LaTeX 编译。
- 本 planning change 不修改实验设计，不虚构新结果，也不直接重写论文正文。

## Capabilities

### New Capabilities

- `overall-paper-plan`：定义一个可复现、受证据边界约束的论文整体编写计划所需的结构和验收标准。

### Modified Capabilities

- 无。

## Impact

- 受影响的规划文件：`write_plan.md` 以及 `openspec/changes/write-overall-plan/` 下的 OpenSpec change 文件。
- 受影响的论文写作流程：后续修改 `sn-article-template/sn-article.tex` 时，应遵循已批准的计划，并将主要论述映射到 `claims.md` 和 `experiments.md`。
- 本 proposal 不修改代码、Geth API、实验脚本、依赖项或区块链执行行为。
