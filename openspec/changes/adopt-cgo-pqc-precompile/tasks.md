## 1. Native PQMagic 构建

- [ ] 1.1 为 `experiments/pqcc` 增加独立的 gcc CMake 脚本，输出到 `native-build` / `native-prefix`，不覆盖现有 Emscripten `libs/`
- [ ] 1.2 原生构建启用 SHAKE 与 ML-DSA，关闭 KEM、SPHINCS-A、TEST、BENCH，并用 `file` 确认产物为 ELF 而非 WASM
- [ ] 1.3 记录复现命令和 `CGO_CFLAGS` / `CGO_LDFLAGS` 路径，避免 CGO 误链 wasm 静态库

## 2. CGO 封装

- [ ] 2.1 新增 `cryptoupgrade/internal/pqcnative`，用 CGO 链接原生 ML-DSA-65 符号，只导出 verify
- [ ] 2.2 在进入 C 层前校验公钥 1952 字节、签名 3309 字节；长度非法直接返回错误
- [ ] 2.3 为 `!cgo` 提供失败实现，不注册静默成功或 WASM 回退
- [ ] 2.4 用固定消息/密钥向量测试 verify 成功、失败和长度拒绝，并确认不读取宿主随机源

## 3. Precompile 注册

- [ ] 3.1 在 `common/cryptoupgrade_contract.go` 增加 `CryptoUpgradeMlDsa65VerifyAddress`（0x5b）
- [ ] 3.2 在 `cryptoupgrade/precompile.go` 注册 `MlDsa65Verify`，ABI 为 `(bytes,bytes,bytes) → bool`，handler 只调 `pqcnative`
- [ ] 3.3 确认 `core/vm` 激活 precompile 映射包含 0x5b，且不与 0x47–0x5a 或 Ethereum 预编译冲突
- [ ] 3.4 增加 EVM 调用测试：合法签名返回 true，错误签名返回 false，非法 ABI 失败

## 4. 路径隔离与校验

- [ ] 4.1 确认 `wasmruntime` / activation / CodeStorage 不引用 `pqcnative`
- [ ] 4.2 确认现有 0x47–0x5a 纯 Go 预编译测试仍然通过
- [ ] 4.3 运行 `openspec validate adopt-cgo-pqc-precompile`
