# 第五章实验部分修改计划

## 一、总体目标

本轮修改不增加新的实验数据，而是重组第五章已有结果的叙述和版面，使实验部分从“逐项报告数值”转为“先给出主要观察，再用代表性数据支撑，最后说明含义和适用边界”。

核心原则：

1. 图表承担完整数据展示，正文承担主要趋势、比较关系和结论边界。
2. 每个结果段落采用“主要观察 → 代表性数值 → 结果含义 → 必要限定”的顺序。
3. 不为了增加解释而补写未经实验支持的因果机制；结果含义只总结数据直接支持的比较结论。
4. 节省出的版面优先用于关键定义、设计理由和实验条件说明，不继续重复图表中的完整数值。
5. 统一算法顺序、算法简称、图例命名、颜色含义和指标名称。

## 二、第五章文字重构

### 5.1 实现与实验定义

文件位置：`paper/Springer LNCS/evocrypt.tex:257-295`

- 保留实验目标、原型组成、网络配置、编译环境和测量定义。
- 将 `network-wide upgrade duration`、``eth_call`` latency、``gasEstimate``、`gasUsed` 的定义集中在首次出现处，避免在后文反复解释。
- 明确三条执行路径的固定命名：`EvoCrypt`、`Solidity`、`Native`；后续正文和图表均使用相同名称。
- 保留 `Native` 不是固定主网 Precompiled Contract 地址，而是评估客户端内进程 Go reference implementation 的限定，避免读者误解比较对象。
- 评估算法表继续保留完整算法和实现大小；正文不再逐项复述表中全部尺寸。

### 5.2 升级效率

文件位置：`paper/Springer LNCS/evocrypt.tex:297-308`

将现有“轻量算法逐项列值、复杂算法逐项列值”的段落改为以下结构：

1. **主要观察**：EvoCrypt 与 Solidity 在轻量工作负载上的 setup latency 接近；随着算法复杂度增加，EvoCrypt 的 setup latency 相对 Solidity 更低。
2. **代表性数值**：仅保留两个支撑点，例如 Add 的 `0.60 ms` 对 `0.37 ms`，以及 SchnorrVerify 的 `2.34 ms` 对 `4.30 ms`。
3. **结果含义**：说明升级路径的比较表明，EvoCrypt 可以在不部署完整 Solidity 算法合约的情况下完成算法注册和准备；不要进一步推断未被实验直接测量的网络或治理原因。
4. **必要限定**：保留 network-wide upgrade duration 的测量范围，明确排除 network propagation、consensus delay 和 unrelated EVM traffic 的影响。

其余算法的完整数值交由图 `Figure~\ref{fig:upgrade-latency}` 展示，不在正文逐个重复。

### 5.3 调用效率

文件位置：`paper/Springer LNCS/evocrypt.tex:310-330`

将调用延迟和 Gas 结果拆成两个紧凑的结果段落，分别对应两张图。

#### 5.3.1 调用延迟

建议采用以下论证顺序：

1. **主要观察**：`Native` 在所有工作负载上最快；`EvoCrypt` 在轻量工作负载上接近 `Solidity`，在较重的密码学工作负载上低于 `Solidity`，但仍高于 `Native`。
2. **代表性数值**：保留一组轻量工作负载范围和一个重型代表案例，例如轻量工作负载中 EvoCrypt 为 `0.50--0.62 ms`，以及 SchnorrVerify 中 EvoCrypt、Solidity、Native 分别为 `3.67 ms`、`4.33 ms` 和 `1.26 ms`。
3. **结果含义**：说明调用延迟的相对表现随工作负载变化，EvoCrypt 的优势主要出现在 Solidity 指令展开较重的工作负载；不额外声称该现象由某个尚未单独验证的底层因素导致。
4. **必要限定**：明确指标为 warm-up 后重复 `eth_call` 的 mean latency，且结果来自本文评估的 private-chain prototype。

删除原文中对 Add、SHA-256、BLAKE2b、PBKDF2、PolynomialMul、DH-2048、PedersenCommit 和 SchnorrVerify 的完整逐项报数；保留完整结果在图中。

#### 5.3.2 Gas

建议采用以下论证顺序：

