## Context

`add-multi-node-network` 已经在 `cryptoupgrade/network` 中定义了多节点私链的配置和部署接口：YAML 描述 `network`、`consensus`、`accounts` 和 `nodes`，render 生成 genesis、static peers、节点目录、`start.sh` 和 Docker Compose，validate 通过 RPC 检查 chain ID、peer count、区块高度增长、`clique_getSigners` 和可选 cryptoupgrade smoke test。

当前缺口在运行时共识层。仓库仍保留 `consensus/clique` 的 header 校验、snapshot、投票和 `SealHash` 逻辑，但 `Clique.Seal` 直接 panic，不能生产 Clique 区块。普通 geth 启动路径也缺少 multi-node 生成命令期望的账户解锁、mining/sealing 和 Clique RPC 闭环。因此本 change 不是孤立恢复一个单节点 sealer，而是为 `add-multi-node-network` 提供可被 Docker 多节点网络直接调用的 Clique 共识服务。

## Goals / Non-Goals

**Goals:**

- 恢复 `consensus/clique.Clique.Seal`，使已授权 signer 能对候选 header 生成可验证的 Clique 签名。
- 将 `add-multi-node-network` 中 signer 节点的 `account`、`keystore` 和 `password` 接入 geth 账户解锁与 Clique 签名流程。
- 支持 multi-node render/compose 生成的 signer 启动命令，使 `role: signer` 节点启动后自动提供 Clique 出块服务。
- 保证 `role: observer` 和 `role: rpc` 节点不解锁 signer 账户，仅通过 P2P 同步并验证 Clique 区块。
- 提供 `clique_getSigners` 等 validate 需要的最小 Clique RPC 查询接口。
- 让 `cryptoupgrade/cmd/multinode validate` 能验证 node1 出块、node2 同步、peer count、chain ID 和 signer 状态。

**Non-Goals:**

- 不把本 change 扩展为通用云部署或 Docker 编排系统；Docker/YAML 仍由 `add-multi-node-network` 维护。
- 不实现 Clique signer 投票治理 UI、动态 signer 变更、rollback 或多 signer 容错策略。
- 不引入外部共识服务、beacon/Engine API 替代方案或新的第三方依赖。
- 不改变非 Clique 网络、PoS payload 构建、EVM 执行语义或 cryptoupgrade 算法路径。

## Decisions

1. 以 `add-multi-node-network` 的 YAML 和渲染输出作为外部接口契约

   `clique-sealer` 的实现必须消费或兼容以下配置语义：`consensus.type=clique`、`consensus.period`、`consensus.epoch`、`consensus.signers`，以及节点 `role`、`account`、`keystore`、`password`、`httpApis` 和 `extraArgs`。`cryptoupgrade/network.GethArgs` 生成的 signer 命令必须可以直接启动可出块节点；observer/rpc 命令必须可以直接启动同步节点。

   Alternative considered: 为 Clique sealer 设计一套独立配置。该方案会让多节点工具和共识运行时出现两套入口，破坏实验复现性。

2. 恢复 Clique engine 内部 sealing，而不是在外部手写 header 签名

   `Clique.Seal` 复用现有 snapshot、`SealHash`、recent signer 和 difficulty 规则，负责检查本地 signer 是否被授权、是否因近期签名暂时不可出块、计算 in-turn/out-of-turn delay，并把 65 字节签名写入 `header.Extra` 末尾。这样 node2 导入区块时仍通过同一 Clique 校验路径证明区块合规。

   Alternative considered: 在 miner 或脚本中直接对 header hash 签名并插入区块。该方案会绕过 Clique engine 的授权和轮次规则。

3. Clique `Authorize` 保存 signer 地址和账户签名回调

   恢复 signer callback 类型，由已解锁 wallet 或 keystore 实现签名。`Authorize` 同时记录 signer address 和 sign function；仅设置地址但没有 sign function 时，`Prepare` 和 difficulty 计算仍可工作，但 `Seal` 必须返回明确错误。这样能把 signer 节点的账户状态接入共识签名流程，而不是让 Clique 包直接读取私钥。

   Alternative considered: 在 Clique engine 中直接加载 keystore 私钥。该方案会把账户管理和共识强耦合，扩大 Geth 原有代码侵入面。

