# 独立算法模块

验证核从 `experiments/gnark/` 或 `experiments/pqcc/` 抽出，可单独编译运行。不依赖 `cryptoupgrade` 实验入口。

四个后量子验签集中在 `experiments/pqc_verify/`（共用 `common/`）。

```bash
go run ./experiments/pqc_verify
go run ./experiments/pqc_verify aigis_sig2
go run ./experiments/groth16_bn254_verify
go run ./experiments/groth16_bls12381_verify
```

| 目录 | 抽出的实现 | 来源 |
|---|---|---|
| `pqc_verify/aigis_sig2/` | Aigis-sig2 `crypto_sign_verify` | `pqcc/pqmagic/sig/aigis-sig` |
| `pqc_verify/dilithium3/` | Dilithium3 verify | `pqcc/pqmagic/sig/dilithium` |
| `pqc_verify/ml_dsa_65/` | ML-DSA-65 verify | `pqcc/pqmagic/sig/ml_dsa` |
| `pqc_verify/slh_dsa_shake_192f/` | SLH-DSA-SHAKE-192f verify | `pqcc/pqmagic/sig/slh_dsa` |
| `groth16_bn254_verify/` | Groth16 pairing verify | `gnark/backend/groth16/bn254/verify.go` |
| `groth16_bls12381_verify/` | Groth16 pairing verify | `gnark/backend/groth16/bls12-381/verify.go` |

可选参数：算法名、testdata 目录路径。默认使用模块内 `testdata/`。
