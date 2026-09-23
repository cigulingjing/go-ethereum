# pqc_verify

四个后量子验签从 `experiments/pqcc` 抽出，放在同一目录：

```text
pqc_verify/
  common/                 # SHAKE、randombytes、verify.c
  aigis_sig2/
  dilithium3/
  ml_dsa_65/
  slh_dsa_shake_192f/
```

```bash
go run ./experiments/pqc_verify
go run ./experiments/pqc_verify aigis_sig2
go run ./experiments/pqc_verify dilithium3 /path/to/testdata
```
