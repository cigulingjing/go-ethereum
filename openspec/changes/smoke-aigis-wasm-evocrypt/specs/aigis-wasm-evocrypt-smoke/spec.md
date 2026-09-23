## ADDED Requirements

### Requirement: Aigis-sig2 compiles to a WASI execute module

系统 SHALL 将 Aigis-sig2 verify 编译为 WebAssembly 模块，该模块 MUST 导出 `execute`，MUST NOT 导入 `wasi_snapshot_preview1` 以外的 host 模块。

#### Scenario: 模块可被 wazero 实例化

- **WHEN** 使用 wasi-sdk 按 reactor 模型链接 Aigis-sig2 verify
- **THEN** 产物 SHALL 以 `\0asm` 开头
- **AND** `wasmruntime.Activate` SHALL 成功
- **AND** 模块 MUST NOT 声明 `env` 或其他非 WASI import

### Requirement: EvoCrypt upgrade path accepts the module

系统 SHALL 把该 WASM 当作 CodeStorage 升级载荷：编码、登记版本、协处理器激活。

#### Scenario: 上传后可以激活

- **WHEN** 调用方将 Aigis WASM 经 `EncodeWasm` 提交为 `uploadCodeVersion` 的 `code`
- **THEN** CodeStorage SHALL 发出 `codeVersionUploaded`
- **AND** activation 服务 SHALL 解码、编译并标记该版本 prepared

### Requirement: callFunc can verify an Aigis-sig2 signature

激活后的模块 SHALL 通过 `callFunc` 执行验签，ABI 为 `(bytes pk, bytes message, bytes signature) -> bool`。

#### Scenario: 合法签名验证成功

- **WHEN** 使用与 Aigis-sig2 空 ctx 公开 API 生成的公钥、消息和签名调用已激活模块
- **THEN** 返回的 ABI bool SHALL 为 true

#### Scenario: 篡改签名验证失败

- **WHEN** 保持公钥和消息不变但翻转签名中的一个字节
- **THEN** 返回的 ABI bool SHALL 为 false
- **AND** 调用 MUST NOT 使 wasmruntime 崩溃
