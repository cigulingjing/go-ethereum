## Context

当前 cryptoupgrade 功能已经可以在本地 `geth --dev` 或单节点私链上完成动态算法上传、调用和实验数据收集。这个环境足以验证 EVM 调用边界和算法实现成本，但缺少多节点部署、节点互联、共识出块和服务器复现实验能力。

现有代码中，cryptoupgrade 插件目录已经支持通过 `GETH_CRYPTOUPGRADE_PLUGIN_DIR` 做进程级隔离；实验命令集中在 `cryptoupgrade/cmd/`；仓库已有通用 `Dockerfile` 和 `Dockerfile.alltools`。需要注意的是，Go plugin 动态编译依赖 Go toolchain、CGO 和与 geth 兼容的源码模块环境，最小运行时镜像不能直接满足升级实验。另一个约束是当前 `consensus/clique.Seal` 标记为不再支持出块，多节点 Clique 方案必须先验证或恢复实验用 signer 路径。

## Goals / Non-Goals

**Goals:**

- 使用 YAML 描述多节点私有链网络，包括节点 ID、host/IP、端口、数据目录、插件目录、账户和角色。
- 根据配置生成可复现的 genesis、static peers、节点运行目录和启动命令。
- 提供 Docker 镜像和 Docker Compose 入口，在一台或多台服务器上拉起多个 Geth 节点并互联。
- 提供 Clique signer 节点 profile，使私有链能够由授权账户出块并通过共识认证。
- 保持现有 cryptoupgrade 上传、调用、precompile 和 Solidity 对照实验语义不变。

**Non-Goals:**

- 不把多节点网络扩展为通用云平台编排系统。
- 不在本 change 中实现跨节点升级性能实验或复杂网络故障注入。
- 不修改动态密码算法 ABI、CodeStorage 合约语义、precompile gas 规则或算法生命周期。
- 不要求 YAML 保存私钥明文；配置只引用 keystore、password 文件或 nodekey 文件路径。

## Decisions

1. 新增独立的私链网络辅助命令和配置包

   在 `cryptoupgrade/cmd/` 下新增多节点网络辅助命令，并在 `cryptoupgrade/network` 或同级内部包中实现 YAML 解析、校验和 artifact 渲染。这样可以复用现有实验命令组织方式，同时避免把实验部署逻辑塞入 `cmd/geth` 主入口。

   Alternative considered: 直接给 `geth` 增加 `--network-config`。该方案会侵入 Geth 原有 CLI 和节点启动路径，不符合最小侵入原则。

2. YAML 配置作为网络拓扑的单一输入

   配置文件包含 `network`、`consensus`、`docker`、`accounts` 和 `nodes` 几类信息。节点字段至少覆盖逻辑 ID、advertise host/IP、P2P/RPC 端口、datadir、pluginDir、nodekey、角色和 signer account。工具在内存中归一化默认值，并在输出目录写入实际使用的配置快照，保证实验可复现。

   建议结构示例：

   ```yaml
   network:
     name: cryptoupgrade-lab
     chainId: 11223344
     networkId: 11223344
   consensus:
     type: clique
     period: 5
     epoch: 30000
     signers:
       - 0xF5F871aA6Bd253914705898c66251f994aa426FA
   docker:
     image: cryptoupgrade-geth:lab
     networkName: cryptoupgrade-lab
   nodes:
     - id: node1
       role: signer
       host: 10.0.0.11
       p2pPort: 30303
       httpPort: 8666
       datadir: ./data/node1
       pluginDir: ./plugin/node1
       nodeKey: ./keys/node1.key
       account: 0xF5F871aA6Bd253914705898c66251f994aa426FA
       keystore: ./keystore/node1
       password: ./passwords/node1.txt
   ```

3. Genesis 生成以模板合并为主

   生成器负责写入 chain/network ID、Clique `period`/`epoch`、按 signer 列表生成 `extradata`，并合并 funded account 与已有 cryptoupgrade 预部署合约 alloc。对于已经手工维护的 genesis，工具应提供校验模式，确认 Clique signer、chain ID、CodeStorage 地址和 fork 配置满足实验要求。

   Alternative considered: 完全手写固定 genesis。该方案容易把 signer 地址、chain ID 和预部署合约写死，后续多服务器复现实验成本较高。

