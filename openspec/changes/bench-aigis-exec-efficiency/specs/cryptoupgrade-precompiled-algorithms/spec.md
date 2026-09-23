## ADDED Requirements

### Requirement: Aigis-sig2 native 预编译

系统 SHALL 将 Aigis-sig2 verify 注册为确定性 native precompile，实现可使用 CGO，MUST NOT 调用 WASM runtime。

#### Scenario: 地址与 ABI

- **WHEN** 查询 cryptoupgrade 预编译集合
- **THEN** SHALL 包含地址 `0x5b`、名称 `AigisSig2Verify`
- **AND** ABI SHALL 为 `(bytes,bytes,bytes) → bool`

#### Scenario: 合法与篡改签名

- **WHEN** 预编译收到与 EvoCrypt 试点相同的合法向量
- **THEN** 输出 SHALL 为 `true`
- **WHEN** 签名首字节被翻转
- **THEN** 输出 SHALL 为 `false`

#### Scenario: 禁止 WASM 回退

- **WHEN** CGO 不可用或 native verify 失败
- **THEN** 预编译 SHALL 返回错误
- **AND** SHALL NOT 改去执行 Aigis WASM 模块
