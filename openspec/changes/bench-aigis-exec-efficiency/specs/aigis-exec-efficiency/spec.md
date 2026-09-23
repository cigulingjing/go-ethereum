## ADDED Requirements

### Requirement: Aigis-sig2 WASM 与预编译延迟对比

系统 SHALL 在同一组 Aigis-sig2 验签向量上测量 EvoCrypt WASM 调用与 native 预编译调用的墙钟延迟。

#### Scenario: 合法签名两条路径都成功

- **WHEN** 使用嵌入的 pk、message、signature 分别调用已激活 WASM `callFunc` 与 `AigisSig2Verify` 预编译
- **THEN** 两条路径 SHALL 返回 ABI `true`
- **AND** 实验 SHALL 记录 warmup 之后的 mean、p50、p95、min、max 毫秒

#### Scenario: 相对预编译的开销

- **WHEN** 两条路径都完成采样
- **THEN** 结果 SHALL 包含 `(T_wasm - T_precompile) / T_precompile`，其中 `T_*` 为 mean 延迟
- **AND** SHALL 将 JSON 写入 `experiments/cryptoupgrade/results/`

#### Scenario: 缺少 CGO 时跳过对比

- **WHEN** 构建未启用 CGO
- **THEN** 效率对比 SHALL 跳过并说明原因，MUST NOT 把 WASM 结果伪造成预编译结果
