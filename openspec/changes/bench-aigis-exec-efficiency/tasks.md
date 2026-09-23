## 1. Native CGO

- [x] 1.1 新增 `cryptoupgrade/internal/pqcnative`，CGO 编译 Aigis-sig2 + SHAKE 源码并导出 verify
- [x] 1.2 `!cgo` 提供失败桩，`Enabled()` 为 false

## 2. 预编译注册

- [x] 2.1 在 `common` 增加 `CryptoUpgradeAigisSig2VerifyAddress`（0x5b）
- [x] 2.2 在 `precompile.go` 注册 `AigisSig2Verify`，handler 只调 `pqcnative`
- [x] 2.3 用嵌入向量断言合法签名 true、篡改 false

## 3. 效率对比

- [x] 3.1 增加 in-process 测试：激活 WASM 后对比 `callFunc` 与 `Precompile.Run`
- [x] 3.2 warmup=10、n=100，写出 JSON 到 `experiments/cryptoupgrade/results/aigis-exec-efficiency/`
- [x] 3.3 运行对比测试并确认 WASM 与预编译输出一致
