## ADDED Requirements

### Requirement: 多项式乘法内核

系统 SHALL 提供一个固定的多项式乘法内核，用于作为多项式矩阵-向量乘法实验的基础计算。

#### Scenario: 统一算法

- **WHEN** 执行 `PolynomialMul`
- **THEN** 系统 SHALL 使用朴素卷积算法计算两个输入多项式的乘积
- **AND** SHALL 在每次乘加后对给定 `modulus` 完成系数归约
- **AND** SHALL NOT 使用 NTT、Karatsuba 或其他改变执行结构的优化算法

#### Scenario: 统一输入输出

- **WHEN** 调用 `PolynomialMul`
- **THEN** 输入 SHALL 为 `uint256[] left, uint256[] right, uint256 modulus`
- **AND** 输出 SHALL 为 `uint256[] result`
- **AND** `result.length` SHALL 等于 `left.length + right.length - 1`

### Requirement: 三路径等价实现

系统 SHALL 为同一 `PolynomialMul` 操作提供 Solidity、native Go precompile 和 WASM 三种功能等价实现。

#### Scenario: Solidity 基线

- **WHEN** 选择 Solidity 对照组
- **THEN** 多项式乘法 SHALL 在 Solidity 合约中执行
- **AND** SHALL 使用 EVM `mulmod` 和 `addmod` 完成乘加与归约

#### Scenario: Native precompile

- **WHEN** 选择 precompile 对照组
- **THEN** 多项式乘法 SHALL 在 Geth native Go precompile 中执行
- **AND** precompile SHALL 使用与 Solidity 和 WASM 相同的 ABI 输入输出类型

#### Scenario: WASM 协处理器

- **WHEN** 选择 WASM 升级方案对照组
- **THEN** 多项式乘法 SHALL 由 Go/TinyGo 编译出的 WASM 模块执行
- **AND** WASM SHALL 通过当前 Cryptographic Coprocessor 的 `CodeStorage.callFunc` 路径调用

### Requirement: 可复现实验输入

系统 SHALL 在统一 benchmark 中为三路径复用同一组输入。

#### Scenario: 默认输入规模

- **WHEN** 未显式指定多项式规模
- **THEN** benchmark SHALL 使用两个长度为 16 的多项式
- **AND** SHALL 使用模数 12289

#### Scenario: 输出一致性校验

- **WHEN** benchmark 执行三路径调用
- **THEN** benchmark SHALL 比较三种实现返回的 canonical `uint256[]` 输出
- **AND** 任一路径输出不一致时 SHALL 使本轮实验失败