1. **主要观察**：Solidity 的 `gasEstimate` 随算法复杂度和 EVM 指令展开明显增长，而 EvoCrypt 使用注册的 algorithm-level gas schedule，增长关系不同。
2. **代表性数值**：保留一个高复杂度对比，例如 SchnorrVerify 的 Solidity 为 `2,301,691` gas、EvoCrypt 为 `229,437` gas；必要时再保留一个轻量工作负载的 `13--15\%` 接近范围。
3. **结果含义**：结论限定为 EvoCrypt 展示了算法级计价与 Solidity 指令级 Gas 放大的解耦机制；不把当前数值表述为经过校准的经济 Gas schedule，也不把结果扩大为普遍性的 Gas 节省保证。
4. **必要限定**：保留 `gasEstimate` 不是 receipt-level `gasUsed`，且注册 Gas 参数是实验参数而非校准后的经济最优值。

删除原文中对 Blake2bSum256、PedersenCommit、SchnorrVerify 逐项完整报数的重复说明，只保留代表性案例；完整比较由 `Figure~\ref{fig:execution-gas}` 承担。

### 5.4 复杂算法压力实验

文件位置：`paper/Springer LNCS/evocrypt.tex:332-353`

- 明确本实验的验证目标：检验 EvoCrypt 是否能够为当前难以直接部署到 EVM 的复杂密码学算法提供可部署、可替换的 contract-facing 执行路径，并同时验证 Solidity/EVM 路径在复杂算法上的灵活性与成本之间难以兼容这一问题。
- 将痛点拆成两个可观察维度：一是复杂算法能否在不修改 Geth、无需为每个算法重新设计大规模 Solidity 合约的情况下被注册和调用；二是直接展开为 EVM 指令后是否面临过高的执行成本和 Gas 成本。前者由 EvoCrypt 能否成功完成上传、准备和调用来支持，后者由复杂算法的实现边界、模块规模和与客户端原生路径的成本差异来刻画。
- 首句先给出主要观察：EvoCrypt 能执行超出常规 Solidity 实现边界的复杂算法，但相对 Precompiled Contract 存在明确延迟开销。
- 使用一个代表性范围支撑观察：EvoCrypt 相对 Precompiled Contract 的延迟为 `1.3--25.1\times`。
- 保留 Groth16/BLS12-381 等极端案例时，只选择一个用于说明长尾，不逐项复述表中所有算法数值。
- 结果含义限定为：EvoCrypt 缓解了“复杂密码算法可以实现但难以以可接受 EVM 成本部署”的兼容性痛点，扩展了可部署算法的能力边界；不声称 EvoCrypt 在执行速度上优于 Precompiled Contract，也不声称实验已经测量了所有算法的实际 Solidity 部署成本。
- 保留适用性边界：当算法已有客户端原生实现时，Precompiled Contract 仍更适合；EvoCrypt 的价值在于避免为复杂算法修改客户端并协调网络升级。

## 三、图表与图注调整

### 3.1 图形组织

- 检查 `figure_upgrade_latency_comparison.pdf`、`figure_execution_latency_comparison.pdf` 和 `figure_execution_gas_comparison.pdf` 的纵向尺寸，优先压缩过高比例和多余留白。
- 不通过降低字号、缩小坐标刻度或压缩图例来换取版面；正文和图中字号必须保持可读。
- 评估是否将执行延迟和执行 Gas 组织为同一张紧凑的双子图：
  - `(a)` invocation latency；
  - `(b)` invocation `gasEstimate`。
- 若合并后导致坐标轴、图例或算法标签难以阅读，则保留两张图，但通过调整宽高和浮动位置减少第 15 页的纵向空白。
- 升级效率图单独保留，因为它对应独立的 setup-latency 问题。

### 3.2 视觉编码统一

- 三条路径统一命名为 `EvoCrypt`、`Solidity`、`Native`；压力实验中的 `Precompiled Contract` 使用完整名称，不与 `Native` 混用。
- 所有相关图使用相同的算法顺序；若按数值排序，必须在所有图中采用同一排序规则，并在图注中说明。
- 固定每条路径的颜色和线型，避免同一路径在不同图中改变颜色含义。
- 图内指标名称、纵轴单位和正文术语保持一致：
  - 延迟使用 `Mean eth_call latency (ms)`；
  - Gas 使用 `Invocation gasEstimate`，不要写成 `gasUsed`；
  - 模块大小使用 `WASM size (B)`。

### 3.3 图注重写

图注不再使用笼统的 “comparison”，而要直接说明比较对象、指标和数值含义。例如：

