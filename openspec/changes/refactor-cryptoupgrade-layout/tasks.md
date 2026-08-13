## 1. 目录盘点与迁移边界

- [x] 1.1 盘点 `cryptoupgrade/` 根包文件、实验命令、算法资产、网络库、示例配置、文档和结果目录，确认每类文件的目标目录。
- [x] 1.2 使用 `rg` 查找项目内所有 `cryptoupgrade/cmd`、`cryptoupgrade/algorithm`、`cryptoupgrade/examples` 和 `cryptoupgrade/results` 路径引用，记录需要随迁移更新的代码和文档。
- [x] 1.3 确认 `core/vm`、`node` 和 `cmd/geth` 仍只依赖 `github.com/ethereum/go-ethereum/cryptoupgrade` 运行时根包，不引入实验目录导入。

## 2. 实验命令目录迁移

- [x] 2.1 创建 `cryptoupgrade/bench/cmd` 目录，并将现有 `cryptoupgrade/cmd/*` 实验命令移动到该目录下。
- [x] 2.2 更新移动后命令中的相对路径、测试 package 路径和内部引用，保持原有 flag、JSON/CSV 输出 schema 和默认行为不变。
- [x] 2.3 删除空的旧 `cryptoupgrade/cmd` 目录，并确认仓库内不再新增实验命令到旧目录。

## 3. 默认路径与文档更新

- [x] 3.1 更新 benchmark 和升级延迟工具的默认源码路径，确保 Go 输入仍来自 `cryptoupgrade/algorithm/go`，Solidity 输入仍来自 `cryptoupgrade/algorithm/contracts`。
- [x] 3.2 更新多节点工具和升级延迟实验的默认示例配置路径，确保继续指向 `cryptoupgrade/examples` 下的可复现实验配置。
- [x] 3.3 更新 `cryptoupgrade/docs`、OpenSpec 文档和 README 中的 `go run ./cryptoupgrade/cmd/...` 示例到 `go run ./cryptoupgrade/bench/cmd/...`。
- [x] 3.4 保留历史 `cryptoupgrade/results` 数据，不重写历史结果中的绝对路径；仅更新新实验默认输出路径和说明。

## 4. 布局检查

- [x] 4.1 增加或扩展目录布局检查脚本，验证实验命令位于 `cryptoupgrade/bench/cmd`，算法资产位于规范算法目录。
- [x] 4.2 在布局检查中标记新增的 `cryptoupgrade/cmd/*` 实验命令、根目录实验结果文件和偏离规范的默认路径。
- [x] 4.3 将布局检查命令记录到相关文档，便于后续新增实验前复用。

## 5. 验证

- [x] 5.1 对迁移涉及的 Go 文件运行 `gofmt`。
- [x] 5.2 运行 `go test ./cryptoupgrade/...`，确认运行时、网络库和移动后的实验命令测试通过。
- [x] 5.3 对关键命令运行 `go test` 或 `go run -h` smoke test，确认新路径下命令可构建。
- [x] 5.4 运行算法接口检查和新增布局检查。
- [x] 5.5 运行 `openspec validate refactor-cryptoupgrade-layout`。
