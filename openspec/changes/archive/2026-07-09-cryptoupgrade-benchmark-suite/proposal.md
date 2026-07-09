# Proposal: Cryptoupgrade Benchmark Suite

## What

为智能合约升级方案补齐一组可复现的性能基准实验与结果文档，覆盖：

- 升级方案部署阶段的资源消耗：gas 与耗时。
- 升级方案部署后的调用资源对比：调用耗时，并为调用 gas 指标保留规范入口。
- 密码算法三组对照：Solidity 纯合约实现、升级方案 Go plugin 实现、预编译合约实现。

## Why

当前智能合约升级方案的核心价值是将算法逻辑迁移到 geth 客户端侧动态编译执行。实验需要证明两个问题：

- 部署阶段引入 Go plugin 编译和激活后，资源成本是否可接受。
- 部署后调用阶段，相比 Solidity 纯合约和预编译合约，升级方案的执行效率处于什么位置。

## Scope

本次变更覆盖以下内容：

- 新增 Add 基线 benchmark，比较升级方案与传统最小 EVM 合约。
- 新增 Blake2b Sum256 三组 benchmark，比较 Solidity 纯合约、升级方案和预编译合约适配。
- 新增 Solidity 纯 Blake2b-256 合约实现。
- 修正 Go plugin 编译环境，使上传算法可以引用 go-ethereum 模块内算法库。
- 记录测试结果到 `cryptoupgrade/docs/test_result.md`。
- 建立 OpenSpec 过程文档和长期 capability spec。

## Non-Goals

- 不将 Solidity Blake2b 实现扩展到任意长度输入；当前实验输入限制为不超过 128 字节。
- 不把预编译合约注册为新的自定义地址；当前使用 BLAKE2F 预编译能力并通过适配合约对齐接口。
- 不在本次归档中强制补齐所有调用 gas 结果；调用 gas 已作为基准指标写入长期规范。

## Success Criteria

- Blake2b Sum256 三组实现返回相同 digest。
- 基准结果能展示部署耗时、部署 gas、调用耗时和组间比值。
- 结果文档能说明 Solidity、升级方案、预编译合约三组实验的实现差异。
- 相关 Go 包测试通过。
