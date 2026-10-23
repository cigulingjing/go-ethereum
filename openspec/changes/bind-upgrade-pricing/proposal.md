## Why

当前动态 Gas 的 Threshold、Alpha 来自链级 Profile，升级提案只能指定固定费用，无法保证每个算法版本拥有独立计价规则。

## What Changes

- 随版本升级提交、校验、查询和持久化输入分档、调用阈值及增长系数。
- 调用根据生效版本计价，保留未携带动态参数的固定收费接口。
- **BREAKING**：拒绝旧链级计价配置及原地 updataGas 修改；费用变更必须发布新版本。
- 更新上传工具、实验配置和验证用例。

## Capabilities

### New Capabilities
- `cryptoupgrade-version-pricing`: 算法版本独立计价参数的发布与执行。

### Modified Capabilities
无。

## Impact

cryptoupgrade、CodeStorage ABI/合约、EVM 调用入口、params 兼容检查及实验工具；不增加外部依赖。

## 非目标

不重新校准价格、不实现跨区块负载定价、不改变共识或治理权限模型。
