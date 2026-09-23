## Context

技术栈：Go、CGO、gcc、wasi-sdk `clang --target=wasm32-wasi`、blst 可移植 C、gnark Groth16 BLS12-381、wazero、cryptoupgrade。

`experiments/gnark/backend/groth16/bls12-381/verify.go` 的验证核是 pairing product：

```text
e(Ar, Bs) · e(Krs, -δ) · e(L_pub, -γ) · e(α, -β) = 1
```

其中 `L_pub = K[0] + Σ public[i]·K[i+1]`。无 commitment 的电路不走 Pedersen 分支。BLS12-381 Solidity 依赖 EIP-2537，不能进 Lab2。对照方式对齐 `bench-pqc-cgo-wasm`：同一 C 源、同一向量、进程内 CGO vs WASM Execute。

## Goals / Non-Goals

**Goals:**

- 抽出无 commitment 的 Groth16 BLS12-381 验证核，并用 blst C 实现同一方程。
- 测量 CGO 与 WASM 的 in-process 耗时，记录制品大小。
- 原生源码与 `.wasm` 备份到 `experiments/cryptoupgrade/algorithm/zkgo/groth16_bls12381_verify/`。

**Non-Goals:**

- 不补 Solidity，不跑 RPC Lab2，不测升级/上传。
- 不注册生产预编译。
- 不实现 Prove/Setup，不覆盖 commitment 变体。
- 不把本次结果写入论文正式表。

## Decisions

### 1. 验证核来自 gnark，运行时用 blst C

从 `verify.go` 提取无 commitment 的 pairing check。testdata 由 gnark Setup/Prove 生成，点编码用 ZCash uncompressed（G1 96B、G2 192B），与 blst `deserialize` 对齐。运行时 CGO/WASM 都走 blst，避免把完整 gnark 链进 WASM。

备选：TinyGo 编译 gnark-crypto。否决：当前环境无 tinygo，且配对库含汇编。

### 2. 对照粒度：进程内 CGO vs WASM Execute

ABI：`(bytes vk, bytes proof, bytes public) → bool`，实验入口仍按三段 `bytes` 打包，与 PQC harness 相同。两边输出必须同为成功。

备选：再注册预编译地址。否决：本 change 是实验能力。

### 3. 制品目录

```text
experiments/cryptoupgrade/algorithm/zkgo/groth16_bls12381_verify/
  native/          # verify.c + blst 可移植源码快照
  groth16_bls12381_verify.wasm
  testdata/        # vk.bin / proof.bin / public.bin
```

`nativeBytes` 只统计 `native/` 根目录下抽出的算法 `.c`（`verify.c`），不含 blst 封装库与备份。blst 使用 `-D__BLST_NO_ASM__ -D__BLST_NO_CPUID__`，使宿主与 WASM 源码一致。

### 4. 采样与输出

warmup=10，n=100。记录 mean/p50/p95/min/max 毫秒，以及 `(T_wasm-T_cgo)/T_cgo`。结果写入 `experiments/cryptoupgrade/results/groth16-bls12381/<run-id>/`。缺少 CGO 或制品时跳过并写原因。

## Risks / Trade-offs

- [Risk] gnark 与 blst 的 G2 坐标序或 pairing 约定不一致。 → Mitigation: 生成器先用 gnark `Verify` 确认证明合法，再要求 C 路径对同一向量返回成功。
- [Risk] WASM 上 pairing 栈/内存不足。 → Mitigation: 增大 wasi stack 与 linear memory；超时记失败而非冒充 CGO。
- [Risk] 快照 blst 与 geth 依赖版本漂移。 → Mitigation: `SOURCE.txt` 记录 blst 版本与构建命令。

## Migration Plan

新增 `zkgo/`、构建脚本与 `benchgroth16cgowasm`。回滚删除该目录与命令即可。

## Open Questions

- 若 pairing 单次超过数秒，是否把 n 降到 20；默认仍 n=100，超时再降。
