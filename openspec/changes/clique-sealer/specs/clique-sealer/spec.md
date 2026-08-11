## ADDED Requirements

### Requirement: 多节点网络启动接口兼容

系统 SHALL 兼容 `add-multi-node-network` 定义的 YAML 配置、启动命令渲染和 Docker Compose 运行接口，使 multi-node 工具生成的节点命令无需人工修改即可启动 Clique 共识网络。

#### Scenario: signer 节点命令可直接启动

- **WHEN** `cryptoupgrade/network.GethArgs` 根据 `role: signer`、`account`、`keystore`、`password` 和 Clique consensus 配置生成 geth 启动参数
- **THEN** geth SHALL 接受该启动参数且不因 unknown flag 失败
- **AND** 节点 SHALL 解锁配置的 signer 账户并启用 Clique 出块服务

#### Scenario: observer 节点命令可直接同步

- **WHEN** `cryptoupgrade/network.GethArgs` 根据 `role: observer` 或 `role: rpc` 生成 geth 启动参数
- **THEN** geth SHALL 接受该启动参数且不解锁 signer 账户
- **AND** 节点 SHALL 通过 static peers 或 bootnodes 同步 Clique signer 节点产生的区块

#### Scenario: Docker Compose 命令兼容

- **WHEN** `cryptoupgrade/network.ComposeYAML` 生成包含 signer 和 observer 的 Docker Compose 服务
- **THEN** 每个服务的 command SHALL 能在实验镜像中启动 geth
- **AND** signer 服务 SHALL 使用容器内 password、nodekey、datadir 和 pluginDir 路径完成 Clique sealing 初始化

#### Scenario: HTTP RPC namespace 兼容 validate

- **WHEN** multi-node 配置的 `httpApis` 包含 `clique`、`eth`、`net`、`admin` 或 `miner`
- **THEN** geth SHALL 注册 validate 所需的 RPC namespace
- **AND** `clique_getSigners` SHALL 能返回当前 Clique 授权 signer 集合

### Requirement: Clique signer 账户接入

系统 SHALL 在显式启用 Clique sealing 时，将本地节点配置的 signer 地址与已解锁账户签名能力接入 Clique 共识引擎。

#### Scenario: signer 账户成功接入

- **WHEN** 节点运行在 Clique 链上，multi-node 配置中该节点 `role` 为 `signer`，并提供 signer 地址、keystore 和 password 文件
- **THEN** 系统 SHALL 解锁该 signer 账户
- **AND** SHALL 将该 signer 地址和签名函数注册到 Clique 共识引擎

#### Scenario: signer 解锁失败

- **WHEN** 节点启用了 Clique sealing，但 signer 账户不存在或 password 文件无法解锁该账户
- **THEN** 节点启动 SHALL 失败
- **AND** 错误信息 SHALL 标识 signer 账户解锁失败

#### Scenario: observer 不解锁 signer

- **WHEN** multi-node 配置中节点 `role` 为 `observer` 或 `rpc`
- **THEN** 系统 SHALL NOT 为 Clique 共识解锁 signer 账户
- **AND** 节点 SHALL 仍可作为普通同步节点验证和导入 Clique 区块

### Requirement: Clique 区块签名

系统 SHALL 使用 Clique 共识规则对候选区块进行签名，并生成可被其他节点验证的 sealed block。

#### Scenario: 授权 signer 签名区块

- **WHEN** 本地 signer 已注册到 Clique 共识引擎，且该 signer 位于当前授权 signer 集合中
- **THEN** Clique sealing SHALL 在 header extra-data 的签名区域写入该 signer 的有效签名
- **AND** sealed block 的 header SHALL 通过 Clique 共识校验

#### Scenario: 拒绝未授权 signer

- **WHEN** 本地 signer 未包含在当前 Clique 授权 signer 集合中
- **THEN** Clique sealing SHALL NOT 生成新区块
- **AND** 系统 SHALL 返回或记录 unauthorized signer 错误

#### Scenario: 遵守近期签名限制

- **WHEN** Clique 规则暂时禁止本地 signer 连续签署新区块
- **THEN** Clique sealing SHALL NOT 立即生成违反 recent signer 限制的区块
- **AND** 后续出块 SHALL 在 signer 重新具备出块资格后继续尝试

### Requirement: signer 节点持续出块

系统 SHALL 在启用 Clique sealing 的 signer 节点上持续提供本地区块生产服务。

#### Scenario: node1 持续产生区块

- **WHEN** node1 使用包含自身 signer 的 Clique genesis 启动，并显式启用 Clique sealing
- **THEN** node1 的当前区块高度 SHALL 在至少两个 Clique period 内持续增长
- **AND** 新产生的区块 SHALL 能通过本地链导入和共识校验

#### Scenario: 未启用 sealing 不出块

- **WHEN** 节点运行在 Clique 链上但未显式启用 Clique sealing
- **THEN** 节点 SHALL NOT 主动生产新区块
- **AND** 节点 SHALL 保持可同步其他 signer 节点产生的区块

#### Scenario: 停止节点时退出 sealing 循环

- **WHEN** signer 节点正在执行 Clique sealing 且节点收到停止信号
- **THEN** sealing 循环 SHALL 停止创建新区块
- **AND** 节点关闭 SHALL NOT 因 sealing goroutine 阻塞

### Requirement: node2 同步 Clique 区块

系统 SHALL 支持 observer 节点通过 P2P 同步 signer 节点产生的 Clique 区块。

#### Scenario: node2 同步 node1 区块

- **WHEN** node1 持续出块，node2 使用同一 genesis 启动并连接到 node1
- **THEN** node2 SHALL 导入 node1 产生的 Clique 区块
- **AND** node2 的当前区块高度 SHALL 最终等于或接近 node1 的当前区块高度

#### Scenario: node2 拒绝错误 genesis

- **WHEN** node2 使用与 node1 不兼容的 genesis 或 chain ID 启动
- **THEN** node2 SHALL NOT 将 node1 作为同链 peer 完成区块同步
- **AND** 验证入口 SHALL 将该情况记录为同步失败

#### Scenario: 双节点验证结果可复现

- **WHEN** 执行 `cryptoupgrade/cmd/multinode -mode validate` 或等价 Clique 双节点验证入口
- **THEN** 输出结果 SHALL 记录 node1 起止高度、node2 起止高度、peer 数量、chain ID 和验证耗时
- **AND** 验证结果 SHALL 能明确表示 node1 出块和 node2 同步是否通过
