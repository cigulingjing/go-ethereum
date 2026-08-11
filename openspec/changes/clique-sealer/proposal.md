## Why

`add-multi-node-network` 已经定义了基于 YAML、Docker Compose、static peers 和 validate 命令的多节点私有链部署能力，但其中 Clique signer 节点仍缺少真实可用的共识出块服务。没有这个运行时能力，Docker 多节点网络只能完成配置渲染和节点互联，不能证明网络具备持续共识服务能力。

本 change 解决 `add-multi-node-network` 所需的 Clique 共识运行时接口，使 node1 能按网络配置使用已解锁 signer 账户签名区块，node2 能通过 P2P 同步该链，并让现有 multi-node validate 流程能够验证出块、同步和 signer 状态。

## What Changes

- 恢复或实现最小 Clique sealer，使 Clique 共识引擎能够驱动授权 signer 生产新区块。
- 将 signer 节点已解锁账户接入 Clique 签名流程，避免单独维护与节点账户状态脱节的签名入口。
- 兼容 `add-multi-node-network` 的 YAML 和渲染接口，包括 `consensus.type/period/epoch/signers`、节点 `role/account/keystore/password`、Docker 启动命令和必要 RPC namespace。
- 恢复或补齐 multi-node 生成命令需要的 geth 启动契约，使 signer 节点可以通过渲染出的 Docker Compose 或 `start.sh` 直接启动出块。
- 提供可复现的双节点验证流程，确认 node1 能持续出块，node2 能同步到 node1 产生的区块，并与 `cryptoupgrade/cmd/multinode validate` 输出兼容。
- 保持非 Clique 链、非 signer 节点、EVM 执行语义和 cryptoupgrade 现有行为不变。

## Capabilities

### New Capabilities

- `clique-sealer`: 定义 Clique signer 解锁账户接入、区块签名、持续出块和双节点同步验证要求。

### Modified Capabilities

- None.

## Impact

- Affected code: Clique consensus engine 的 sealing 路径、node/miner 启动参数接入、账户解锁与 signer 查找逻辑、Clique RPC 查询接口、`cryptoupgrade/network` 启动参数渲染和 validate 兼容验证。
- Affected inputs: `add-multi-node-network` YAML 中的 Clique consensus 配置、node role、signer account、keystore/password、nodekey/static peers、Docker 环境变量和 HTTP RPC namespace。
- Affected systems: 本地双节点私有链运行目录、Docker Compose 多节点网络、P2P 同步、RPC 出块高度查询、`clique_getSigners` 查询和后续多节点实验环境。
- No breaking changes to non-Clique consensus behavior, existing cryptoupgrade algorithm loading, native precompile registration, Solidity benchmarks, or single-node development workflows.
