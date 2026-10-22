## Purpose

实验 WASM 算法在不导入标准库 `crypto` 的条件下保持原有算法结果和 ABI。

## ADDED Requirements

### Requirement: Pedersen 承诺兼容

`PedersenCommit` SHALL 对相同输入返回与已有实现一致的 256 字节承诺。

#### Scenario: 输入与边界
- **WHEN** 消息或盲化因子为空、超过群阶，或哈希生成器经过多个计数器候选
- **THEN** 承诺 SHALL 与原有计算结果一致

### Requirement: PBKDF2 导出兼容

`Pbkdf2Sha256` SHALL 保留迭代次数及密钥长度边界，按 PBKDF2-HMAC-SHA256 产生相同结果。

#### Scenario: 多块导出与无效参数
- **WHEN** 请求多于 32 字节密钥、空密钥，或传入越界参数
- **THEN** SHALL 返回相应标准结果、空切片或 nil

### Requirement: Ed25519 验签兼容

`Ed25519Verify` SHALL 对有效纯 Ed25519 签名返回 true，对无效长度、非规范标量或不成立的签名返回 false，并可经现有 WASM 执行入口调用。

#### Scenario: 签名与篡改
- **WHEN** 使用有效签名、被修改的消息或非规范签名
- **THEN** SHALL 分别返回 true、false 或 false

### Requirement: 独立源码构建

三种实验源码 SHALL 不导入 `crypto` 标准库，并保持 WASM 原有的部署接口。

#### Scenario: TinyGo 构建
- **WHEN** 使用项目工具分别编译三个算法
- **THEN** SHALL 获得可部署的 WASM 制品，已有制品输出语义不变
