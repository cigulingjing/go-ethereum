## Why

论文评审指出现有算法偏简单，需要 ZK-SNARK 等含双线性对的复杂协议。`experiments/gnark` 已有 Groth16 BLS12-381 验证核，BLS12-381 的 Solidity 依赖 EIP-2537，不能进入 Lab2 三方案对照。需要按 `bench-pqc-cgo-wasm` 的方式，把该验证核做成 CGO 与 WASM 的进程内对照。

## What Changes

- 从 gnark Groth16 BLS12-381 `Verify` 抽出无 commitment 的 pairing 验证核。
- 用 blst 可移植 C 实现同一方程，CGO 与 wasi-sdk WASM 共用源码。
- 用小型电路生成归档 testdata，并测量 in-process 执行耗时与制品大小。

## Capabilities

### New Capabilities

- `groth16-bls12381-benchmark`: Groth16 BLS12-381 验证的 CGO/WASM 对照、指标落盘与制品备份。

### Modified Capabilities

- None.

## Impact

新增 `zkgo/groth16_bls12381_verify/`、CGO 封装与实验命令。不改 EvoCrypt ABI、共识规则、Lab2 语义或生产预编译地址表。

## 非目标

- 不补 Solidity 对照，不跑 RPC Lab2。
- 不把本次结果写成正式论文表。
- 不实现 Prove/Setup，不覆盖 Pedersen commitment 变体。
- 不把该算法注册为生产预编译地址。
