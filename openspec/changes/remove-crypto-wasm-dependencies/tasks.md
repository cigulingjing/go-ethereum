## 1. 实现

- [x] 1.1 移除 Pedersen 的 crypto 依赖并验证结果。
- [x] 1.2 移除 PBKDF2 的 crypto 依赖并验证边界和迭代结果。
- [x] 1.3 实现无 crypto 依赖的 Ed25519 验签和三参数 ABI。

## 2. 验证

- [x] 2.1 添加标准库和 RFC 测试及实际 WASM 调用测试。
- [x] 2.2 重建制品、比较体积并通过 OpenSpec 严格验证。
