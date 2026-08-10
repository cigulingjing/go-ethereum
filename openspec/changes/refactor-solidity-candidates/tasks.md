## 1. Solidity 合约拆分

- [x] 1.1 将每个实际可行的 `cryptoupgrade/algorithm/go/*.go` 候选映射到可部署 Solidity contract name。
- [x] 1.2 将单体 Solidity 候选合约拆分为按算法部署的合约。
- [x] 1.3 将共享 2048-bit 算术和 `modexp` helper 移入非部署目标 helper 代码。
- [x] 1.4 保留或增加无法形成实际可行 Solidity 对照的候选跳过文档。

## 2. Benchmark 集成

- [x] 2.1 更新 Solidity 编译选择逻辑，使其接受或解析目标 contract name。
- [x] 2.2 更新 benchmark 输入，使每个算法可以指定 Solidity source、contract、function 和 ABI arguments。
- [x] 2.3 确保按选定 Solidity 算法合约采集部署指标。
- [x] 2.4 确保按选定 Solidity 算法函数采集部署后调用耗时和调用 gas。

## 3. 验证

- [x] 3.1 使用 `solc-0.8.26 --optimize --via-ir` 编译全部 Solidity 候选合约。
- [x] 3.2 对修改的 benchmark tooling 运行聚焦 Go 测试或构建检查。
- [x] 3.3 验证 OpenSpec 制品。
