## 1. 实验流程梳理

- [x] 1.1 复核 `chain_setup.md` 指定的 dev 链启动命令和本地目录前置条件。
- [x] 1.2 梳理并记录现有 `CodeStorage.uploadCode`、receipt、activation 和 `CodeStorage.callFunc` 的真实执行流程。

## 2. Add-only 实验命令

- [x] 2.1 新增独立 `cryptoupgrade/cmd/benchupgradeefficiency` 命令结构和 flags。
- [x] 2.2 实现按 `chain_setup.md` 命令启动 `geth --dev`、保存日志、等待 RPC ready 和退出清理。
- [x] 2.3 实现 Add 源码压缩、`uploadCode` calldata 构造、升级交易提交和 receipt 轮询。
- [x] 2.4 实现 activation observation、`callFunc("Add", encodedInput)` 验证和 Add 返回值校验。
- [x] 2.5 实现 JSON 原始结果输出，覆盖区块号、阶段时间、延迟、payload size、gas used、日志路径和观测限制。

## 3. 验证与单次结果

- [x] 3.1 对新增 Go 代码运行 `gofmt` 并完成编译检查。
- [x] 3.2 运行 `openspec validate benchmark-upgrade-efficiency`。
- [x] 3.3 使用新增命令启动本地 dev 链并完成一次 Add 升级效率实验。
- [x] 3.4 检查 JSON 原始结果和 Geth 日志，确认指标定义、路径和无法精确测量项均已记录。
