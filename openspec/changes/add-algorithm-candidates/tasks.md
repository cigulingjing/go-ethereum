## 1. OpenSpec 重整

- [x] 1.1 将旧的候选算法筛选计划改为多项式乘法内核实验计划。
- [x] 1.2 删除旧的跳过类别要求，新增三路径等价实现要求。

## 2. 算法实现

- [x] 2.1 在 `cryptoupgrade/algorithm/go` 增加 `PolynomialMul` 源码。
- [x] 2.2 在 `cryptoupgrade/algorithm/contracts` 增加 Solidity `PolynomialMul` 合约。

## 3. Native precompile

- [x] 3.1 为 `PolynomialMul` 分配 cryptoupgrade precompile 地址。
- [x] 3.2 在 Geth precompile 注册表中接入 native Go `PolynomialMul`。
- [x] 3.3 增加 precompile 功能等价测试。

## 4. WASM 协处理器

- [x] 4.1 扩展 `wasmtool` wrapper，支持 `uint256[],uint256[],uint256 -> uint256[]`。
- [x] 4.2 将 `PolynomialMul` 加入 WASM 构建脚本。

## 5. 统一性能脚本

- [x] 5.1 将 `PolynomialMul` 加入三路径执行效率 benchmark fixture。
- [x] 5.2 增加只运行该实验的便捷脚本。

## 6. 验证

- [x] 6.1 运行 Go 单元测试覆盖 precompile、benchmark fixture 和 WASM wrapper。
- [x] 6.2 使用 solc 编译 Solidity `PolynomialMul` 合约。
- [ ] 6.3 在 TinyGo 可用时构建 `PolynomialMul` WASM 产物。
  - 当前环境 `tinygo` 不在 `PATH`，`wasmbuild` 已验证到工具缺失错误。
