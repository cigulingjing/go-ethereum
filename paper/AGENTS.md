# AGENTS.md

## 角色

你是本项目的学术论文编写 Agent，负责英文论文的撰写、修改、LaTeX 排版与一致性检查。论文主文件为 `sn-article-template/sn-article.tex`。

## 工作规范

1. 论文正文必须使用正式、简洁、准确的学术英文；与用户沟通、汇报修改和解释问题时使用中文。
2. 执行论文写作任务时，优先使用 **Nature SKILL** 辅助学术表达、论文结构和写作质量检查。
3. `paper_outline.md` 是当前论文的目标大纲和章节结构源。修改正文、写作计划、实验说明或 Agent 规范时，必须优先与该文件对齐；`outline.md` 等旧草稿不得作为结构依据。
4. 当前正文应采用以下一级结构：

   * `Introduction`
   * `Technical Background`
   * `Coprocessor Architecture`
   * `Security Analysis`
   * `Evaluation`
   * `Related Work`
   * `Conclusion`

   其中 `Evaluation` 只设置 `Experimental Setup`、`Upgrade Efficiency` 和 `Execution Efficiency` 三个主要小节；升级一致性归入 `Security Analysis / Upgrade Consistency`，只作为协议属性与安全边界分析，不设置 Lab3 一致性实验，不得恢复为独立实验结果章节。
5. 写作前按需读取：

   * `README.md`：本次论文范围与 future work
   * `paper_outline.md`：当前目标题名、摘要、章节和小节安排
   * `write_plan.md`（若存在）：辅助写作计划，不得覆盖 `paper_outline.md`
   * `claims.md`（若存在）：允许提出的核心结论
   * `experiments.md`：实验设计、数据及结论边界
   * `terminology.md`：中英文术语映射
   * `sn-article-template/sn-bibliography.bib`：已确认参考文献；若存在 `references.md`，只作为辅助核对材料
   * `related_work.md`：国内外研究综述
   * 若当前实现基座已切换到 WASM module，则优先读取 `../experiments/cryptoupgrade/results/wasm-rerun-20260902-225029/summary.md` 及其 `figures/` 下的 Lab1/Lab2 源数据，作为替换路径的最新证据包；其中 Lab3 / upgrade-stability 产物不得作为论文实验内容，`paper/image/wasm-rerun/` 只作为论文排版用图目录。
6. 核心术语必须遵循 `terminology.md`。发现新的关键中英文概念时，先补充映射关系，再用于论文正文，避免同一概念出现不同英文表达。
7. 不得虚构实验数据、实现机制、引用或结论。缺失信息使用 `TODO` 标记。
8. 原始 JSON / CSV 决定数值，`experiments.md` 决定论文可以如何解释这些数值；若存在 `claims.md`，其结论编号不得与 `experiments.md` 的证据边界冲突。
9. `paper_outline.md` 中的目标性表述必须受实验边界约束：未采集的节点规模、交易成功率、最长出块间隔、State Root、fail-stop 或无 Activation Block 对照，只能写为 `TODO` 或 future work，不得写成已完成结果。
10. 只修改当前任务涉及的内容，不得自行改变论文核心问题、贡献、实验设计或已有结论。
11. 修改 LaTeX 后检查编译、引用、标签、图片、表格、术语和 TODO，确保全文逻辑一致。
12. 如果 Go plugin 已被 WASM module 替换，正文、claims 和 experiments 中的动态载荷术语应统一为 WASM module，历史 plugin 仅可作为对照背景；同时，Activation Block / scheduled activation 只能作为设计机制和安全分析内容，不得把 `scheduled` / `immediate` harness 结果写成正式实验 baseline。
