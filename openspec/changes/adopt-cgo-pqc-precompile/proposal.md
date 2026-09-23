## Why

此前把“纯 Go、禁止 CGO、必须 WASM”错误套到了 cryptoupgrade 预编译路径。预编译对照必须是客户端原生实现，不能走 WASM；pqcc/PQMagic 这类 C 算法应通过 CGO 调用原生 `.so`。WASM 只属于 EvoCrypt 动态升级路径。

## What Changes

- 明确双路径：cryptoupgrade 预编译 = CGO/原生 `.so`；EvoCrypt = wazero WASM。
- 用 gcc/clang 编译 PQMagic 原生共享库，不再把 Emscripten wasm 目标文件当 `.so`。
- 新增 CGO 封装，把 ML-DSA 等后量子签名接入 native precompile。
- 纠正 `dynamic-wasm` 中“Geth 不得使用 CGO”的约束范围：该约束只约束 EvoCrypt runtime 选型，不约束预编译对照。

## Capabilities

### New Capabilities

- `cryptoupgrade-cgo-native-runtime`: 原生 PQMagic `.so` 的编译、CGO 绑定、以及仅用于 precompile 的调用边界。

### Modified Capabilities

- `cryptoupgrade-precompiled-algorithms`: 预编译实现允许 CGO / dlopen `.so`，禁止 WASM；后量子验签作为确定性 native 入口注册。

## Impact

影响 `cryptoupgrade/precompile.go`、`common/cryptoupgrade_contract.go`、`experiments/pqcc` 原生构建，以及 Geth 构建需 `CGO_ENABLED=1`。不改 EvoCrypt 的 WASM 上传、activation 与 `callFunc` 语义。

## 非目标

- 不把 EvoCrypt 改回 Go plugin，不删除 wazero。
- 不在本 change 完成 KEM 预编译、论文重写或正式 Lab 重跑。
- 不把 Emscripten `ENVIRONMENT=web` 产物链进 Geth。
