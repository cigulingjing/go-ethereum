# cryptoupgrade 预编译算法规范

## Purpose

定义将 cryptoupgrade 候选算法作为 geth native precompile 暴露给 EVM 调用时的注册、ABI、gas、结果一致性和状态隔离要求。

## Requirements

### Requirement: cryptoupgrade 预编译注册

系统 SHALL 将 cryptoupgrade 算法 precompile 注册到当前激活的 geth precompile 集合中，并且不删除或替换已有预编译合约。

#### Scenario: 激活的 precompile 映射包含 cryptoupgrade 算法

- **WHEN** EVM 根据当前 chain rules 构建激活的预编译合约
- **THEN** 返回的 precompile map SHALL 包含所有已注册的 cryptoupgrade 算法地址
- **AND** 返回的 precompile map SHALL 仍包含已有 Ethereum precompile 和 cryptoupgrade 系统 precompile

#### Scenario: 地址列表包含已注册算法

- **WHEN** 请求当前 chain rules 下的激活 precompile 地址列表
- **THEN** 返回的地址列表 SHALL 包含所有已注册的 cryptoupgrade 算法地址
- **AND** 已注册 cryptoupgrade 算法地址 SHALL 不与其他激活 precompile 地址冲突

### Requirement: 确定性算法选择

系统 SHALL 仅将确定性且有界的 `cryptoupgrade` 算法入口暴露为 native precompile。

#### Scenario: 注册确定性入口

- **WHEN** 一个 `cryptoupgrade` 算法入口对 ABI 等价输入具有确定性输出，并且具有确定性 gas 规则
- **THEN** 该实现 SHALL 可被注册为 cryptoupgrade precompile

#### Scenario: 排除随机入口

- **WHEN** 一个 `cryptoupgrade` 算法入口依赖系统随机数、文件系统状态、本地编译器状态或运行时 plugin 加载
- **THEN** 实现 SHALL NOT 将该入口注册为 native precompile

#### Scenario: 拒绝随机 fallback

- **WHEN** 一个原本确定性的算法函数收到会触发随机行为的输入形式
- **THEN** precompile SHALL 拒绝该输入，而不是执行随机路径

### Requirement: ABI 调用语义

系统 SHALL 为每个 cryptoupgrade 算法 precompile 使用稳定 ABI。

#### Scenario: precompile 调用成功

- **WHEN** 调用方向已注册的 cryptoupgrade 算法 precompile 地址发送 ABI 编码参数
- **THEN** precompile SHALL 使用该算法配置的输入 ABI 类型解码输入
- **AND** SHALL 执行选定的 native 算法入口
- **AND** SHALL 使用该算法配置的输出 ABI 类型返回 ABI 编码输出

#### Scenario: 无效 ABI 输入

- **WHEN** 调用方发送的输入无法使用该 precompile 配置的输入 ABI 类型解码
- **THEN** precompile SHALL 返回执行错误
- **AND** SHALL NOT 使用部分解码的参数调用算法 handler

#### Scenario: 不需要方法选择器

- **WHEN** 调用方调用已注册的 cryptoupgrade 算法 precompile
- **THEN** 输入 SHALL 被解释为该地址对应算法的参数
- **AND** 调用方 SHALL NOT 需要包含函数选择器或 `CodeStorage.callFunc` 包装

### Requirement: Gas 计费

系统 SHALL 在执行算法 handler 前，为每个 cryptoupgrade 算法 precompile 收取确定性 gas。

#### Scenario: gas 充足

- **WHEN** 调用方以不少于 required gas 的 gas 调用已注册 cryptoupgrade 算法 precompile
- **THEN** EVM SHALL 通过既有 precompile gas 路径扣减 precompile required gas
- **AND** SHALL 执行算法 handler
- **AND** SHALL 返回调用后的剩余 gas budget

#### Scenario: gas 不足

- **WHEN** 调用方以低于 required gas 的 gas 调用已注册 cryptoupgrade 算法 precompile
- **THEN** EVM SHALL 通过既有 precompile gas 路径返回 out-of-gas
- **AND** SHALL NOT 执行算法 handler

#### Scenario: gas 敏感输入格式错误

- **WHEN** 调用方向 gas 规则依赖解码参数的算法发送格式错误输入
- **THEN** required gas 计算 SHALL 保持确定性
- **AND** 随后的 run SHALL 返回输入解码错误

### Requirement: 算法结果等价

系统 SHALL 对受支持输入返回与选定 cryptoupgrade 算法实现等价的结果。

#### Scenario: fixture 输出匹配

- **WHEN** 使用受支持的测试 fixture 输入调用已注册 cryptoupgrade 算法 precompile
- **THEN** 返回 bytes SHALL 匹配对应 native cryptoupgrade 算法实现产生的 ABI 编码结果

#### Scenario: bool 输出匹配 ABI 编码

- **WHEN** 已注册验证算法返回 bool 结果
- **THEN** precompile SHALL 使用标准 ABI 编码返回该 bool

### Requirement: 状态隔离

系统 SHALL 保持 cryptoupgrade 算法 precompile 执行独立于动态代码上传状态。

#### Scenario: 纯算法 static call 成功

- **WHEN** 调用方通过 `STATICCALL` 调用已注册 cryptoupgrade 算法 precompile
- **THEN** 对相同输入和 gas，该调用 SHALL 与普通调用产生相同结果

#### Scenario: CodeStorage 状态不变

- **WHEN** 调用方调用已注册 cryptoupgrade 算法 precompile
- **THEN** 该调用 SHALL NOT 上传代码、修改算法元数据、写入日志或依赖此前上传的 `CodeStorage` plugin 状态
