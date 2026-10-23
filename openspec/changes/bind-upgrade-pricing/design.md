## Context

技术栈为 Go/Geth、Solidity 和现有 ABI 编解码。现有算法元数据按名称和版本存储，但 Profile 属于链配置，EVM 运行时覆盖固定费用。

## Goals / Non-Goals

目标是让提案成为每个算法版本费用参数的唯一来源，历史固定费用记录仍可读取。此次不增加经济校准、父区块负载机制或治理权限。

## Decisions

1. 新增独立 Schedule：Threshold、Alpha（有理数）、Bands，输入指标固定为编码字节数。原负载因子一直为一，新的调用规则不引入未实现的负载参数。Profile 仅保留为旧实验计算工具及旧配置识别类型。
2. 增加 uploadCodeVersionWithPricing / uploadCodeImmediateWithPricing 和 getVersionPricing，保持原接口 selector 不变。动态参数以规范 ABI bytes 同一交易提交，内容为三个 uint64（阈值、增长分子、分母）和两组 uint64[]（输入上限、费用）。Go 与 Solidity 均验证完整分档、非零费用、分母范围和数量上限。
3. Metadata 以 hex 字符串保存规范 ABI bytes，避免 map 中存放可变 slice/pointer 引入共享修改。原来的 getVersionInfo 不改变返回类型，事件加载追加查询 getVersionPricing，持久化及重启自然保留该字段。
4. EVM 对所选版本的 Schedule 求值，计数按交易和算法隔离，扣费成功后递增；错误执行也算已收费调用。没有 Schedule 时使用该版本的固定 Gas。
5. 上传时拒绝同名同版本的不同内容，相同提案允许幂等重放，禁止 updataGas。旧链级配置在配置检查中拒绝，避免悄悄改变旧实验收费。ABI 新增接口应同步公共定义和 Solidity 合约。
6. 上传工具接受单算法 JSON 文件并转换为 ABI payload；计费实验从链上读取生效版本参数核验，网络配置不再安装全局价格。

## Risks / Trade-offs

- 旧实验链依赖全局 Profile → 使用更新的 genesis 和带参数升级提案重新建立实验；此变更不是已部署公链的无缝升级。
- 编解码开销 → 限制分档数量，使用整数精确计算并在溢出时拒绝执行。
- 本项目现有元数据仓库是节点本地组件 → 此次沿用现有提交/事件/恢复架构，不宣称修复其全部回滚或共识持久化问题。

## Migration Plan

删除实验配置中的链级 pricing，使用 upload-wasm 的 pricing 参数为新版本发布费用。旧固定收费版本不需重写；新版本启用后不得原地改价。回退需恢复旧客户端和对应实验链数据。
