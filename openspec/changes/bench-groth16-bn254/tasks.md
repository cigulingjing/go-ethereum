## 1. 验证核与制品

- [x] 1.1 确认 pqcgo 无 Groth16/BN254，从 gnark BN254 `Verify` 抽出 pairing 方程写成 mcl C `verify.c`
- [x] 1.2 快照 mcl 可移植源码到 `zkgo/groth16_bn254_verify/native/`，testdata 为 `vk.bin`/`proof.bin`/`public.bin`
- [x] 1.3 增加 gnark BN254 小型电路生成器，并用 gnark `Verify` 确认证明合法

## 2. CGO 与 WASM

- [x] 2.1 提供 CGO verify，只读 `zkgo/groth16_bn254_verify/native/`；`!cgo` 返回可识别错误
- [x] 2.2 用 wasi-sdk 编译同一 mcl 源与 verify.c 为 `execute` 导出的 `.wasm`

## 3. 实验入口

- [x] 3.1 扩展 in-process 对照入口，支持 `groth16_bn254_verify`，warmup=10、n=100
- [x] 3.2 JSON 写入 `experiments/cryptoupgrade/results/groth16-bn254/<run-id>/`
- [x] 3.3 运行首轮对照，确认完整记录或明确跳过原因
