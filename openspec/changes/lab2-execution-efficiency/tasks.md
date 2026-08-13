## 1. 实验入口准备

- [x] 1.1 复核 Add 三类实现入口：`cryptoupgrade/algorithm/go/add.go`、`cryptoupgrade/algorithm/contracts/Add.sol` 和 `common.CryptoUpgradeAddAddress`。
- [x] 1.2 确认 Add 默认输入、ABI 类型和期望输出，保证动态升级、contract 和 precompile 三条路径可比较。
- [x] 1.3 确认本地 RPC、sender、solc 路径和结果输出目录等实验运行参数。

## 2. 三组调用路径实现

- [x] 2.1 新增 `cryptoupgrade/bench/cmd/benchexecutionefficiency` 命令和 Add-only flags。
- [x] 2.2 实现动态升级路径 setup，并在升级完成后构造 `CodeStorage.callFunc("Add", encodedInput)` calldata。
- [x] 2.3 实现 contract Add 合约编译、部署或复用地址，并构造 `Add(int256,int256)` calldata。
- [x] 2.4 实现 precompile Add 地址解析，并构造 `uint256,uint256` ABI calldata。
- [x] 2.5 对三类实现执行首次调用校验，输出不一致时失败。

## 3. 指标采集与结果格式

- [x] 3.1 为三类实现分别执行 warmup 和 `n` 次 `eth_call`，采集 first、mean、p50、p95、min、max 调用耗时。
- [x] 3.2 为三类实现分别执行 `eth_estimateGas`，将结果记录为 `gasEstimate`。
- [x] 3.3 输出 JSON 原始结果，包含算法、输入、ABI 类型、地址、calldata size、时间统计、Gas 估算、输出校验状态和比值。
- [x] 3.4 输出简明文本摘要，直接展示 Add 三类实现的调用时间与 Gas 消耗对比。
- [x] 3.5 确保 setup 交易、部署地址和上传状态只作为复现信息记录，不进入升级后执行效率结论。

## 4. 验证

- [x] 4.1 对新增 Go 代码运行 `gofmt`。
- [x] 4.2 对新增命令运行聚焦构建或测试检查。
- [x] 4.3 运行 `openspec validate lab2-execution-efficiency`。

## 5. Add 实验结果反馈

- [x] 5.1 在本地 Geth 测试链上运行 Add 执行效率实验。
- [x] 5.2 检查 JSON 原始结果和文本摘要，确认调用时间、Gas 估算和输出一致性字段完整。
- [x] 5.3 向用户反馈 Add 三类实现资源消耗结果，并等待确认后再扩展后续算法实验。

## 6. Precompile 命名同步

- [x] 6.1 将项目内实验和 precompile registry 相关的 `native` 命名统一改为 `precompile`。
- [x] 6.2 同步更新 VM 集成、测试和实验命令中的 precompile API、文件名和输出标识。

## 7. 已有算法三方对比

- [x] 7.1 为已有算法中三方均可调用的实现增加 benchmark fixture。
- [x] 7.2 对不同 ABI 返回类型进行解码和逻辑归一化，确保三类实现按输出语义比较。
- [x] 7.3 对缺少三方可比实现的算法记录 skipped 原因。

## 8. 批量实验输出

- [x] 8.1 扩展 benchmark JSON 与 CLI 摘要，按算法输出 `upgrade`、`precompile` 和 `contract` 三类结果。
- [x] 8.2 保持 setup 阶段不进入调用延迟统计，并支持每个算法重复测量。

## 9. 验证与结果反馈

- [x] 9.1 对重命名和新增 Go 代码运行 `gofmt` 与聚焦测试。
- [x] 9.2 运行 `openspec validate lab2-execution-efficiency`。
- [x] 9.3 在本地 Geth dev 链上运行已有可对齐算法的执行效率实验。
- [x] 9.4 向用户反馈三类方案的实测资源消耗结果，等待确认后再完成后续实验。
