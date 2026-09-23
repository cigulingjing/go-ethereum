## 1. WASI 构建

- [x] 1.1 新增 Aigis-sig2 WASI `execute` 封装，解码 ABI `(bytes,bytes,bytes)` 并调用 `pqmagic_aigis_sig2_std_verify`
- [x] 1.2 增加 native 向量生成程序，用同一公开 API 产出 pk/message/signature 测试数据
- [x] 1.3 编写 wasi-sdk 构建脚本，产出 reactor `.wasm` 到 `cryptoupgrade/internal/model/aigis_sig2/`

## 2. EvoCrypt 冒烟

- [x] 2.1 增加 cryptoupgrade 测试：EncodeWasm 后 `uploadCodeVersion` 能登记并发出事件
- [x] 2.2 同一测试中 `ActivateAlgorithm` 成功，随后 `callFunc` 对合法签名返回 true、篡改签名返回 false
- [x] 2.3 确认模块 Activate 不引入非 WASI import