4. Docker 使用实验镜像和最小镜像分离

   保留现有最小 `Dockerfile` 用于普通 geth 镜像；新增或扩展实验用 Dockerfile，保留 Go toolchain、gcc、源码模块和构建缓存，并设置 `CRYPTOUPGRADE_MODULE`、`GETH_CRYPTOUPGRADE_PLUGIN_DIR` 等运行时环境。这样可以保证容器内 `CodeStorage.uploadCode` 触发的 Go plugin 编译与当前 geth 二进制兼容。

   Alternative considered: 容器启动时挂载宿主机 Go toolchain。该方案对服务器环境依赖过强，不利于复现实验。

5. Docker Compose 作为单服务器默认编排入口

   渲染出的 compose 文件使用同一个 Docker network、固定容器名、独立 volume/datadir/pluginDir、明确端口映射和每个节点的 geth 启动命令。多服务器部署则复用同一 YAML 中的 `host`/`advertiseHost` 生成 enode 和启动参数，不强依赖 compose 跨主机能力。

6. Clique signer 作为节点角色而不是独立共识协议改造

   YAML 中 `role: signer` 的节点负责解锁授权账户、启用 mining、暴露必要的 `clique`/`admin`/`eth` RPC，并被写入 genesis signer 列表。observer/rpc 节点只同步和服务请求，不解锁 signer 账户。

   当前仓库的 Clique engine 可以验证 Clique header，但 `Seal` 路径不可用。实现阶段应先增加 Clique 出块 smoke test；如果仍触发 `consensus/clique.Seal` panic，则恢复一个仅对 Clique chain config 生效的实验用 sealer，复用现有 `Authorize`、snapshot、`SealHash` 和 signer 校验逻辑，并用测试约束不会影响非 Clique 网络。

7. 验证命令必须覆盖网络、共识和 cryptoupgrade 基础路径

   验证入口读取同一 YAML 和渲染输出，检查 RPC 可达、chain ID 一致、peer count 满足配置、区块高度在 Clique period 后增长、signer 状态可查询，并可选执行 `Add` 算法上传/调用 smoke test。验证结果输出 JSON，供论文实验记录引用。

## Risks / Trade-offs

- [Risk] 当前 Clique sealing 在仓库内不可用，直接启动 signer 可能无法出块。→ Mitigation: 把 Clique 出块 smoke test 作为第一批实现任务；必要时恢复实验用 sealer，并限制影响范围到 Clique chain config。
- [Risk] Go plugin 要求 geth 二进制、Go 版本、module source 和 CGO 环境一致。→ Mitigation: 实验镜像保留同一源码树和 Go toolchain，固定 `CRYPTOUPGRADE_MODULE`，并在验证中执行 upload smoke test。
- [Risk] YAML 中 host、端口或 nodekey 配置错误会导致节点无法互联。→ Mitigation: 配置校验阶段检查重复节点 ID、重复端口、缺失 nodekey、无 signer、无效 signer 地址和不可生成 enode 的节点。
- [Risk] 在 YAML 中保存私钥会造成泄露。→ Mitigation: 配置只允许引用 keystore/password/nodekey 路径；示例文件不提交真实密钥。
- [Risk] 多服务器 Docker 网络不能像单机 compose 一样自动解析容器名。→ Mitigation: 配置显式使用 advertise host/IP 生成 enode，并要求服务器开放 P2P/RPC 端口。

## Migration Plan

1. 保留现有 `geth --dev` 和单节点实验命令，先新增独立多节点工具和示例配置。
2. 用示例 YAML 在本地 Docker Compose 环境验证 2-3 个节点的 init、互联和 Clique 出块。
3. 在服务器环境复用同一配置结构，只调整 host/IP、端口映射和 volume 根目录。
4. 后续实验命令默认仍可指向任意 RPC URL；多节点网络稳定后，再把实验 RPC 指向配置中的 observer 或 signer 节点。

Rollback 策略：删除渲染输出目录和容器/volume 即可回到单节点实验环境；代码层面不改变现有单节点命令和 Geth 默认启动路径。

## Open Questions

- 是否需要支持跨服务器自动分发文件，还是只生成每台服务器可执行的配置和命令。
- Clique signer 数量默认使用 1 个还是 3 个；论文实验如果需要容错，应使用 3 个 signer。
- 多节点 cryptoupgrade smoke test 是否只在一个 RPC 节点上传算法，还是要求所有节点都完成本地 plugin 编译后再验证调用一致性。
