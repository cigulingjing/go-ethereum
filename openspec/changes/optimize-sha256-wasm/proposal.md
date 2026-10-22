## Why

`Sha256` 的 WASM 虽然只调用标准库函数，导入 `crypto/sha256` 却使构建产物达到约 68 KB；相同 ABI 包装的无依赖对照只有约 7 KB，显著增加 `uploadCode` 的 calldata Gas。

## What Changes

- 将独立的 WASM `Sha256` 算法改为不导入标准库 `crypto` 栈的源码实现，保留原函数签名和 SHA-256 输出语义。
- 添加与标准库对照的边界、随机和多块输入测试；重新生成 WASM 并记录编码大小及升级 Gas 的可复现对照。
- 修复小型 WASM 模块因固定输入地址与初始堆重叠而造成的长输入错误。

## Capabilities

### New Capabilities

- `lightweight-sha256-wasm`: 独立 SHA-256 WASM 算法应保持正确性，同时避免不必要的密码学标准库依赖及上传体积。

### Modified Capabilities

无。

## Impact

涉及 `experiments/cryptoupgrade/algorithm/go/sha256.go`、其 WASM 制品、WASM 执行边界、算法测试和实验结果；不修改 CodeStorage ABI 或原有 Gas 计价规则。

## 非目标

不替换 `Pbkdf2Sha256`、`PedersenCommit` 或 `SchnorrVerify` 的依赖，也不将实验性源码实现作为经过审计的生产密码库。
