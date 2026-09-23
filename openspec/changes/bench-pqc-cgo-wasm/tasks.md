## 1. 制品归档

- [x] 1.1 建立 `experiments/cryptoupgrade/algorithm/pqcgo/`，按算法名建目录，约定 `native/`、`.wasm`、`testdata/`
- [x] 1.2 将已有 Aigis-sig2 原生源码、WASM 与向量复制到 `pqcgo/aigis_sig2_verify/`
- [x] 1.3 为 Dilithium3、ML-DSA-65、SLH-DSA-SHAKE-192f 增加 WASI `execute` 封装、native 向量生成与 `pqcgo/<algorithm>/` 归档

## 2. CGO 对照

- [x] 2.1 为每个归档算法提供独立 CGO verify（或按算法文件拆分），只读 `pqcgo/<algorithm>/native/`
- [x] 2.2 `!cgo` 与缺失制品时返回可识别错误，实验侧跳过而不是静默成功

## 3. 实验入口

- [x] 3.1 新增 in-process 对照：warmup=10、n=100，CGO verify 与 WASM Execute，校验输出一致
- [x] 3.2 JSON 写出 `algorithm`、CGO/WASM 耗时、`nativeBytes`、`wasmBytes` 到 `experiments/cryptoupgrade/results/pqc-cgo-wasm/<run-id>/`
- [x] 3.3 运行首轮对照，确认 Aigis 至少产出完整记录，其余算法有结果或跳过原因