- 升级图：说明比较 `EvoCrypt` upgrade 与 `Solidity` deployment 的 mean setup latency，单位为 ms。
- 调用延迟图：说明比较 `EvoCrypt`、`Solidity` 和 `Native` 路径在 warm-up 后重复 `eth_call` 中的 mean latency，单位为 ms。
- Gas 图：说明比较 `EvoCrypt` 和 `Solidity` contract invocation 的 `gasEstimate`，不是 receipt-level `gasUsed`。
- 合并双子图时，在同一图注中分别说明 `(a)` 与 `(b)` 的指标和单位。
- 若继续使用对数坐标，图注和坐标刻度必须明确说明 logarithmic scale，使读者无需依赖正文才能解释柱高关系。

## 四、浮动体与章节版面调整

### 4.1 协议算法框

- 检查 `Algorithm~\ref{alg:upgrade-protocol}` 和 `Algorithm~\ref{alg:invocation-protocol}` 的浮动位置。
- 避免协议算法框在所属小节只出现一页之后才浮动到前一页或远处，尤其修正第 3.3 节开头叙述与 Invocation Protocol 算法框分离的问题。
- 优先策略：
  1. 将算法框放在所属小节首次完整介绍之后；
  2. 必要时使用更合适的浮动选项或缩短算法框上下留白；
  3. 不为了固定页码而破坏算法与解释之间的语义顺序。
- 算法框前先给出用途和输入输出，算法框后只解释读者需要理解的设计理由，避免算法框与正文互相重复。

### 4.2 第五章图表位置

- 让升级图紧邻 5.2 首次完整说明处。
- 让执行延迟图、Gas 图或合并后的双子图紧邻 5.3 对应的首次完整解释处。
- 让复杂算法表紧邻压力实验的主要观察段落；正文先概括趋势，再引导读者查看表格。
- 检查图表是否造成正文只剩一两行的孤立段落；如有，优先通过合并相关图、调整图高或改变浮动位置解决。
- 版面节省后优先增加必要的设计理由、指标定义和实验边界，而不是恢复被删除的数字列表。

## 五、具体实施顺序

1. 先重写 `5.1--5.4` 的结果段落，删除逐算法数值复述。
2. 统一正文中的路径名称、指标名称、算法简称和比较对象。
3. 检查并修改三个实验图的源文件或生成脚本，统一排序、配色、图例和坐标信息。
4. 根据图形尺寸决定是否合并执行延迟图与 Gas 图；优先保证可读性，再考虑节省页数。
5. 修改图注，使其包含比较对象、指标、单位、测量条件和必要限定。
6. 调整算法框和图表浮动位置，重点检查第 8 页、第 14 页和第 15 页的上下文连续性。
7. 编译全文并检查页数、浮动体位置、图中文字大小、交叉引用和图表编号。
8. 对第五章进行最终审查：正文是否先给结论，代表性数据是否足够，是否仍有重复报数，是否出现未经证据支持的因果解释。

## 六、验收标准

- 每个实验结果段落都遵循“主要观察 → 代表性数值 → 结果含义 → 必要限定”。
- 正文不再逐项复述图表中已经完整展示的算法数值。
- 关键比较关系在段落首句即可读出，无需读者自行从数字归纳。
- 所有数字均能在对应图或表中定位，且正文保留的数字具有代表性。
- `gasEstimate` 与 `gasUsed` 不混用，`Native` 与 `Precompiled Contract` 不混用。
- 图表字号可读，算法顺序、颜色和图例命名一致。
- 图注能够独立说明比较对象、指标、单位和测量条件。
- 协议算法框和图表靠近其所属小节的首次完整解释位置。
- 不新增未经实验支持的因果结论，不扩大实验结论的适用范围。
- 编译无 LaTeX 错误，交叉引用和浮动体位置稳定。

## 七、暂不在本轮处理的内容

- 不新增算法、节点数量、硬件配置或实验重复次数等数据。
- 不把当前实验参数重新解释为经过经济校准的 Gas schedule。
- 不将 `Native` 基线改写成主网 Precompiled Contract 基线。
- 不将压力实验结果扩展为对所有复杂密码学算法的普遍性能保证。
- 不将“灵活性和成本无法兼容”写成对所有 EVM 算法部署的普遍定理；正文应将其表述为本文选定复杂工作负载所体现的工程痛点，并区分“可调用性证据”和“成本证据”。
- 不在第五章加入新的安全证明；安全性论证继续放在第四章。
