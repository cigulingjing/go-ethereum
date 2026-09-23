## Context

技术栈：C（PQMagic Aigis-sig2 + SHAKE）、wasi-sdk `clang --target=wasm32-wasi`、Geth cryptoupgrade / wazero。

EvoCrypt 只接受 WASI import 与 `execute` 导出。现有 pqcc `build.sh` 产出浏览器 JS glue，不能加载。Aigis 旧 wrapper 用 `AIGIS_SIG_MODE=2` 却调用 `aigis_sig3_*` 且走 `_verify_internal`（未加 ctx 前缀），验签失败。本试点纠正这两点。

## Goals / Non-Goals

**Goals:**

- 编译 Aigis-sig2 verify 为 reactor WASI 模块。
- 走 EncodeWasm → activation → wasmruntime.Execute / callFunc。
- 合法签名返回 true，篡改签名返回 false。

**Non-Goals:**

- 不上链跑私有网络（可用 in-process CodeStorage 冒烟）。
- 不实现 keygen/sign 的链上入口。
- 不改 precompile / CGO 路径。

## Decisions

### 1. 参数集固定 Aigis-sig2

与 PQMagic 默认 `AIGIS_SIG_MODE=2` 一致。调用 `pqmagic_aigis_sig2_std_verify(..., ctx=NULL, ctx_len=0)`，与公开 sign API 的 ctx 包装对齐。

### 2. Guest ABI 对齐 Schnorr 风格的 bytes/bool，但三参数

`itype=bytes,bytes,bytes`，`otype=bool`。模块内解码 ABI 动态 bytes，返回 32 字节 ABI bool，再加 4 字节小端长度前缀。宿主不改 `wasmruntime`。

### 3. 构建工具用 wasi-sdk，不用 Emscripten web

`clang --target=wasm32-wasi -mexec-model=reactor -Wl,--no-entry --export=execute`。随机数仅 native 向量生成使用；verify 不依赖 RNG。wasi-libc 的 WASI import 已在 wazero 白名单内。

## Risks / Trade-offs

- [Risk] 本机无 wasi-sdk。 → Mitigation: 构建脚本下载到 `~/.local/opt/wasi-sdk`；测试在缺少 `.wasm` 时失败并提示构建命令。
- [Risk] Aigis 内部 `malloc` 与 16MB 内存上限。 → Mitigation: 设置 initial memory 2MB、max 16MB。
- [Risk] 合约 24KB 限制与本路径无关。 → WASM 上传走 CodeStorage 字符串，受 gzip 后 calldata 约束，Aigis 模块体积需在构建后记录。

## Migration Plan

1. 增加 WASI wrapper 与 native 向量生成。
2. 编译 wasm，写入 `cryptoupgrade/internal/model/aigis_sig2/`。
3. 增加 cryptoupgrade 冒烟测试。
4. 回滚：删除 wasi 目录与测试即可，不影响现有算法。

## Open Questions

- 制品是提交预编译 `.wasm` 还是 CI 现编。倾向提交一份构建产物以便无 wasi-sdk 的环境也能跑测试。
