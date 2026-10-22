## Purpose

保证 Schnorr 验签实验源码在不导入 `crypto` 包时仍可独立构建和正确验签。

## ADDED Requirements

### Requirement: 挑战值一致

`SchnorrVerify` SHALL 使用原有的固定前缀、定长公钥和承诺、消息顺序计算 SHA-256 挑战值，所得结果 SHALL 与旧实现一致。

#### Scenario: 消息边界
- **WHEN** 消息为空、跨越 SHA-256 块边界或含有多个块
- **THEN** 挑战值 SHALL 与标准库基准结果一致

### Requirement: 验签结果兼容

`SchnorrVerify` SHALL 维持原来的 `bytes,bytes -> bool` ABI，并对有效证明返回 true、无效长度或篡改证明返回 false。

#### Scenario: 部署后执行
- **WHEN** 使用现有工具构建并执行独立 WASM 模块
- **THEN** 正确证明 SHALL 通过验签，篡改消息或证明 SHALL 验签失败

### Requirement: 连续调用安全

新构建的 WASM 模块 SHALL 在连续调用期间使 ABI 输入不与模块堆分配互相覆盖，并保持旧制品的执行兼容性。

#### Scenario: 连续验证不同长度消息
- **WHEN** 在同一模块实例中依次验证有效、篡改及长度变化的证明
- **THEN** 每次结果 SHALL 与独立重新实例化的结果一致
