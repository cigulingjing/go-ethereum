## Context

EvoCrypt 侧已有 Aigis-sig2 WASI 模块与固定测试向量。现有 `libs/*.so` 是 WASM，不能给预编译用。预编译必须编译宿主 ELF，经 CGO 调用同一套 `crypto_sign_verify(..., ctx=NULL)`。

## Goals / Non-Goals

**Goals:**

- 同一算法、同一向量上比较 WASM `callFunc` 与 native `Precompile.Run` 的墙钟延迟。
- 预编译实现不得调用 `wasmruntime`。

**Non-Goals:**

- 不测升级/上传耗时，不测 RPC/`eth_estimateGas`。
- 不把 CGO 扩到全部 PQMagic 算法。

## Decisions

### 1. 对照粒度：in-process 调用路径

Lab2 的 `eth_call` 含 RPC。本次只量进程内路径：WASM 走已激活模块的 `callFunc`，预编译走 `Precompile.Run`，输入均为 ABI `(bytes,bytes,bytes)`。

备选：完整 EVM `CALL`。否决：CodeStorage 与 0x5b 的外层包装不对称，会把调度噪声算进算法差。

### 2. CGO 直接编 PQMagic 源码

`cryptoupgrade/internal/pqcnative` 用 CGO 编译 Aigis-sig2 + SHAKE 源文件，避免依赖 Emscripten `.so`。`!cgo` 提供失败桩，不注册静默成功。

### 3. 预编译地址 0x5b

接在现有 0x47–0x5a 之后。ABI 与 WASM 试点一致。gas 使用 heavy 档，与模型中 200000 对齐量级。

### 4. 采样

warmup=10，n=100，报告 mean/p50/p95/min/max，以及 `(T_wasm - T_precompile) / T_precompile`。

## Risks / Trade-offs

- [Risk] CGO 使 cryptoupgrade 测试变慢。 → Mitigation: 源码只含 verify 依赖；结果不作为 CI 门禁超时来源。
- [Risk] WASM 与 native 优化级别不同。 → Mitigation: 双方都用 `-O3` / 已提交的 wasm 制品，结果注明构建方式。

## Migration Plan

新增包与预编译条目；回滚删除 `pqcnative` 与 0x5b 即可。
