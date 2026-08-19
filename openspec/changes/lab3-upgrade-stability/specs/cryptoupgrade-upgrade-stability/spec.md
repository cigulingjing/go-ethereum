## ADDED Requirements

### Requirement: 升级稳定性功能审计
系统 SHALL 提供升级稳定性实验实施前的功能审计，用于确认当前合约、ABI、Geth 改造和实验工具是否支持合约触发升级、链上版本管理、指定区块生效和立即切换。

#### Scenario: 审计当前升级能力
- **WHEN** 用户执行升级稳定性实验的功能审计
- **THEN** 系统 SHALL 检查 `CodeStorage` 合约、内置 ABI、EVM dispatcher、事件监听激活路径和多节点实验命令
- **AND** 系统 SHALL 输出每项目标能力的 `supported`、`missing` 或 `partial` 状态
- **AND** 系统 SHALL 为缺失能力记录对应代码位置和后续实现任务

### Requirement: 链上版本升级计划
系统 SHALL 由升级合约保存算法版本、源码 metadata 和生效区块，使所有节点能够从同一链上状态推导同一算法版本。

#### Scenario: 提交指定区块生效的升级
- **WHEN** 用户提交包含算法名、版本号、源码 metadata 和 `activationBlock` 的升级交易
- **THEN** 系统 SHALL 将该版本保存为链上可查询的升级计划
- **AND** 系统 SHALL 发出包含算法名、版本号和 `activationBlock` 的升级事件
- **AND** 系统 MUST NOT 在交易执行路径中直接以本地编译结果决定交易成功或失败

#### Scenario: 提交立即切换升级
- **WHEN** 用户提交立即切换到新算法版本的升级交易
- **THEN** 系统 SHALL 将该版本保存为链上可查询的升级计划
- **AND** 系统 SHALL 将该版本的生效高度定义为升级交易进入链后所有节点可按区块号一致判断的高度
- **AND** 系统 SHALL 发出与指定区块生效模式结构一致的升级事件

### Requirement: 按区块选择算法版本
系统 SHALL 在 `CodeStorage.callFunc` 和版本查询路径中根据当前 EVM block number 选择应生效的算法版本。

#### Scenario: 生效区块前保持旧版本
- **WHEN** 当前 EVM block number 小于新版本 `activationBlock`
- **THEN** 系统 SHALL 继续选择上一个已生效版本
- **AND** 系统 SHALL 在无已生效版本时保持算法不可调用

#### Scenario: 生效区块后选择新版本
- **WHEN** 当前 EVM block number 大于或等于新版本 `activationBlock`
- **THEN** 系统 SHALL 选择满足生效条件的最高版本或最新计划版本
- **AND** 系统 SHALL 使用该版本对应的 metadata 和本地 artifact 执行算法调用

### Requirement: 事件驱动本地版本激活
系统 SHALL 通过升级事件触发每个节点的本地版本激活，并将本地编译加载状态与链上版本计划分开处理。

#### Scenario: 节点处理升级事件
- **WHEN** 节点监听到升级事件
- **THEN** 系统 SHALL 根据事件中的算法名和版本号查询链上版本 metadata
- **AND** 系统 SHALL 在节点本地按版本保存源码、plugin artifact 和激活 metadata
- **AND** 系统 SHALL 允许在 `activationBlock` 前完成本地 artifact 准备

#### Scenario: 本地激活失败
- **WHEN** 节点处理升级事件但源码解码、编译或 plugin 加载失败
- **THEN** 系统 SHALL 记录算法名、版本号、区块号和失败原因
- **AND** 系统 MUST NOT 将该节点标记为升级完成
- **AND** 系统 MUST NOT 因单个版本激活失败改变链上升级计划

### Requirement: 多节点升级一致性验证
系统 SHALL 在私有链多节点环境中验证升级交易不会导致链视图、升级事件、版本选择或算法输出不一致。

#### Scenario: 链视图一致
- **WHEN** 升级交易被提交并达到实验等待条件
- **THEN** 系统 SHALL 对所有目标节点采集 head number、head hash、升级交易 receipt 和 receipt block hash
- **AND** 系统 SHALL 在所有目标节点的目标区块 hash 一致时标记链视图一致
- **AND** 系统 SHALL 在任一节点缺失 receipt 或 block hash 不一致时标记该轮实验失败

#### Scenario: 版本视图一致
- **WHEN** 实验在生效区块前后采样版本状态
- **THEN** 系统 SHALL 在所有目标节点查询同一算法的当前版本或计划版本
- **AND** 系统 SHALL 验证所有节点在同一 block number 下返回相同版本号、`activationBlock` 和 metadata hash
- **AND** 系统 SHALL 在任一节点版本视图不同或查询失败时标记该轮实验失败

#### Scenario: 算法输出一致
- **WHEN** 实验在生效区块前后调用同一算法 fixture
- **THEN** 系统 SHALL 对所有目标节点使用同一输入执行 `CodeStorage.callFunc`
- **AND** 系统 SHALL 验证生效区块前输出符合旧版本预期，生效区块后输出符合新版本预期
- **AND** 系统 SHALL 在任一节点输出不一致、提前切换、未切换或调用失败时记录失败维度

### Requirement: 升级稳定性实验输出
系统 SHALL 输出机器可读结果和简明摘要，用于复现实验并论证升级完整性。

#### Scenario: 生成原始结果
- **WHEN** 升级稳定性实验完成
- **THEN** 系统 SHALL 输出 JSON 原始结果
- **AND** JSON SHALL 包含实验配置快照、节点列表、算法 fixture、版本计划、交易 hash、receipt 信息、事件字段、采样 block number/hash、节点级版本视图、节点级调用输出和失败原因
- **AND** 系统 SHALL 输出按节点和采样高度展开的 CSV 结果

#### Scenario: 生成摘要
- **WHEN** 升级稳定性实验完成
- **THEN** 系统 SHALL 输出简明文本摘要
- **AND** 摘要 SHALL 报告链视图一致性、事件一致性、版本一致性、算法输出一致性、通过状态和失败节点
- **AND** 摘要 MUST NOT 将执行效率或 Gas 对比作为本实验主结论
