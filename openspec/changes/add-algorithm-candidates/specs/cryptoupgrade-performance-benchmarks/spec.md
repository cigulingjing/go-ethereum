## MODIFIED Requirements

### Requirement: 三路径多项式乘法对照

系统 SHALL 支持同一多项式乘法内核的三组实验对照。

#### Scenario: Solidity 纯合约实现

- **WHEN** 选择 Solidity 对照组
- **THEN** `PolynomialMul` SHALL 由 Solidity 合约实现
- **AND** 不应通过预编译合约或升级方案绕过多项式乘法主体执行

#### Scenario: 升级方案实现

- **WHEN** 选择升级方案对照组
- **THEN** `PolynomialMul` SHALL 通过 Go/TinyGo WASM 编译、上传和调用
- **AND** SHALL 由当前 Cryptographic Coprocessor 执行 WASM 模块

#### Scenario: 预编译合约实现

- **WHEN** 选择预编译合约对照组
- **THEN** `PolynomialMul` SHALL 在 geth 客户端侧以 native precompile 路径执行
- **AND** 调用接口 SHALL 与其他对照组保持语义等价

#### Scenario: 统一实验输入

- **WHEN** benchmark 比较三组实验对照
- **THEN** 三组 SHALL 使用相同的多项式长度、系数、模数和输出格式
- **AND** benchmark SHALL 在计时前校验三组输出一致
