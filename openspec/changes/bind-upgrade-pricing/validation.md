# 验证记录

## 实现范围

Schedule 的输入分档、阈值和增长系数随算法版本提案原子提交。EVM 不再读取链级 Profile。旧固定费用提案仍可执行；费用变化须使用新版本，完全相同的提案允许幂等重放。

## 已通过

```text
go test ./cryptoupgrade/pricing ./cryptoupgrade/internal/evm ./cryptoupgrade/internal/event ./cryptoupgrade/internal/repository ./cryptoupgrade/internal/activation ./params ./experiments/cryptoupgrade/core/network ./experiments/cryptoupgrade/scripts/upload-wasm ./experiments/cryptoupgrade/scripts/chapterfive/benchbilling
go test ./cryptoupgrade -run 'Test(VersionPricing|PricingProposal|UpdataGas|UploadCode|GetActiveVersion|HandleCodeUploaded|HandleCodeVersion|StageTimingWASM)' -count=1
go test ./core/vm -run 'TestCryptoUpgrade' -count=1
openspec validate bind-upgrade-pricing --strict
```

- EVM 集成测试在隔离进程中发布三份版本提案，通过真实事件加载 WASM，加法结果为 7。
- 算法 A：阈值 1、增长 1，交错调用收费为 10、20、40；算法 B：阈值 3、增长 1/2，前两次各 30。
- 不足预算不执行且不推进计数；新交易重置两个算法的计数；激活高度 50 后算法 A 使用版本 2 的 100 Gas 起价。
- 覆盖溢出、分档边界、规范编码、无效参数、只读上传、冲突提案、原地改价拒绝及固定收费兼容。
- 元数据保存/重载后，旧版本和新版本的费用均保持一致。
- Solidity 0.8.26 使用 `--optimize --via-ir --combined-json abi,bin` 编译成功，新增三个方法的 ABI 与 Go 公共 ABI 完全一致。
- 修改后的 Python 敏感性脚本通过语法解析。

## 使用与边界

上传工具使用 `-pricing experiments/cryptoupgrade/config/pbkdf2-pricing.json`，配合 `-mode version` 或 `-mode immediate`。旧实验配置的链级 pricing 必须移除；新建实验链并发布带参数的新版本。升级工具默认使用固定交易 Gas 上限，避免现有本地 metadata 仓库在估算交易时提前登记 immediate 版本。

本次验证为组件测试及进程内 EVM/WASM 集成测试，未重跑完整私链性能实验。沿用项目现有节点本地元数据架构，未扩展为通用 StateDB 回滚机制。
