## 为什么

当前 Go 候选算法和 Solidity 对照算法的目录、文件名、合约名和导出入口不完全一致，且部分文件在同一文件中暴露多个算法入口，不利于后续自动化 benchmark、结果归档和跨实现一致性检查。需要统一为“一个算法文件只暴露一个主入口”，并让 Go 文件与 Solidity 文件使用同一算法名称。

## 变更内容

- **BREAKING**：将 Go 候选算法源码统一放到 `cryptoupgrade/algorithm/go`。
- **BREAKING**：将 Solidity 算法合约统一放到 `cryptoupgrade/algorithm/contracts`。
- 统一文件命名：例如 `add.go` 对应 `Add.sol`，两者都表达同一个算法操作。
- 每个 Go 算法文件只保留一个导出的算法入口；内部 helper 保持非导出。
- 每个 Solidity 算法合约只暴露一个用于 benchmark 的外部算法入口；helper 合约或内部函数不作为算法入口统计。
- 对 Solidity 当前未实现或实现困难的算法，将 Go 文件放入 `cryptoupgrade/algorithm/go/archive`，不再保留占位 Solidity 文件。
- 更新 benchmark/文档默认路径，使其指向新目录。
- 增加一致性检查，统计 Go 算法与 Solidity 算法的名称匹配和操作语义匹配情况。

## 能力

### 新增能力

- `cryptoupgrade-algorithm-interface`：定义候选算法源码目录、Go/Solidity 命名对应关系、单一暴露入口和一致性检查要求。

### 修改能力

- `cryptoupgrade-performance-benchmarks`：benchmark 输入路径和 Solidity 合约选择 SHALL 使用新的算法目录与命名规则。

## 影响

- 影响代码：`cryptoupgrade/algorithm/go/*.go`、`cryptoupgrade/algorithm/go/archive/*.go`、`cryptoupgrade/algorithm/contracts/*.sol`、`cryptoupgrade/cmd/*` 中的默认路径。
- 影响文档：`cryptoupgrade/docs/` 中引用旧算法路径的命令示例和结果说明。
- 影响 OpenSpec：新增算法接口规范，并更新 benchmark 相关 delta spec。
- 不新增外部依赖。