4. 补齐 geth 启动契约，并与 multi-node 渲染器保持单一事实来源

   `add-multi-node-network` 当前设计和文档期望 signer 节点通过 geth 启动参数解锁账户、启用 mining 或等价 Clique signer 模式，并暴露 `clique`、`admin`、`eth`、`net`、`miner` 等 RPC namespace。实现阶段必须同步处理两侧接口：要么让 geth 接受渲染器当前输出的兼容参数，例如 `--unlock`、`--password`、`--allow-insecure-unlock`、`--mine`、`--miner.pending.feeRecipient` 及必要兼容别名；要么更新 `cryptoupgrade/network.GethArgs`、示例 YAML 和文档，使其只输出最终支持的参数。完成后不得留下 Docker command 中存在未知 flag 的状态。

   Alternative considered: 只新增内部配置字段，让用户手工修改 Docker command。该方案无法满足多节点实验的可复现性。

5. 在本地节点生命周期内实现最小 Clique sealing driver

   新增受控 goroutine：按当前 head 和 Clique period 构建候选区块，调用 consensus engine `Seal`，收到 sealed block 后通过 `BlockChain.InsertChain` 导入本地链。driver 只在 Clique 链、signer 账户已接入并显式启用 signer 模式时运行；停止节点时必须退出。

   Alternative considered: 通过外部 RPC 不断调用 Engine API 构建 payload。Clique 是 execution-layer PoA 共识，不需要外部 CL；使用 Engine API 会引入与本实验无关的 post-merge 控制面。

6. 提供 validate 所需的最小 Clique RPC

   `cryptoupgrade/network.ValidateNetwork` 调用 `clique_getSigners` 检查授权 signer。实现应注册最小 Clique RPC 服务，至少支持 latest signer 查询，并尽量复用 existing snapshot 逻辑。RPC 不应允许 observer 节点执行本地签名或解锁账户。

7. 双节点验证复用 multi-node 工具

   验证不再维护独立脚本为主，而是优先使用 `cryptoupgrade/cmd/multinode -mode render/init/validate` 和 `cryptoupgrade/examples/networks/local-2nodes.yaml`。通过条件为：node1 在两个以上 Clique period 内高度增长；node2 peer count 非零；node2 高度最终等于或接近 node1；`clique_getSigners` 返回 genesis signer；可选 Add smoke test 仍能通过 signer RPC 发送交易。

## Risks / Trade-offs

- [Risk] 当前 multi-node 渲染器和 geth CLI flag 名称已经不完全一致。→ Mitigation: 本 change 将启动参数兼容性列为实现任务，必须通过 render/compose smoke test 验证无 unknown flag。
- [Risk] 当前 beacon wrapper 可能把某些 header 当作 PoS header 跳过 Clique `Seal`。→ Mitigation: Clique work generation 必须生成带 Clique difficulty 的非 PoS header，并在测试中覆盖 beacon-wrapped Clique engine 的 `Seal` 委派路径。
- [Risk] signer 未在 genesis `extraData` 授权时，node1 启动正常但不能出块。→ Mitigation: 启动时检查或在第一轮出块错误中记录 unauthorized signer；validate 检查 genesis signer 列表与 node1 signer 地址一致。
- [Risk] 密码文件或 keystore 配置错误会导致 signer 无法解锁。→ Mitigation: signer 模式下将解锁失败作为启动失败处理，observer/rpc 节点不执行解锁逻辑。
- [Risk] 单 signer Clique 没有容错能力。→ Mitigation: 本 change 明确服务最小实验闭环；后续多 signer 和投票治理可通过独立 change 扩展。

## Migration Plan

1. 先恢复 `Clique.Seal` 并补齐单元测试，确认授权 signer 能签出可验证 header。
2. 补齐 geth 账户解锁、signer 授权和 CLI/render 参数契约，使 multi-node signer command 能直接启动。
3. 增加本地 sealing driver，并用短周期 Clique genesis 测试连续出块。
4. 注册最小 Clique RPC，确保 `clique_getSigners` 满足 validate。
5. 使用 `cryptoupgrade/cmd/multinode` 的 render/init/validate 流程验证 Docker 双节点网络。

Rollback 策略：不启用 signer/mine 模式时节点仍作为普通 observer/full node 运行；render 输出可移除 signer flags 回到只同步网络。

## Open Questions

- 最终外部启动参数保留旧式 `--mine/--unlock` 兼容，还是改为更显式的 `--clique.seal/--clique.signer` 并同步更新 multi-node 渲染器。
- `clique_getSigners` 是否只实现 validate 需要的 latest 查询，还是同时恢复 snapshot/status/proposals 等完整 Clique RPC。
