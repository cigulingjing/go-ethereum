## Context

论文目录已经包含 LaTeX 主文档、项目范围说明、论文逻辑计划草稿、允许使用的核心结论、实验边界、术语表、参考文献笔记，以及完整的中文国内外研究综述。当前材料已经定义了论文可以提出什么结论，但尚未提供一个统一的执行计划，用于按受控顺序完成全文写作。

本设计将论文定位为算法型系统论文：论文提出密码学协处理器，解释该架构为何有效，评估升级延迟、执行效率和一致性，并明确结论边界。该计划需要支持英文论文写作，同时保留中文源材料的真实意图和项目专用术语。

## Goals / Non-Goals

**Goals:**

- 产出一个整体写作计划，用于指导后续修改 `sn-article-template/sn-article.tex`。
- 将现有源文件转换为逐章节的写作流程。
- 将论文主要章节映射到对应的 claims、experiments、图片、表格和参考文献来源。
- 通过明确的 TODO 标记暴露缺少证据支撑的论述。
- 保持 `README.md`、`claims.md` 和 `experiments.md` 中定义的论文范围。
- 将 `related_work.md` 作为 Related Work 章节结构的主要来源。

**Non-Goals:**

- 本 change 不重写完整论文正文。
- 不修改实验设计、原始数据、图片、表格或结果解释。
- 不引入新的结论、未经验证的引用或未测量的性能结论。
- 不改变 Geth、`cryptoupgrade/` 或 `experiments/cryptoupgrade/` 的行为。

## Decisions

### Decision 1: 将可执行写作计划存放在 `write_plan.md`

实现阶段应新增 `write_plan.md`，因为用户明确要求在 paper 目录下生成该文件，用于承载完整的论文整体内容编写计划。

考虑过的替代方案：直接更新已有 `paper_plan.md`。该方式与最初 change 设计一致，但不符合本次用户指定输出文件名，因此本次将 `paper_plan.md` 作为输入材料，`write_plan.md` 作为新的执行计划文件。

### Decision 2: 使用算法型系统论文的论证链

计划应遵循以下论证链：

```text
问题与范围 -> 密码学协处理器架构 -> 异步升级准备 -> 确定性激活 -> 实验证据 -> 局限性与 future work
```

这一结构比纯综述结构或一般经验研究结构更符合本文的实际贡献。

考虑过的替代方案：只围绕三个 Lab 组织论文。这样实验部分会更容易阅读，但会弱化架构贡献，并削弱设计选择与实验证据之间的联系。

### Decision 3: 维护明确的 claim-evidence-section 映射

每个论文章节都应明确其允许支撑 `claims.md` 中的哪些结论，以及可以使用 `experiments.md` 或 `related_work.md` 中的哪些证据来源。

考虑过的替代方案：让每个章节在写作时单独决定可以使用哪些 claims。这样局部写作更快，但会增加 Abstract、Introduction、Results 和 Conclusion 之间过度推断的风险。

### Decision 4: 将 Related Work 写成主题综合，而不是文献罗列

Related Work 计划应按照机制和研究问题组织文献，并使用 `related_work.md` 中的结构：区块链拓展性、区块链虚拟机优化、动态软件升级、分布式系统热更新，以及区块链升级机制。

考虑过的替代方案：按照年份或单篇论文逐条罗列文献。这样更容易起草，但不利于定位密码学协处理器的贡献。

### Decision 5: 在论文正文编辑前加入质量门

计划应要求检查术语、结论边界、原始数据来源、引用可用性、图片和表格就绪状态、TODO 标记、标签以及 LaTeX 编译。

考虑过的替代方案：只依赖最终校对。这样风险较高，因为缺少证据支撑的结论和术语漂移在正文写作前更容易被发现和修正。

## Risks / Trade-offs

- 章节规划过于刚性可能降低写作速度 -> 保持章节任务简洁，并允许在目标期刊明确后进行后续更新。
- 实验数据可能无法支撑计划中的某些句子 -> 要求使用 TODO 标记，并禁止推断数值。
- Related Work 可能包含尚未核验的文献信息 -> 正式写入论文前需要进行引用核验。
- `write_plan.md` 可能与 `claims.md` 和 `experiments.md` 出现部分内容重复 -> 将 `write_plan.md` 定位为索引和流程图，而不是替代 source-of-truth 文件。
- 目标期刊尚未确定 -> 使用偏 Nature 风格的通用结构，但不强制执行具体期刊的字数限制。

## Migration Plan

1. 使用整体写作计划新增 `write_plan.md`。
2. 保持 `claims.md`、`experiments.md`、`terminology.md` 和 `related_work.md` 作为 source-of-truth 文档。
3. 使用批准后的计划指导后续对 `sn-article-template/sn-article.tex` 的修改。
4. 如果计划后续被证明不合适，应通过新的 OpenSpec change 修订 `write_plan.md`，而不是静默改变论文范围。

回滚方式较简单，因为本 change 只涉及文档：如果计划被拒绝，删除或回退 `write_plan.md` 和对应 OpenSpec change artifacts 即可。

## Open Questions

- 最终目标期刊和 Abstract 格式是什么？
- Related Work 应作为独立章节保留，还是按 Nature-family 风格部分并入 Introduction？
- 哪些图片和表格已经足够稳定，可以作为 Results 章节的组织锚点？
- 是否需要补全 `references.md`，还是直接在 `sn-bibliography.bib` 中管理已核验引用？
