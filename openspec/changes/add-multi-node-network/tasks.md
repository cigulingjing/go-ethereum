## 1. Context And Schema

- [x] 1.1 复核现有单节点私链启动文档、`Dockerfile`、`Dockerfile.alltools`、cryptoupgrade plugin 运行时环境和 `consensus/clique.Seal` 当前行为。
- [x] 1.2 定义多节点 YAML schema，覆盖 network、consensus、docker、accounts 和 nodes 字段。
- [x] 1.3 新增示例配置文件，包含 1 个 signer 节点和至少 1 个 observer/rpc 节点。
- [x] 1.4 实现配置解析和归一化逻辑，支持默认端口、相对路径解析和节点角色解析。
- [x] 1.5 添加配置校验测试，覆盖重复节点 ID、重复端口、缺失 signer、无效地址和密钥路径缺失。

## 2. Network Artifact Generation

- [x] 2.1 实现 Clique genesis 生成或校验逻辑，正确写入 chain ID、period、epoch 和 signer extradata。
- [x] 2.2 合并 cryptoupgrade 实验链 alloc，确保 CodeStorage 预部署账户和 funded account 被保留。
- [x] 2.3 根据 nodekey 和 advertise host 生成 enode 地址。
- [x] 2.4 为每个节点生成 static peers 或等价 peer 配置，并排除节点自身。
- [x] 2.5 实现节点 datadir、pluginDir、keystore、password 文件路径的输出目录布局和初始化命令渲染。
- [x] 2.6 添加 artifact 生成单元测试，验证 genesis、static peers 和目录布局稳定可复现。

## 3. Docker Packaging

- [x] 3.1 新增实验用 Dockerfile，保留 Go toolchain、CGO 依赖、当前源码模块和构建出的 geth 二进制。
- [x] 3.2 配置容器环境变量 `CRYPTOUPGRADE_MODULE` 和 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`，保证 Go plugin 能在容器内编译与加载。
- [x] 3.3 保留现有最小 geth 镜像入口语义，不改变普通 Dockerfile 的默认启动方式。
- [x] 3.4 实现 Docker Compose 模板或生成器，为每个节点输出独立服务、volume、端口映射和启动命令。
- [x] 3.5 添加 Docker build 和 compose 渲染验证脚本，能够在没有 Docker daemon 时给出明确跳过原因。

## 4. Clique Signer Support

- [x] 4.1 添加 Clique 出块 smoke test，确认当前仓库的 signer 节点能否生产区块。
- [ ] 4.2 如果当前 `consensus/clique.Seal` 仍不可用，恢复实验用 Clique sealer，并限制影响范围到 Clique chain config。
- [ ] 4.3 为 signer 节点启动命令渲染 `--mine`、账户解锁、password、etherbase 和必要 RPC namespace。
- [x] 4.4 为 observer/rpc 节点启动命令确认不解锁 signer 账户，只通过 peer 同步区块。
- [x] 4.5 添加 Clique 状态查询或验证逻辑，确认授权 signer 与 genesis/投票状态一致。

## 5. Validation Command

- [x] 5.1 新增多节点网络辅助命令入口，支持 render、init 和 validate 等基础模式。
- [x] 5.2 在 validate 模式中检查每个配置节点的 RPC 可达性和 chain ID 一致性。
- [x] 5.3 在 validate 模式中检查 peer count，并记录未达到期望连接数的节点。
- [x] 5.4 在 validate 模式中等待至少一个 Clique period，确认区块高度增长。
- [x] 5.5 添加可选 cryptoupgrade smoke test，通过 RPC 上传并调用基础 Add 算法。
- [x] 5.6 输出机器可读 JSON 验证结果，记录配置路径、节点 RPC、peer count、区块高度和 smoke test 状态。

## 6. Documentation And Server Workflow

- [x] 6.1 编写多节点私链部署说明，覆盖配置文件、artifact 渲染、Docker build、compose 启动和日志查看。
- [x] 6.2 编写多服务器部署说明，说明 advertise host/IP、开放端口、nodekey 分发和静态 peer 生成方式。
- [x] 6.3 记录 Clique signer 密钥管理注意事项，禁止在示例 YAML 中保存私钥明文。
- [x] 6.4 记录 cryptoupgrade plugin 容器环境要求和常见编译失败排查方式。

## 7. Verification

- [x] 7.1 运行配置解析、artifact 生成和 Clique 相关 Go 单元测试。
- [ ] 7.2 运行多节点辅助命令的 render/init/validate smoke test。
- [ ] 7.3 在可用 Docker 环境中构建实验镜像并启动至少 2 个节点的 compose 网络。
- [ ] 7.4 验证 compose 网络中 RPC、peer count、Clique 出块和 cryptoupgrade Add smoke test 均通过。
- [x] 7.5 运行 `openspec validate add-multi-node-network` 并根据结果修正文档或 spec。
