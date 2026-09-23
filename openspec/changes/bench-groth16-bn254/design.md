## Context

技术栈：Go、CGO、g++、wasi-sdk `clang++ --target=wasm32-wasi`、herumi mcl（BN_SNARK1 / alt_bn128）、gnark Groth16 BN254、wazero。

`pqcgo` 仅有 Aigis/Dilithium/ML-DSA/SLH-DSA，没有 Groth16。`zkgo` 已有 BLS12-381（blst）。BN254 的验证方程与 BLS12-381 相同，但曲线是以太坊 alt_bn128，不能复用 blst。

## Goals / Non-Goals

**Goals:**

- 抽出 gnark Groth16 BN254 无 commitment pairing 核，用 mcl C API 实现。
- CGO 与 WASM 对照，归档到 `zkgo/groth16_bn254_verify/`。
- 复用 zkgo testdata 布局与 in-process 实验入口。

**Non-Goals:**

- 不补 Solidity Lab2，不注册预编译。
- 不把 mcl 用于 BLS12-381（那条路径继续用 blst）。

## Decisions

### 1. pqcgo 无实现，改造 zkgo 而不是 pqcgo

pqcgo 是后量子验签 ABI。BN254 验证核挂到 `zkgo/`，与 BLS12-381 并列。

### 2. 运行时库用 mcl `MCL_BN_SNARK1`

Ethereum / gnark-crypto `bn254` 对应 mcl `BN_SNARK1`，不是 mcl `BN254`。testdata 由 gnark Setup/Prove 生成；G1 为 64B uncompressed，G2 为 128B（gnark `A1|A0` 序，C 侧换到 mcl `A0|A1`）。

备选：手写 BN254 pairing。否决：体积与正确性风险高于快照 mcl。

### 3. 制品与采样对齐 BLS12-381 实验

```text
zkgo/groth16_bn254_verify/
  native/          # verify.c + mcl include/src 快照
  groth16_bn254_verify.wasm
  testdata/        # vk.bin proof.bin public.bin
```

warmup=10，n=100。结果写入 `experiments/cryptoupgrade/results/groth16-bn254/<run-id>/`。`nativeBytes` 只统计 `native/` 根目录下抽出的算法 `.c`（`verify.c`），不含 mcl 封装库、`check.c` 与备份。

## Risks / Trade-offs

- [Risk] gnark G2 坐标序与 mcl 不一致。 → Mitigation: C 侧交换 A0/A1；生成器先 gnark Verify。
- [Risk] mcl 初始化非线程安全。 → Mitigation: 进程内只 init 一次；WASM execute 有锁。
- [Risk] WASM 上 C++ 异常/RTTI。 → Mitigation: `-fno-exceptions -fno-rtti`，`CYBOZU_DONT_USE_EXCEPTION`。

## Migration Plan

新增 BN254 目录与 CGO 包；扩展 `build.sh` 与 bench 算法表。回滚删除该算法目录即可。
