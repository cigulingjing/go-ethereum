# 探索：cryptoupgrade 基准套件

## 背景问题

升级方案理论上在 geth 客户端侧执行 Go 代码，应该比 Solidity 在 EVM 中逐步执行密码算法更快。但基准实验必须避免把对照组混淆：

- Add 的传统路径不是 Solidity，而是手写最小 EVM 字节码。
- 早期 Blake2b 传统路径实际使用 BLAKE2F 预编译合约适配，不是 Solidity 纯算法实现。

因此实验需要拆成三组，才能解释性能来源。

## 对照关系

```text
同一算法语义：sum256(bytes) -> bytes32

┌─────────────────────┐
│ Solidity 纯合约实现 │  在 EVM 中执行算法主体
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ 升级方案 Go plugin  │  CodeStorage -> ABI -> plugin -> Go blake2b.Sum256
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ 预编译合约适配实现  │  EVM adapter -> BLAKE2F precompile
└─────────────────────┘
```

## 关键判断

- Solidity 纯合约实现用于展示 EVM 内执行密码算法的成本。
- 升级方案用于展示动态上传后在 geth 内调用 Go 算法的成本。
- 预编译合约适配用于展示 geth native precompile 路径的上界或近似上界。

## 风险与约束

- `eth_estimateGas` 对 `uploadCode` 会触发执行副作用，因此部署测试不能先走估算路径。
- Go plugin 要求编译参数、Go 版本、module path 与 geth 主进程兼容。
- 调用耗时包含 RPC、EVM 调度、ABI 编解码、plugin lookup、reflect call 等固定开销，不是纯算法耗时。
- Solidity Blake2b 为控制复杂度，当前只覆盖单块输入，即不超过 128 字节。

## 探索结论

实验应分别报告：

- 部署阶段：时间和 gas。
- 已部署调用阶段：时间，后续补齐调用 gas。
- 结果一致性：三组 Blake2b 输出必须相同。
