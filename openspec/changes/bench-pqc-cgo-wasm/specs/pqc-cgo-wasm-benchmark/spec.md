## ADDED Requirements

### Requirement: Solidity 不适用算法的 CGO 与 WASM 对照

系统 SHALL 对没有 Solidity 对照的后量子验签算法，在同一组向量上测量 CGO 原生 verify 与 EvoCrypt WASM 的进程内执行耗时。

#### Scenario: 合法签名两条路径都成功

- **WHEN** 对已归档算法使用其 testdata 中的 pk、message、signature 分别调用 CGO verify 与已激活 WASM 模块
- **THEN** 两条路径 SHALL 返回成功（ABI `true` 或等价布尔真）
- **AND** 实验 SHALL 在 warmup 之后记录 mean、p50、p95、min、max 毫秒

#### Scenario: 输出不一致则失败

- **WHEN** 同一向量上 CGO 与 WASM 的验签结果不同
- **THEN** 该算法的对照 SHALL 失败
- **AND** MUST NOT 写入成功耗时行

#### Scenario: 缺少 CGO 或制品时跳过

- **WHEN** 构建未启用 CGO，或 `pqcgo/<algorithm>/` 缺少原生源码、WASM 或 testdata
- **THEN** 实验 SHALL 跳过该算法并记录原因
- **AND** MUST NOT 把另一路径的结果写成该路径的耗时

### Requirement: 结果记录算法名、耗时与制品大小

系统 SHALL 为每个完成对照的算法写出机器可读记录，字段至少包括算法名、CGO 执行耗时、WASM 执行耗时、原生实现文件大小和 WASM 字节码长度。

#### Scenario: JSON 字段完整

- **WHEN** 一轮实验成功测完一个算法
- **THEN** 结果 JSON SHALL 包含 `algorithm`、`cgo` 耗时统计、`wasm` 耗时统计、`nativeBytes`、`wasmBytes`
- **AND** `nativeBytes` SHALL 为 `experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/native/` 根目录下抽出的算法 `.c`（`verify.c`）字节数，不含封装库、头文件与备份
- **AND** `wasmBytes` SHALL 为该目录中 `.wasm` 文件的字节长度

#### Scenario: 结果写入实验目录

- **WHEN** 实验正常结束
- **THEN** JSON 与文本摘要 SHALL 写入 `experiments/cryptoupgrade/results/pqc-cgo-wasm/<run-id>/`

### Requirement: 按算法名备份原生代码与 WASM

系统 SHALL 将测试使用的原生实现源码与 WASM 模块按算法名归档，供复现实验读取。

#### Scenario: 目录按算法划分

- **WHEN** 新增或更新一个对照算法
- **THEN** 原生 C 源码 SHALL 位于 `experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/native/`
- **AND** WASM 模块 SHALL 位于 `experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/`
- **AND** SHALL NOT 把不同算法的源码或字节码混放在同一算法目录

#### Scenario: 实验从归档目录读取

- **WHEN** 运行 CGO/WASM 对照
- **THEN** 构建与测量 SHALL 使用 `pqcgo/<algorithm>/` 中的原生源码、WASM 和 testdata
- **AND** MUST NOT 依赖未归档的临时构建产物作为唯一输入
