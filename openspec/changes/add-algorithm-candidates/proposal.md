## 为什么

原 change 以筛选多类密码算法候选为目标，容易把性能差异混入算法复杂度、输入形状和依赖差异中。当前实验需要改为选择一个明确的密码学操作：多项式矩阵-向量乘法中的多项式乘法内核。

本轮先实现可复现实验内核：两个多项式在给定模数下做朴素卷积乘法，并对每个输出系数完成模归约。Solidity、native Go precompile 和 WASM 协处理器三条路径必须执行同一算法、同一输入规模、同一模数和同一 ABI 输出，保证比较结果反映执行路径差异，而不是算法差异。

## 变更内容

- 用 `PolynomialMul(uint256[],uint256[],uint256) returns (uint256[])` 替代旧的多算法候选集。
- 增加 Solidity 合约实现，作为 EVM 原生执行基线。
- 增加 native Go 实现，并注册为 Geth 自定义预编译合约。
- 增加可由 Go/TinyGo 编译为 WASM 的实现，通过当前 Cryptographic Coprocessor 的 `CodeStorage.callFunc` 路径执行。
- 扩展 WASM 构建 wrapper，使其支持 `uint256[]` 输入和输出。
- 增加统一测试脚本，复用同一组输入比较 Solidity、precompile 和 WASM 三种实现的 `eth_call` 执行时间。

## 能力

### 新增能力

- `cryptoupgrade-polynomial-multiplication`：提供多项式乘法内核的 Solidity、native Go precompile 和 WASM 三种等价实现。

### 修改能力

- `cryptoupgrade-performance-benchmarks`：增加单一多项式乘法内核的三路径性能对照要求。

## 影响

- 影响代码：`common/cryptoupgrade_contract.go`、`cryptoupgrade/precompile.go`、`cryptoupgrade/algorithm/go/*.go`、`cryptoupgrade/algorithm/contracts/*.sol`、`cryptoupgrade/wasmtool/*`、`experiments/cryptoupgrade/bench/*`
- 影响规划制品：`openspec/changes/add-algorithm-candidates`
- 不修改 `/home/liuqi/project/blockchain-crypto` 源文件。
