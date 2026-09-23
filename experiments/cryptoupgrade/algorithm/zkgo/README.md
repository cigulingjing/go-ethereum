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

重建：

```bash
experiments/cryptoupgrade/algorithm/zkgo/build.sh
go run ./experiments/groth16_bls12381_verify
go run ./experiments/groth16_bn254_verify
```

只构建 BN254：

```bash
ZKGO_ALGOS=groth16_bn254_verify experiments/cryptoupgrade/algorithm/zkgo/build.sh
go run ./experiments/groth16_bn254_verify
```
