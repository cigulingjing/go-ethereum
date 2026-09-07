## Why

当前升级路径上传 Go 源码并在各节点执行 `go build -buildmode=plugin`，把 Native 代码和异构本地编译同时放进 Geth 地址空间。论文方案要求改为 WASM 字节码上传、广播下发、协处理器编译，调用仍从 Solidity 进入。

## What Changes

- **BREAKING**：`CodeStorage` 上传物从 Go 源码改为 WASM bytecode，链上权威身份为 `wasmHash`。
- 事件广播后，协处理器异步校验、编译并实例化 WASM，不在升级交易的 EVM 关键路径上执行。
- 用 WASM sandbox 替换 Go plugin 运行时；Solidity `callFunc` 入口保持不变。
- 本地制品目录从 `src/` + `so/` 改为 WASM bytecode 与编译缓存。
- 实验上传路径改为 WASM 模块；旧 Go plugin 结果不作为新正文数值来源。

## Capabilities

### New Capabilities

- `cryptoupgrade-wasm-runtime`: WASM 字节码上传、广播、协处理器编译、沙箱执行，以及保持不变的 Solidity 调用入口。

### Modified Capabilities

- `cryptoupgrade-plugin-directory-management`: 本地制品从 Go 源码 / plugin 改为 WASM bytecode 与编译产物。
- `cryptoupgrade-performance-benchmarks`: 升级对照路径改为 WASM 模块，不再编译 Go plugin。

## Impact

影响 `cryptoupgrade` 的 CodeStorage 载荷语义、event/activation、compiler、pluginruntime、repository，以及实验上传脚本。新增 WASM runtime 依赖。不修改 EVM 解释器与 Precompile 地址。

## 非目标

- 不把 WASM 解释进 EVM，不改变 `callFunc` 调用形状。
- 不在本 change 完成 10 节点规模实验、stateRoot 补采、密码迁移 Lab4 或论文正文重写。
- 不声称 WASM 与 Native Precompile 性能等价，不支持任意不可信代码。
