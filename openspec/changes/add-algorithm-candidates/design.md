## 上下文

多项式矩阵-向量乘法是 lattice-based cryptography 中常见的核心操作。为了避免一次实验混入多个算法族、输入编码和依赖栈，本 change 将旧的候选算法集合收敛为一个可复现的多项式乘法内核。矩阵-向量乘法可以由该内核重复组合，因此先用单次多项式乘法建立 Solidity、native precompile 和 WASM 协处理器的可比基线。

## 目标 / 非目标

**目标：**

- 三种实现均提供 `PolynomialMul(uint256[] left, uint256[] right, uint256 modulus) returns (uint256[])`。
- 默认 benchmark 输入规模固定为两个长度为 16 的多项式。
- 默认模数固定为 12289。
- 输出为 Solidity ABI 编码的 `uint256[]`，长度为 `len(left)+len(right)-1`。
- 三种实现均使用朴素 `O(n*m)` 卷积算法，并在每次乘加后做模归约。
- 统一 benchmark 脚本通过同一个 RPC 节点、同一组输入和同一校验逻辑测量三条路径。

**非目标：**

- 不在本 change 中实现完整多项式矩阵-向量乘法协议。
- 不实现 NTT、Karatsuba 或其他优化算法。
- 不比较不同模数、不同多项式长度或不同算法优化策略。
- 不修改 `/home/liuqi/project/blockchain-crypto`。

## 算法定义

输入：

- `left`：长度为 `n` 的非空 `uint256[]`。
- `right`：长度为 `m` 的非空 `uint256[]`。
- `modulus`：正整数。

输出：

- `result`：长度为 `n+m-1` 的 `uint256[]`。

计算：

```text
result[k] = 0
for i in [0, n):
  for j in [0, m):
    result[i+j] = (result[i+j] + (left[i] * right[j] mod modulus)) mod modulus
```

Solidity 使用 `mulmod` 和 `addmod` 避免 `uint256` 中间乘法溢出；Go 和 WASM 使用 `math/big` 保持语义一致。

## 设计决策

- 选用 `uint256[]` ABI，而不是自定义 bytes 编码。这样 Solidity 合约、precompile 和 `CodeStorage.callFunc` 的输入输出语义一致，结果更容易核验。
- 将 WASM 源码复用 `cryptoupgrade/algorithm/go/polynomial_mul.go`，由 `cryptoupgrade/wasmtool` 注入 `execute` wrapper。这样同一份 Go 算法既可用于 TinyGo/WASM，又可作为 native precompile 的参考实现。
- native precompile 注册独立地址，避免复用动态升级入口。这样可以测量 Geth 客户端侧原生执行路径。
- 默认多项式长度 16、模数 12289。长度足以产生可观察的 `O(n^2)` 循环，又能避免 Solidity benchmark 在普通私链环境中过度耗气。
- benchmark 命令保留 `-poly-n` 和 `-poly-modulus` 参数，但每轮运行只生成一次输入并传给三种实现。

## 风险 / 权衡

- 朴素卷积不是实际高性能 lattice scheme 的最终实现。此处故意不使用 NTT，避免不同语言实现的优化差异影响路径对比。
- Solidity 的动态数组 ABI 会引入编码/解码成本。三种路径都使用同一 ABI 形状，因此该成本属于本实验定义的一部分。
- WASM 构建依赖 TinyGo。仓库保留 Go 源码和构建脚本，实际生成 `.wasm` 产物需要实验环境安装 TinyGo。
