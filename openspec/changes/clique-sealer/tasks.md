## 1. Multi-Node Interface Audit

- [x] 1.1 复核 `openspec/changes/add-multi-node-network/design.md`、spec 和 tasks，列出 Clique signer 需要满足的 YAML、启动命令、RPC 和 validate 接口。
- [x] 1.2 复核 `cryptoupgrade/network.Config`、`GethArgs`、`ComposeYAML`、`ValidateNetwork` 和 `local-2nodes.yaml`，确认 signer/observer 的现有字段和生成命令。
- [x] 1.3 复核当前 geth CLI flags、account manager、keystore、RPC API 注册和 `consensus/clique.Seal` 当前行为，标记 multi-node 生成命令中会触发 unknown flag 或缺失 API 的位置。

## 2. Clique Engine Sealing

- [x] 2.1 恢复 Clique signer callback 类型，并让 `Authorize` 保存 signer 地址和签名函数。
- [x] 2.2 实现 `Clique.Seal`，覆盖授权检查、recent signer 检查、in-turn/out-of-turn delay、stop channel 和 header extra-data 签名写入。
- [x] 2.3 保持 `Prepare`、`CalcDifficulty`、`SealHash` 和 header 校验路径兼容，确保 observer/rpc 节点可验证 Clique 区块。
- [x] 2.4 添加 Clique engine 单元测试，验证授权 signer 可签名、未授权 signer 被拒绝、缺少签名函数时报错、stop channel 可中断。

## 3. Geth Startup Contract

- [x] 3.1 补齐 geth signer 启动参数契约，支持 multi-node signer command 所需的账户解锁、password 文件、insecure unlock 兼容开关和 mining/Clique signer 模式。
- [x] 3.2 处理 miner fee recipient/etherbase 参数兼容，确保 signer 节点未显式设置时默认使用 node account 作为区块执行上下文地址。
- [x] 3.3 如果最终 flag 名称与 `cryptoupgrade/network.GethArgs` 当前输出不一致，同步更新 `GethArgs`、Docker Compose 生成和文档，避免生成 unknown flag。
- [x] 3.4 在节点启动时使用 keystore/account manager 和 password 文件解锁 signer 账户，并将 wallet signing function 注册到 inner Clique engine，兼容 beacon-wrapped Clique engine。
- [x] 3.5 确保 `role: observer` 和 `role: rpc` 生成的节点不解锁 signer，不注册 signing function，也不会启动本地 sealing。
- [x] 3.6 添加启动配置测试或 CLI 测试，覆盖 signer command 可解析、observer command 可解析、signer 解锁失败时启动失败。

## 4. Local Clique Sealing Driver

- [x] 4.1 在 `miner` 或 `eth` 生命周期中新增最小本地 Clique sealing goroutine，只在 Clique 链且 signer 模式启用时运行。
- [x] 4.2 复用现有 work generation 构建候选区块，调用 consensus engine `Seal` 获取 sealed block。
- [x] 4.3 将 sealed block 通过本地 blockchain 导入，并对可恢复错误进行日志记录和下一轮重试。
- [x] 4.4 使用 Clique period 控制出块间隔，并在节点停止时可靠退出 sealing goroutine。
- [x] 4.5 添加连续出块测试，验证 signer 节点在短 period Clique 链上能产生多个区块。

## 5. Clique RPC And Validation Contract

- [x] 5.1 注册 validate 所需的最小 Clique RPC 服务，至少支持 `clique_getSigners` 查询 latest 授权 signer。
- [x] 5.2 确保 HTTP API 中包含 `clique` 时，signer/observer/rpc 节点都能响应 signer 状态查询。
- [x] 5.3 更新或补充 `cryptoupgrade/network.ValidateNetwork` 测试，确认 chain ID、peer count、区块高度增长和 signer 查询字段符合 JSON 输出契约。
- [x] 5.4 将 `cryptoupgrade/network/clique_smoke_test.go` 从“预期 panic”更新为真实出块或 sealing 成功 smoke test。

## 6. Docker Multi-Node Smoke Test

- [x] 6.1 使用 `cryptoupgrade/cmd/multinode -mode render` 生成 `local-2nodes.yaml` 的 genesis、static peers、start scripts 和 Docker Compose。
- [x] 6.2 使用生成的 node1 command 验证 signer 节点可解锁账户并持续出块。
- [x] 6.3 使用生成的 node2 command 验证 observer 节点不解锁账户且能同步 node1 新区块。
- [x] 6.4 在可用 Docker 环境中运行 compose 双节点网络，确认 node1 持续出块、node2 同步、`clique_getSigners` 返回授权 signer。（当前环境有 Docker daemon 但无 `docker compose` 插件，已使用等价的 `sudo docker run` 双容器验证）
- [x] 6.5 运行可选 cryptoupgrade Add smoke test，确认共识网络能支撑后续动态升级实验交易确认。

## 7. Verification

- [x] 7.1 运行 `go test ./consensus/clique ./miner ./eth/... ./cryptoupgrade/network` 或更小的相关包测试，并记录结果。
- [x] 7.2 运行 `go run ./cryptoupgrade/cmd/multinode -mode validate`，记录 node1/node2 高度变化和 signer 查询结果。
- [x] 7.3 运行 `openspec validate clique-sealer` 和 `openspec validate add-multi-node-network`，根据结果修正文档、spec 或 tasks。
