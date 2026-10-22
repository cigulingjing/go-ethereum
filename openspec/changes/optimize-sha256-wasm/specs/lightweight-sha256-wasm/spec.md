## Purpose

提供可独立上传的低依赖 SHA-256 WASM 算法，在保持哈希计算和 ABI 输出完全兼容的前提下减少制品体积及其升级交易的 calldata Gas。

## ADDED Requirements

### Requirement: SHA-256 结果兼容

独立 WASM 算法 SHALL 对任意输入返回标准 SHA-256 的 32 字节摘要，并保持现有 `bytes -> bytes` 调用接口。

#### Scenario: 边界输入与多块输入
- **WHEN** 输入为空、位于 64 字节块及填充边界两侧或包含多个块
- **THEN** 输出 SHALL 与 Go 标准库 `crypto/sha256` 的 SHA-256 结果完全一致

### Requirement: 控制上传制品体积

实验制品 SHALL 避免因通用密码学包的初始化引入无关算法实现，并保持 SHA-256 的 WASM 原始文件小于 16 KiB。

#### Scenario: 构建独立 WASM
- **WHEN** 使用现有 WASM 构建工具编译 SHA-256 算法
- **THEN** 构建产物 SHALL 小于 16 KiB，且可由原有升级和调用流程使用

### Requirement: 小型模块输入安全

WASM 运行时 SHALL 将 ABI 输入写入模块初始化内存范围之外，避免输入覆盖模块内部堆数据。

#### Scenario: 跨越原有固定地址的输入
- **WHEN** 小型 SHA-256 模块接收跨越 192 字节边界或更长的输入
- **THEN** 其摘要 SHALL 与标准 SHA-256 一致，不得因输入缓冲区与模块内部内存重叠而变化

### Requirement: 可复现实验对照

实验 SHALL 保留优化前后的制品大小、上传编码大小及同链配置下交易收据的 Gas 对照，不将 estimate 值充当 receipt Gas。

#### Scenario: 报告升级 Gas
- **WHEN** 上传优化后的 WASM 并得到成功收据
- **THEN** 实验记录 SHALL 包含文件摘要、字节长度、calldata 长度及 receipt gasUsed
