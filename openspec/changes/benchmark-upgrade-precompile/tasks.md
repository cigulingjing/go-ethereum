## 1. Benchmark 命令结构

- [x] 1.1 决定扩展 `cryptoupgrade/cmd/benchcandidate`，或新增聚焦命令 `cryptoupgrade/cmd/benchrealchain`。
- [x] 1.2 增加默认 RPC 端点 `http://127.0.0.1:8666`，并提供可覆盖的 `-rpc` flag。
- [x] 1.3 增加算法名、源码文件、输入 ABI 类型、输出 ABI 类型、ABI 编码输入、期望输出、native precompile 地址、sender、warmup 数量、样本数量和输出 JSON 路径等 flags。
- [x] 1.4 增加 supported mode 和 required parameter 校验，并不改变现有 benchmark 命令。

## 2. Upgrade 和 Precompile 执行路径

- [x] 2.1 通过 `CodeStorage.uploadCode` 实现 upgrade setup，并记录 transaction hash、receipt status、elapsed time、gas used、source size 和 compressed size metrics。
- [x] 2.2 通过 `CodeStorage.callFunc(name, encodedInput)` 生成 upgrade call data。
- [x] 2.3 使用直接发送到已配置 precompile 地址的 ABI 编码参数生成 native precompile call data。
- [x] 2.4 将 native precompile setup metrics 记录为零部署交易和零部署 gas，同时保留所选地址。

## 3. 测量和校验

- [x] 3.1 为两种方案实现 warmup 和重复 `eth_call` 计时。
- [x] 3.2 为每种方案计算 first、mean、p50、p95、min 和 max 延迟统计。
- [x] 3.3 使用 `eth_estimateGas` 或等价模拟路径估算两种方案的调用 gas。
- [x] 3.4 使用解码输出或规范 ABI 编码输出比较 upgrade 和 precompile 结果，并在不匹配时失败。
- [x] 3.5 在数值非零时计算 latency 和 gas 的 upgrade/precompile 比值。

## 4. 结果输出和文档

- [x] 4.1 打印简明的人类可读摘要，包括 setup cost、call latency、gas estimate 和 ratios。
- [x] 4.2 写入机器可读 JSON 结果，包含 endpoint、sender、algorithm、source path、ABI types、input、precompile address、setup metrics、call metrics、gas metrics、ratios 和 validation status。
- [x] 4.3 在 `cryptoupgrade/docs/` 下增加本地 `8666` geth 链的精确命令、iteration count、input value 和 expected output。
- [x] 4.4 如果本地链可用，则为至少一个算法 fixture 增加或更新样例结果制品。

## 5. 验证

- [x] 5.1 对修改的 Go 文件运行 `gofmt` 和 `goimports`。
- [x] 5.2 对 benchmark 命令包运行目标测试或编译检查。
- [x] 5.3 在本地 geth 链可用时，对 `http://127.0.0.1:8666` 运行 benchmark 并记录命令/结果。
- [x] 5.4 运行 `openspec validate benchmark-upgrade-precompile`。
