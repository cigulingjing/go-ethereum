## 1. Registry 和地址

- [x] 1.1 审计 `cryptoupgrade/algorithm` 导出入口，并选择 deterministic、有界的子集作为 precompile 暴露。
- [x] 1.2 在现有 `0x43` 到 `0x46` 系统地址之后增加集中式 cryptoupgrade 算法 precompile 地址常量。
- [x] 1.3 增加可 import 的 native 算法 registry，包含 address、name、input ABI types、output ABI types、gas rule 和 handler metadata。
- [x] 1.4 确保 registry 拒绝或忽略依赖随机数、运行时 plugin loading、编译器状态或文件系统状态的入口。

## 2. Native 算法适配器

- [x] 2.1 为选定 deterministic 算法增加 native implementation 或 shared wrapper，不 import `cryptoupgrade/algorithm` 的 package-main 文件。
- [x] 2.2 为直接 precompile 输入和输出实现 ABI encode/decode helpers。
- [x] 2.3 为每个已注册算法实现 deterministic gas formula，包括 overflow-safe saturation。
- [x] 2.4 增加输入校验，拒绝空 deterministic seed 等会触发 random fallback 的形式。

## 3. VM Precompile 集成

- [x] 3.1 增加一个 `core/vm` wrapper，为 cryptoupgrade registry entry 实现 `PrecompiledContract`。
- [x] 3.2 将 cryptoupgrade 算法 precompile 合并到 `activePrecompiledContracts`，且不修改既有 fork map。
- [x] 3.3 在当前 chain rules 的 `ActivePrecompiles` 结果中包含 cryptoupgrade 算法地址。
- [x] 3.4 保持既有 `CodeStorageAddress` special-case 路径不变，用于 upload/query/plugin calls。

## 4. 测试

- [x] 4.1 增加 registry 测试，覆盖 address uniqueness、name uniqueness、deterministic subset selection 和 ABI metadata validity。
- [x] 4.2 增加直接 `RunPrecompiledContract` fixture 测试，覆盖成功输出、bool ABI 输出、无效 ABI 输入和 out-of-gas 行为。
- [x] 4.3 增加 EVM call-path 测试，证明 cryptoupgrade precompile 通过正常 precompile dispatch 执行，包括 `STATICCALL`。
- [x] 4.4 增加回归测试，证明既有 Ethereum precompile、`Blake2bSum256Address` 和 `CodeStorageAddress` 行为仍可用。

## 5. 验证

- [x] 5.1 对修改的 Go 文件运行 `gofmt` 和 `goimports`。
- [x] 5.2 对 `cryptoupgrade` 和 `core/vm` 运行目标 package 测试。
- [x] 5.3 运行 `go run ./build/ci.go test -short`。
- [x] 5.4 如果准备提交，运行完整 AGENTS pre-commit checklist。
