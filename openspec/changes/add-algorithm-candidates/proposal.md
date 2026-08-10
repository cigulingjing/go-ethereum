## 为什么

cryptoupgrade 流程需要一组具有代表性的密码算法，用于编译为 Go plugin 并支撑升级与基准实验。源算法位于 `/home/liuqi/project/blockchain-crypto`，但该仓库源码必须保持不变，因此需要在 `cryptoupgrade/algorithm/go` 中提供可独立构建、适合 plugin 的 Go 候选文件。

## 变更内容

- 为每个适合单文件纯 Go 改造的 blockchain-crypto 类别增加一个独立候选 plugin 源文件。
- 保持每个候选文件为 `package main`，并暴露导出的算法入口。
- 使用 `go build -buildmode=plugin -tags=urfave_cli_no_docs,ckzg -trimpath` 验证每个候选文件。
- 为可以在不新增依赖的前提下实现的确定性或可验证候选操作增加 Solidity 合约对照实现。
- 记录被跳过的类别及原因，包括依赖外部可执行文件、Rust/C binding、大型曲线/证明栈或不适合压缩到单文件中的多包协议状态。

## 能力

### 新增能力

- `cryptoupgrade-algorithm-candidates`：覆盖 cryptoupgrade plugin 使用的独立纯 Go 候选算法源码及其构建验证。

### 修改能力

- `cryptoupgrade-performance-benchmarks`：扩展基准准备要求，要求提供可通过既有 Go plugin 路径编译的可复用候选算法集。

## 影响

- 影响代码：`cryptoupgrade/algorithm/go/*.go`、`cryptoupgrade/algorithm/contracts/*.sol`
- 影响规划制品：`openspec/changes/add-algorithm-candidates`
- 不新增依赖。
- 不修改 `/home/liuqi/project/blockchain-crypto` 源文件。
