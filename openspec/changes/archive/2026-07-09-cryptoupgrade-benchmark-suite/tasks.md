# Tasks: Cryptoupgrade Benchmark Suite

## Proposal

- [x] 明确实验目标：部署资源与已部署调用资源。
- [x] 明确三组密码算法对照：Solidity、升级方案、预编译合约。
- [x] 定义非目标和实验边界。

## Exploration

- [x] 梳理 Add 和 Blake2b 早期实验中的传统对照含义。
- [x] 识别 Blake2b 需要拆分 Solidity 纯实现和预编译适配实现。
- [x] 识别 `eth_estimateGas` 对 `uploadCode` 的副作用风险。

## Implementation

- [x] 新增 Add benchmark 命令 `cryptoupgrade/cmd/benchcall`。
- [x] 新增 Blake2b benchmark 命令 `cryptoupgrade/cmd/benchblake2b`。
- [x] 新增 Solidity Blake2b-256 合约 `cryptoupgrade/contracts/Blake2bSolidity.sol`。
- [x] 支持 Blake2b 三组实验输出：Solidity、升级方案、预编译合约适配。
- [x] 修正 Go plugin 编译参数和模块工作目录。
- [x] 更新 `cryptoupgrade/docs/test_result.md`。

## Verification

- [x] 运行 Add 部署和调用基准实验。
- [x] 运行 Blake2b 三组 smoke test。
- [x] 运行 Blake2b 三组部署实验。
- [x] 运行 Blake2b 三组已部署调用实验。
- [x] 运行相关 Go 包测试。

## Documentation

- [x] 创建 OpenSpec 长期 capability spec。
- [x] 创建 propose、explore、design、tasks、apply、archive 制品。
- [x] 将本次变更归档到 `openspec/changes/archive/2026-07-09-cryptoupgrade-benchmark-suite`。
