## 1. 算法目录和命名

- [x] 1.1 确认 Go 算法源码只位于 `cryptoupgrade/algorithm/go`。
- [x] 1.2 确认 Solidity 算法合约只位于 `cryptoupgrade/algorithm/contracts`。
- [x] 1.3 将 Solidity 算法文件和 contract name 改为稳定映射名称，去除 `Solidity` 后缀并避免与外部入口冲突。
- [x] 1.4 删除当前未实现或实现困难算法的占位 Solidity 合约。
- [x] 1.5 将没有 Solidity 对照的 Go-only 文件移动到 `cryptoupgrade/algorithm/go/archive`。

## 2. 单一暴露入口

- [x] 2.1 清理 Go 算法文件，使每个文件只保留一个导出算法函数。
- [x] 2.2 清理 Solidity 算法合约，使每个可部署算法合约只保留一个 external/public 算法函数。
- [x] 2.3 保留必要的 internal/private helper 和 abstract helper，并确保它们不作为算法入口统计。

## 3. Benchmark 和文档引用

- [x] 3.1 更新 benchmark 命令默认 Go 源码路径到 `cryptoupgrade/algorithm/go`。
- [x] 3.2 更新 benchmark 命令默认 Solidity 源码路径、contract name 和 function name。
- [x] 3.3 更新文档和结果示例中的旧算法路径引用。

## 4. 一致性检查

- [x] 4.1 增加或更新静态一致性检查脚本，统计 Go/Solidity 算法入口匹配情况。
- [x] 4.2 运行一致性检查并确认 Solidity 已实现算法与 Go 操作逻辑一致。

## 5. 验证

- [x] 5.1 对修改的 Go 文件运行 `gofmt` 和 `goimports`。
- [x] 5.2 使用 Go plugin build 命令构建 `cryptoupgrade/algorithm/go` 下的每个候选文件。
- [x] 5.3 使用 `solc-0.8.26 --optimize --via-ir` 编译 Solidity 算法合约，如果本机缺少 solc 则记录原因。
- [x] 5.4 运行相关 Go 测试和 `openspec validate refactor-algorithm-interface`。
