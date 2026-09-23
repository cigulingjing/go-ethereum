# common

四个后量子验签共用的依赖，来自 `experiments/pqcc`：

- SHAKE / FIPS 202（`fips202.c`）
- `randombytes`
- 共用 `verify.c` 入口
- `include/pqmagic_config.h`（用 `-DPQC_NAMESPACE=...` 区分符号）

算法特有的 `sign.c` / `poly.c` / `ntt.c` 等仍留在各自子目录的 `native/`。
