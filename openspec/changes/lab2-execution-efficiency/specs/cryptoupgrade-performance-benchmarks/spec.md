## ADDED Requirements

### Requirement: 升级后执行效率基准

系统 SHALL 提供一个可执行的算法调用效率实验入口，用于比较动态升级实现、contract 实现和 precompile 实现在升级或部署完成后的调用资源消耗。

#### Scenario: 三类实现调用对比

- **WHEN** 用户运行执行效率实验入口
- **THEN** 系统 SHALL 准备动态升级、contract 和 precompile 三类调用路径
- **AND** 系统 SHALL 在同一轮实验中分别调用三类实现
- **AND** 系统 SHALL 将升级、部署或地址准备阶段标记为 setup 数据，不纳入升级后执行效率结论

#### Scenario: 多算法可比 fixture

- **WHEN** 系统构建实验算法列表
- **THEN** 系统 SHALL 只执行已有算法中 `upgrade`、`precompile` 和 `contract` 三方均可调用的算法 fixture
- **AND** 系统 SHALL 使用 fixture 中声明的 ABI 类型对输入和输出进行编码、解码与逻辑归一化
- **AND** 系统 SHALL 在三类输出不一致时使该实验失败并报告不一致的实现类型

#### Scenario: 跳过不可三方对比的算法

- **WHEN** 某个已有算法缺少三方可比实现
- **THEN** 系统 SHALL 跳过该算法
- **AND** 系统 SHALL 在机器可读结果中记录算法名和跳过原因

#### Scenario: 调用时间采集

- **WHEN** 三类调用路径已经可用
- **THEN** 系统 SHALL 通过 RPC `eth_call` 采集每类实现的调用时间
- **AND** 系统 SHALL 记录 warmup 次数、测量次数、first、mean、p50、p95、min 和 max 调用耗时
- **AND** 系统 SHALL 记录足以复现实验的输入参数和调用目标地址

#### Scenario: 调用 Gas 采集

- **WHEN** 三类调用路径已经可用
- **THEN** 系统 SHALL 使用同一逻辑输入对应的真实 calldata 采集每类实现的调用 Gas
- **AND** 系统 SHALL 将通过 `eth_estimateGas` 得到的结果明确记录为 `gasEstimate`
- **AND** 系统 MUST NOT 将 `eth_estimateGas` 结果标记为交易 receipt `gasUsed`

#### Scenario: 结果输出

- **WHEN** 实验完成执行效率对比
- **THEN** 系统 SHALL 输出机器可读 JSON 结果
- **AND** JSON SHALL 包含算法名、输入、三类实现标识、调用时间统计、Gas 估算、输出校验状态、跳过算法和比值
- **AND** 系统 SHALL 输出简明文本摘要，用于向用户反馈三类实现的资源消耗结果
