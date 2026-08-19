## 1. 功能审计

- [x] 1.1 复核 `cryptoupgrade/contracts/code_storage.sol` 的字段、事件和函数，确认是否支持算法版本、指定区块生效和立即切换。
- [x] 1.2 复核 `common.CodeStorageABI_json` 与 Solidity 合约是否一致，记录需要新增或兼容保留的 ABI 项。
- [x] 1.3 复核 `cryptoupgrade/internal/evm/code_storage.go` 和 `core/vm/contracts_cryptoupgrade.go`，确认 `callFunc` 是否能按 EVM block number 选择版本。
- [x] 1.4 复核 `cryptoupgrade/internal/event`、`cryptoupgrade/internal/activation` 和 repository，确认事件激活是否能按版本保存本地 artifact。
- [x] 1.5 复核 `experiments/cryptoupgrade/bench/cmd/benchupgradelatency` 与多节点网络工具，明确可复用的 RPC、fixture 和结果输出逻辑。
- [x] 1.6 输出审计记录，列出 `supported`、`partial`、`missing` 能力及对应实现任务。

## 2. 合约与 ABI 版本管理

- [x] 2.1 扩展 `CodeStorage` 合约数据结构，保存算法名、版本号、源码 metadata、gas、ABI 类型和 `activationBlock`。
- [x] 2.2 增加或兼容包装版本化升级入口，支持指定区块生效和立即切换两类策略。
- [x] 2.3 增加版本查询接口，返回指定版本、当前生效版本和版本计划 metadata。
- [x] 2.4 扩展升级事件字段，至少包含算法名、版本号和 `activationBlock`。
- [x] 2.5 同步更新 `common.CodeStorageABI_json`、ABI 解析测试和旧入口兼容测试。

## 3. Geth 调用边界与本地版本状态

- [x] 3.1 修改 CodeStorage EVM 调用边界，将当前 EVM block number 传入 dispatcher。
- [x] 3.2 在 dispatcher 中实现按 block number 解析当前算法版本的逻辑。
- [x] 3.3 扩展 `model.AlgorithmInfo` 或新增版本 metadata 类型，避免单版本字段与多版本状态混用。
- [x] 3.4 扩展 repository，使本地源码、`.so` 和 metadata 按算法名与版本号隔离保存。
- [x] 3.5 保持旧 `callFunc(name,input)` 调用兼容，并让它按当前区块选择链上应生效版本。

## 4. 事件激活与失败处理

- [x] 4.1 更新事件服务，解析版本化升级事件并查询对应版本 metadata。
- [x] 4.2 更新 activation service，支持提前编译未来生效版本但不决定链上生效语义。
- [x] 4.3 在本地激活失败时记录算法名、版本号、区块号和错误原因。
- [x] 4.4 增加事件去重测试，确认重复事件不会重复编译同一算法版本。
- [x] 4.5 增加失败恢复测试，确认新版本激活失败不会污染旧版本 metadata。

## 5. 升级稳定性实验命令

- [x] 5.1 新增 `experiments/cryptoupgrade/bench/cmd/benchupgradestability` 命令、flags 和结果目录初始化。
- [x] 5.2 复用多节点 YAML 配置解析和前置网络检查，确认 RPC、chain ID、peer、出块和目标节点列表。
- [x] 5.3 实现 scheduled 升级轮次，提交未来 `activationBlock` 并采样生效区块前后的版本和输出。
- [x] 5.4 实现 immediate 升级轮次，提交立即切换交易并验证后续区块版本和输出一致。
- [x] 5.5 采集所有目标节点的 head、block hash、receipt、event 字段、版本视图和 `callFunc` 输出。
- [x] 5.6 输出 JSON、CSV 和文本摘要，明确通过状态、失败维度、失败节点和观测高度。

## 6. 验证与实验论证

- [x] 6.1 为合约 ABI、dispatcher 版本选择、repository 版本隔离和事件解析增加聚焦 Go 测试。
- [x] 6.2 对新增和修改 Go 代码运行 `gofmt`。
- [x] 6.3 运行相关 `go test`，至少覆盖 `cryptoupgrade/...`、`core/vm` 中的 cryptoupgrade 测试和新增实验 helper。
- [x] 6.4 运行 `openspec validate lab3-upgrade-stability`。
- [x] 6.5 在本地 2 节点 Clique 私有链运行 scheduled 与 immediate smoke test。
- [x] 6.6 在实验 2 可复用的私有链规模上运行正式稳定性实验，保存原始 JSON/CSV 和摘要。
- [x] 6.7 整理实验结论，说明升级交易是否影响链一致性、版本一致性和算法输出一致性。
