# zkgo

Solidity 不适合作为对照的配对验证归档。当前算法：

```text
zkgo/groth16_bls12381_verify/
  native/                         # verify.c + blst 可移植源码
  groth16_bls12381_verify.wasm
  testdata/                       # vk.bin proof.bin public.bin

zkgo/groth16_bn254_verify/
  native/                         # verify.c + mcl 可移植源码
  groth16_bn254_verify.wasm
  testdata/                       # vk.bin proof.bin public.bin
```

验证方程来自 gnark Groth16 `Verify`（无 commitment）：

```text
e(Ar, Bs) · e(Krs, -δ) · e(L_pub, -γ) · e(α, -β) = 1
```

BLS12-381 用 blst；BN254 / Ethereum alt_bn128 用 mcl `MCL_BN_SNARK1`。

BN254 默认使用仓库内 `groth16_bn254_verify/native/` 的 mcl 源码快照，不会自动下载 mcl。
如需改用另一个本地 mcl 源码目录，可设置 `MCL_SRC=/path/to/mcl`；缺少所需文件时构建直接报错。
BLS12-381 的 blst 构建路径独立，缺少本地 Go 模块缓存时仍可能下载 blst。

重建：

```bash
experiments/cryptoupgrade/algorithm/zkgo/build.sh
go test ./experiments/cryptoupgrade/core/zkbench/...
```

只构建 BN254：

```bash
ZKGO_ALGOS=groth16_bn254_verify experiments/cryptoupgrade/algorithm/zkgo/build.sh
go test ./experiments/cryptoupgrade/core/zkbench/groth16bn254
```
