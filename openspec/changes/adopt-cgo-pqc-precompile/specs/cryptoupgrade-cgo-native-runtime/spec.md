## ADDED Requirements

### Requirement: Native PQMagic artifacts are ELF shared or static libraries

系统 SHALL 为 cryptoupgrade 预编译路径提供用 gcc 或 clang 编译的 PQMagic-std 原生库。该库 MUST 是 ELF 目标，MUST NOT 是 WebAssembly 模块或 Emscripten 目标文件。

#### Scenario: 原生构建产物可被 file 识别为 ELF

- **WHEN** 执行 PQMagic 原生构建并安装到 native prefix
- **THEN** 产物中的 `.a` 或 `.so` SHALL 包含 ELF 对象
- **AND** 产物 MUST NOT 被识别为 `WebAssembly (wasm) binary module`

#### Scenario: wasm 构建目录保持隔离

- **WHEN** 原生构建写入 `native-build` 或 `native-prefix`
- **THEN** 系统 MUST NOT 覆盖现有 Emscripten `libs/` wasm 目标文件

### Requirement: CGO binds only native symbols at Geth build time

cryptoupgrade 预编译的后量子实现 SHALL 通过 CGO 在 geth 链接期绑定原生 PQMagic 符号，而不是在交易执行时 `plugin.Open` 或加载 WASM。

#### Scenario: CGO 启用时可以调用 ML-DSA-65 verify

- **WHEN** `CGO_ENABLED=1` 且原生库可用
- **THEN** Go 封装 SHALL 能调用 `pqmagic_ml_dsa_65_std_verify_internal`
- **AND** 合法 `(pk, message, signature)` SHALL 返回与 PQMagic 一致的成功或失败结果

#### Scenario: 禁止把 wasm 静态库当作 CGO 输入

- **WHEN** CGO 链接路径指向 Emscripten/wasm 目标文件
- **THEN** 构建 SHALL 失败
- **AND** 系统 MUST NOT 生成可运行的 PQC precompile handler

### Requirement: Verify does not consume host randomness

预编译使用的后量子验签 SHALL 是确定性的，MUST NOT 读取 `/dev/urandom`、`getrandom` 或其他宿主随机源。

#### Scenario: 相同输入两次验签结果一致

- **WHEN** 以相同公钥、消息和签名连续两次调用 CGO verify
- **THEN** 两次返回值 SHALL 相同
- **AND** 调用 MUST NOT 依赖进程级随机状态

#### Scenario: 长度非法时拒绝进入 C 层

- **WHEN** 公钥长度不是 ML-DSA-65 公钥长度，或签名长度不是 ML-DSA-65 签名长度
- **THEN** 封装 SHALL 返回错误
- **AND** MUST NOT 调用 PQMagic C 函数
