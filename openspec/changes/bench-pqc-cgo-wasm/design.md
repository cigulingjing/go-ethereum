## Context

技术栈：Go、CGO、gcc/clang、wasi-sdk `clang --target=wasm32-wasi`、PQMagic C、wazero、cryptoupgrade。

Lab2 要求 upgrade / precompile / Solidity 三方案齐全。后量子验签没有可对照的 Solidity 实现，不能进入 Lab2。`bench-aigis-exec-efficiency` 只覆盖 Aigis-sig2，制品仍散落在 `cryptoupgrade/internal/model` 与 `pqcnative`。需要一组可扩展实验：同一算法的 CGO 原生 verify 对 EvoCrypt WASM，并按算法名归档源码与 `.wasm`。

## Goals / Non-Goals

**Goals:**

- 对 Solidity 不适合实现的验签算法，测量 CGO 与 WASM 的 in-process 执行耗时。
- 每条记录包含算法名、两种耗时、原生实现文件大小、WASM 字节码长度。
- 原生源码与 `.wasm` 备份到 `experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/`。

**Non-Goals:**

- 不补 Solidity 合约，不跑 RPC Lab2，不测升级/上传耗时。
- 不把每个算法注册到生产预编译地址表。
- 不把本次结果写入论文正式表。

## Decisions

### 1. 首批算法来自 PQMagic 验签，不接 Solidity

首批：`aigis_sig2_verify`、`dilithium3_verify`、`ml_dsa_65_verify`、`slh_dsa_shake_192f_verify`。与现有 `pqcsign_wrapper` 四方案对齐。只测 verify，输入 ABI 为 `(bytes pk, bytes message, bytes signature) → bool`。

备选：只做 Aigis。否决：用户要求覆盖一组 Solidity 不适用算法。

### 2. 对照粒度：进程内 CGO 核 vs WASM Execute

CGO 调宿主 ELF 的公开 verify；WASM 走已激活模块的 `wasmruntime.Execute`（或等价 `callFunc` 内层）。两边用同一组向量，输出必须一致。

备选：再注册 0x5c+ 生产预编译。否决：本 change 是实验能力，不应扩大 Geth 预编译集合。

### 3. 制品目录按算法名归档

```text
experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/
  native/          # 该算法 CGO 使用的 C 源与头文件快照
  <algorithm>.wasm
  testdata/        # pk.bin / message.bin / signature.bin
```

实验只读该目录。`nativeBytes` 只统计 `native/` 根目录下抽出的算法 `.c`（`verify.c`），不含 PQMagic 封装库、头文件与备份；`wasmBytes` 为 `.wasm` 文件长度。构建脚本同时产出宿主静态库时，JSON 可另记 `nativeArchiveBytes`，但不替代 `nativeBytes`。

### 4. 采样与输出

warmup=10，n=100。记录 mean/p50/p95/min/max 毫秒，以及 `(T_wasm-T_cgo)/T_cgo`。JSON/文本写入 `experiments/cryptoupgrade/results/pqc-cgo-wasm/<run-id>/`。缺少 CGO 或缺失制品时跳过该算法并写原因，不得用另一路径冒充。

## Risks / Trade-offs

- [Risk] 现有 `libs/*.so` 是 WASM，不能给 CGO。 → Mitigation: 用 gcc 从 PQMagic 源码编宿主对象，与 wasi-sdk 产物分开。
- [Risk] 四算法 CGO 拉长 `go test`。 → Mitigation: 实验入口独立；cryptoupgrade 包测试不默认编全部方案。
- [Risk] 源码快照与 PQMagic 上游漂移。 → Mitigation: 每个算法目录记录构建命令与源版本说明。

## Migration Plan

新增 `pqcgo/`、构建脚本与实验命令。回滚删除该目录与命令即可。已有 Aigis 试点可复制进 `pqcgo/aigis_sig2_verify/`，不删除 `cryptoupgrade/internal/model` 中的嵌入副本，直到后续清理。

## Open Questions

- Dilithium / ML-DSA / SLH-DSA 的 WASI `execute` 封装是否复用 Aigis 的三参数 ABI；默认是。
