## Why

`pqcgo` 只有后量子验签，没有 Groth16。已有 `zkgo` 只覆盖 BLS12-381。BN254 是以太坊既有配对曲线上的 Groth16 验证核，需要同样的 CGO/WASM 进程内对照。

## What Changes

- 从 gnark Groth16 BN254 `Verify` 抽出无 commitment pairing 方程。
- 用 mcl C API（`MCL_BN_SNARK1` / alt_bn128）实现同一方程，CGO 与 wasi-sdk WASM 共用源码。
- 复用 `zkgo` 归档与 `benchgroth16cgowasm` 入口，增加 `groth16_bn254_verify`。

## Capabilities

### New Capabilities

- `groth16-bn254-benchmark`: Groth16 BN254 验证的 CGO/WASM 对照、指标落盘与制品备份。

### Modified Capabilities

- None.

## Impact

新增 `zkgo/groth16_bn254_verify/`、mcl 源码快照与 CGO 封装。扩展现有 groth16 实验入口。不改 EvoCrypt ABI、共识规则或生产预编译。

## 非目标

- 不补 Solidity Lab2，不跑 RPC。
- 不把 BN254 注册为生产预编译。
- 不实现 Prove/Setup，不覆盖 commitment 变体。
- 不把本次结果写成正式论文表。
