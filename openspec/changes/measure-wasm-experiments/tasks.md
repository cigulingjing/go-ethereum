## 1. 规划验证

- [x] 1.1 运行 `openspec validate --changes measure-wasm-experiments`，确认 proposal/spec/design/tasks 可解析

## 2. 节点内 WASM 激活 Trace

- [x] 2.1 新增节点内 activation trace 写入能力，并用单元测试验证 JSONL 字段和禁用状态
- [x] 2.2 在 event、activation 和 wasmruntime 阶段写入事件处理开始、WASM 持久化、编译完成、实例化完成和激活完成 trace，并用现有单元测试验证
- [x] 2.3 扩展 `benchupgradelatency` 聚合 trace 到 JSON/CSV，并用测试验证按算法名、版本和 tx hash 关联

## 3. Docker 资源与规模网络

- [x] 3.1 为网络 YAML 和 Compose 渲染增加统一 CPU/内存限制，并用渲染测试验证每个节点配置一致
- [x] 3.2 增加或生成 20、30、40 节点网络配置与 node key，并用 `multinode render` 验证配置可用

## 4. WASM-only Lab2

- [x] 4.1 新增单节点 WASM-only 执行效率命令，复用算法 fixture、只调用 `CodeStorage.callFunc`、不输出 Gas，并用聚焦构建验证
- [x] 4.2 增加 Docker CPU/内存采样与 summary 输出，并用解析测试验证 stats 格式

## 5. 实验编排与报告

- [x] 5.1 新增实验 runner，按 20 节点全算法、SchnorrProof 5/10/20/30/40、单节点 WASM-only Lab2 的顺序执行，并保存命令、日志和 Docker 状态
- [x] 5.2 新增报告生成与 `paper/outline.md` 审计逻辑，并用 fixture 结果测试报告包含支撑度结论

## 6. 本地验证

- [x] 6.1 运行 `gofmt` 和相关 Go 单元测试，确认新增实验代码通过
- [x] 6.2 重新构建 `cryptoupgrade-geth:lab` 镜像，确认 Docker build 使用当前 WASM 代码
- [x] 6.3 按顺序执行 20 节点全算法升级实验、SchnorrProof 多规模升级实验和单节点 WASM-only 执行效率实验，保存原始结果
- [x] 6.4 生成任务执行报告并审计当前实验数据是否支撑 `paper/outline.md`
- [x] 6.5 运行 `openspec validate --changes measure-wasm-experiments`，确认最终规格一致
