## 1. 源文件复核

- [x] 1.1 修改写作计划前，重新读取 `README.md`、`paper_plan.md`、`claims.md`、`experiments.md`、`terminology.md`、`references.md` 和 `related_work.md`。
- [x] 1.2 确认 `terminology.md` 包含计划所需的所有反复出现的核心术语；使用前先补充缺失的规范术语。
- [x] 1.3 识别空白或不完整的源文件，尤其是 `references.md`，并将其对后续写作的影响标记为 TODO。

## 2. 整体论点与结构

- [x] 2.1 在 `write_plan.md` 中写入全文有边界的一句话核心论点。
- [x] 2.2 定义 Title、Abstract、Introduction、Related Work、Method、Experiments/Results、Discussion 和 Conclusion 的章节顺序与章节任务。
- [x] 2.3 指定推荐写作顺序，明确 Abstract 和 Title 应在主要证据章节稳定后再撰写。
- [x] 2.4 为每个主要章节添加段落级 outline，但不撰写最终论文正文。

## 3. Claim、证据与 Related Work 映射

- [x] 3.1 将每个主要章节映射到 `claims.md` 中允许使用的 claim 编号。
- [x] 3.2 将每个 Results 或 Experiments 小节映射到 `experiments.md` 中的权威数据路径和允许结论。
- [x] 3.3 基于 `related_work.md` 定义 Related Work 的综合结构，并按主题和技术局限分组。
- [x] 3.4 对缺少支撑的结论、缺失引用、缺失数值或尚未最终确定的图片/表格标记 TODO，而不是自行补全。

## 4. 质量门

- [x] 4.1 增加写作前检查，覆盖术语、claim 边界、原始数据来源、引用就绪状态，以及图片/表格就绪状态。
- [x] 4.2 增加编辑后检查，覆盖 LaTeX 编译、引用、标签、图片、表格、术语一致性和剩余 TODO 标记。
- [x] 4.3 验证生成的 `write_plan.md` 满足 `specs/overall-paper-plan/spec.md` 中的所有要求。
- [x] 4.4 运行 `openspec validate write-overall-plan`，并修复所有 validation errors。
