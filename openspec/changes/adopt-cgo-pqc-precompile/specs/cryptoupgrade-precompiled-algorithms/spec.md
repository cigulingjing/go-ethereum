## ADDED Requirements

### Requirement: Precompile path MUST NOT use WASM

cryptoupgrade 预编译算法入口 SHALL 在 Geth 进程内以 native 方式执行。实现可以使用纯 Go 或 CGO 调用原生库，MUST NOT 调用 `wasmruntime`、wazero 或任何 WASM 模块。

#### Scenario: 预编译执行不经过 WASM runtime

- **WHEN** 调用方对已注册的 cryptoupgrade 算法 precompile 地址发出 CALL 或 STATICCALL
- **THEN** 执行路径 SHALL 进入 native handler
- **AND** 该调用 MUST NOT 读取 WASM 制品目录或调用 `wasmruntime.Execute`

#### Scenario: 后量子预编译不得回退到 EvoCrypt

- **WHEN** CGO 不可用或原生库缺失
- **THEN** 后量子 precompile SHALL 返回错误或不注册该地址
- **AND** MUST NOT 改走 `CodeStorage.callFunc` 或 WASM 模块

### Requirement: CGO native libraries are allowed for precompile

确定性且有界的 cryptoupgrade 算法入口 MAY 通过构建期 CGO 链接原生 `.a`/`.so` 实现。该依赖 MUST 在 geth 构建时解析，MUST NOT 依赖运行时 Go plugin 加载。

#### Scenario: 允许 CGO 链接的确定性验签

- **WHEN** 一个验签入口对相同 ABI 输入给出确定性输出，并且通过 CGO 调用原生 PQMagic
- **THEN** 该实现 SHALL 可被注册为 cryptoupgrade precompile

#### Scenario: 继续排除运行时 plugin 加载

- **WHEN** 一个算法入口需要在交易执行时 `plugin.Open` 或按路径 dlopen 未在构建期绑定的模块
- **THEN** 实现 SHALL NOT 将该入口注册为 native precompile

### Requirement: ML-DSA-65 verify precompile

系统 SHALL 将 ML-DSA-65 验签注册为独立的 cryptoupgrade native precompile，地址与现有 0x47–0x5a 算法不冲突。

#### Scenario: 合法签名验证成功

- **WHEN** 调用方按 ABI `(bytes pk, bytes message, bytes signature)` 提交 ML-DSA-65 合法公钥、消息和签名
- **THEN** precompile SHALL 返回 ABI 编码的 `true`

#### Scenario: 非法签名验证失败

- **WHEN** 调用方提交格式正确但签名与消息或公钥不匹配的输入
- **THEN** precompile SHALL 返回 ABI 编码的 `false`
- **AND** MUST NOT 回退到其他算法实现

#### Scenario: 输入无法解码或长度非法

- **WHEN** 输入不能按 `(bytes, bytes, bytes)` 解码，或解码后长度不符合 ML-DSA-65 公钥/签名常量
- **THEN** precompile SHALL 返回执行错误
- **AND** SHALL NOT 调用 PQMagic
