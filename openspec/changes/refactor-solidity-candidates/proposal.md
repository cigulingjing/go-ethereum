## 为什么

基准流程需要按算法拆分的 Solidity 合约，才能独立测量每个算法对应的部署 gas、部署耗时、部署后调用耗时和调用 gas，并与匹配的 `cryptoupgrade/algorithm/go/*.go` plugin 候选对照。当前合并式 Solidity 候选合约会把无关代码一起部署，导致单算法部署指标被噪声污染。

## 变更内容

- 将旧的合并式 Solidity 候选合约拆分为多个 Solidity 合约，与 `cryptoupgrade/algorithm/go` 下的候选 Go 文件对齐。
- 保持每个 Solidity 合约都可以独立编译和部署，用于 benchmark 运行。
- 仅在能够避免重复底层代码且不会强制所有算法进入同一部署合约时，保留共享 helper/library 边界。
- 记录无法表示为实际可行 Solidity 合约的算法，例如安全随机数、AES-CBC、Ed25519 和 521-bit field 上的 Shamir。
- 为 benchmark tooling 明确按 Solidity 合约分别测量部署和部署后调用的预期。

## 能力

### 新增能力

- `cryptoupgrade-solidity-candidate-contracts`：覆盖与 `cryptoupgrade/algorithm/go/*.go` 候选对应的按算法拆分 Solidity 合约源码。

### 修改能力

- `cryptoupgrade-performance-benchmarks`：基准要求 SHALL 支持按算法统计 Solidity 部署和部署后调用指标。

## 影响

- 影响代码：`cryptoupgrade/algorithm/contracts/*.sol`
- 影响 benchmark tooling 预期：按算法选择 Solidity 合约、部署和调用测量。
- 影响规划制品：`openspec/changes/refactor-solidity-candidates`
- 不新增依赖。
- 不修改 `/home/liuqi/project/blockchain-crypto` 源文件。
