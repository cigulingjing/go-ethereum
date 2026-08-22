## ADDED Requirements

### Requirement: Overall manuscript argument
写作计划 SHALL 定义论文的一句话核心论点，并用该论点组织全文。

#### Scenario: Argument is defined
- **WHEN** 写作计划被创建
- **THEN** 计划给出一个有边界的核心论点，覆盖 cryptographic coprocessor、contract-facing cryptographic algorithms 的运行时升级、Native 执行效率、异步准备和确定性激活。

### Requirement: Section-by-section writing workflow
写作计划 SHALL 定义预期论文章节、章节写作顺序和各章节承担的写作任务。

#### Scenario: Section workflow is available
- **WHEN** 后续写作任务修改 `sn-article-template/sn-article.tex`
- **THEN** 写作者可以从计划中识别目标章节、段落级任务和所需源文件。

### Requirement: Claim-evidence alignment
写作计划 SHALL 将主要章节映射到 `claims.md` 中允许使用的结论，以及 `experiments.md` 中定义的证据边界。

#### Scenario: Claim is used in a section
- **WHEN** 计划将某个 claim 分配给 Abstract、Introduction、Results、Discussion 或 Conclusion
- **THEN** 计划标明对应的 claim 编号，并防止写出超出证据边界的结论。

### Requirement: Related work synthesis
写作计划 SHALL 将 `related_work.md` 作为 Related Work 结构的主要来源。

#### Scenario: Related Work is drafted
- **WHEN** Related Work 章节被撰写或修改
- **THEN** 既有研究按照技术主题和局限性分组，而不是仅按作者或年份罗列。

### Requirement: Terminology consistency
写作计划 SHALL 要求论文术语遵循 `terminology.md`。

#### Scenario: Core terminology is used
- **WHEN** 计划引入或引用反复出现的技术概念
- **THEN** 计划使用 `terminology.md` 中的规范英文术语，或者先标记 TODO 以更新术语表。

### Requirement: Missing-information handling
写作计划 SHALL 要求缺少支撑的数据、机制、引用和结论使用 TODO 标记，而不是通过推断补全。

#### Scenario: Evidence is missing
- **WHEN** 计划中的句子或章节需要源文件中不存在的数据
- **THEN** 计划将缺失输入标记为 TODO，并且不允许形成完整结论。

### Requirement: Manuscript quality gates
写作计划 SHALL 定义后续 LaTeX 修改后需要执行的检查项。

#### Scenario: Manuscript text is changed
- **WHEN** 后续实现修改 `sn-article-template/sn-article.tex`
- **THEN** 写作者检查编译、引用、标签、图片、表格、术语、结论边界和剩余 TODO 标记。
