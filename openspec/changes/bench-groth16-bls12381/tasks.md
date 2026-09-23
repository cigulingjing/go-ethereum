## 1. 验证核与制品归档

- [x] 1.1 从 gnark Groth16 BLS12-381 `Verify` 抽出无 commitment pairing 方程，写成 blst C `verify.c`
- [x] 1.2 建立 `experiments/cryptoupgrade/algorithm/zkgo/groth16_bls12381_verify/`，快照 blst 可移植源码到 `native/`，约定 testdata 为 `vk.bin`/`proof.bin`/`public.bin`
- [x] 1.3 增加 gnark 小型电路生成器，产出 testdata，并用 gnark `Verify` 确认证明合法

## 2. CGO 与 WASM

- [x] 2.1 为归档算法提供 CGO verify，只读 `zkgo/.../native/`；`!cgo` 返回可识别错误
- [x] 2.2 用 wasi-sdk 编译同一 C 源为 `execute` 导出的 `.wasm`

## 3. 实验入口

- [x] 3.1 新增 in-process 对照：warmup=10、n=100，CGO verify 与 WASM Execute，校验输出一致
- [x] 3.2 JSON 写出 `algorithm`、CGO/WASM 耗时、`nativeBytes`、`wasmBytes` 到 `experiments/cryptoupgrade/results/groth16-bls12381/<run-id>/`
- [x] 3.3 运行首轮对照，确认产出完整记录或明确跳过原因
