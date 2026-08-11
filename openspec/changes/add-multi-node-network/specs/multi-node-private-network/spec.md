## ADDED Requirements

### Requirement: YAML 网络配置载入

系统 SHALL 从 YAML 配置文件载入多节点私有链拓扑，并将节点身份、网络地址、端口、目录、账户和角色归一化为可用于部署的网络配置。

#### Scenario: 载入有效多节点配置

- **WHEN** 工具读取包含 network、consensus 和 nodes 的有效 YAML 配置
- **THEN** 系统 SHALL 生成包含 chain ID、network ID、共识参数和每个节点配置的归一化结果
- **AND** 每个节点配置 SHALL 包含稳定节点 ID、host 或 advertise host、P2P 端口、RPC 端口、datadir、pluginDir 和 role

#### Scenario: 拒绝重复节点标识

- **WHEN** YAML 中存在重复的节点 ID
- **THEN** 配置载入 SHALL 失败
- **AND** 错误信息 SHALL 标识重复的节点 ID

#### Scenario: 拒绝无效 signer 配置

- **WHEN** consensus type 为 `clique` 但配置中没有 signer 地址或 signer 节点缺少账户配置
- **THEN** 配置载入 SHALL 失败
- **AND** 错误信息 SHALL 标识缺失的 signer 信息

#### Scenario: 配置不保存私钥明文

- **WHEN** 节点需要账户或 nodekey
- **THEN** YAML 配置 SHALL 通过 keystore、password 文件或 nodekey 文件路径引用密钥材料
- **AND** 配置 schema SHALL NOT 要求或推荐保存账户私钥明文

### Requirement: 私有链初始化制品生成

系统 SHALL 根据归一化配置生成多节点私有链初始化制品，包括 genesis、静态 peer、节点目录和可复现的启动参数。

#### Scenario: 生成 Clique genesis

- **WHEN** 配置声明 consensus type 为 `clique`
- **THEN** 生成的 genesis SHALL 包含 Clique period 和 epoch
- **AND** genesis extradata SHALL 包含配置中的授权 signer 地址
- **AND** genesis chain ID SHALL 与 YAML 中的 chain ID 一致

#### Scenario: 保留 cryptoupgrade 预部署状态

- **WHEN** 配置启用 cryptoupgrade 实验链初始化
- **THEN** 生成的 genesis SHALL 包含 CodeStorage 预部署账户
- **AND** SHALL 保留用于实验发送交易的 funded account alloc

#### Scenario: 生成静态 peer 列表

- **WHEN** 节点配置包含 nodekey 和 advertise host
- **THEN** 系统 SHALL 为每个节点生成可被其他节点使用的 enode 地址
- **AND** 每个节点的 static peer 列表 SHALL 包含除自身以外的已配置 peer

#### Scenario: 初始化节点数据目录

- **WHEN** 执行网络初始化
- **THEN** 系统 SHALL 为每个节点创建独立 datadir 和 pluginDir
- **AND** SHALL 使用同一个 genesis 初始化所有节点数据目录

### Requirement: Docker 镜像打包

系统 SHALL 提供可构建的 Docker 镜像，用于运行支持 cryptoupgrade 动态升级路径的 Geth 节点。

#### Scenario: 构建实验用 Geth 镜像

- **WHEN** 执行 Docker build
- **THEN** 镜像 SHALL 包含当前仓库构建出的 geth 二进制
- **AND** 镜像 SHALL 包含运行 Go plugin 编译所需的 Go toolchain、CGO 编译依赖和源码模块

#### Scenario: 配置 cryptoupgrade 运行时目录

- **WHEN** 容器启动节点
- **THEN** 容器 SHALL 为该节点设置独立的 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`
- **AND** 容器 SHALL 设置可用于 Go plugin 编译的 module 路径

#### Scenario: 保留普通镜像兼容性

- **WHEN** 用户继续使用现有最小 geth Docker 镜像
- **THEN** 该镜像 SHALL 仍可按原有入口启动 geth
- **AND** 多节点实验镜像 SHALL NOT 改变现有最小镜像的入口语义

### Requirement: 多节点容器编排

系统 SHALL 根据 YAML 配置生成或提供 Docker Compose 编排，使多个节点能够在服务器上启动并组成同一私有链网络。

#### Scenario: 生成单服务器 compose

- **WHEN** 配置包含多个节点且 docker compose 输出被请求
- **THEN** 系统 SHALL 为每个节点生成一个服务定义
- **AND** 每个服务 SHALL 使用独立 datadir、pluginDir、P2P 端口和 RPC 端口
- **AND** 所有服务 SHALL 使用同一 chain ID、network ID 和 genesis 初始化结果

#### Scenario: 节点启动后互联

- **WHEN** compose 启动完成
- **THEN** 每个节点 SHALL 使用 static peers 或等价启动参数连接到同一私有链网络
- **AND** RPC 查询 `net_peerCount` SHALL 能够反映配置期望的 peer 连接

#### Scenario: 多服务器显式地址

- **WHEN** 配置中的节点位于不同服务器
- **THEN** 生成的 enode 和启动参数 SHALL 使用配置的 advertise host 或 IP
- **AND** SHALL NOT 依赖仅在单机 compose 网络中可解析的容器名

### Requirement: Clique 共识 signer 节点

系统 SHALL 提供 Clique signer 节点配置，使私有链能够通过授权 signer 账户完成出块和共识认证。

#### Scenario: signer 节点启动

- **WHEN** 节点 role 为 `signer`
- **THEN** 启动命令 SHALL 解锁该节点配置的 signer 账户
- **AND** SHALL 启用区块生产所需的 mining 或等价 Clique signer 模式
- **AND** SHALL 暴露验证 signer 状态所需的 RPC namespace

#### Scenario: observer 节点不解锁 signer

- **WHEN** 节点 role 为 `observer` 或 `rpc`
- **THEN** 启动命令 SHALL NOT 解锁 Clique signer 账户
- **AND** 节点 SHALL 通过 peer 同步 signer 节点产生的区块

#### Scenario: Clique 出块验证

- **WHEN** 至少一个授权 signer 节点启动并完成 peer 初始化
- **THEN** 链高度 SHALL 在 Clique period 后增长
- **AND** 新区块 header SHALL 通过 Clique 共识校验

#### Scenario: signer 授权状态查询

- **WHEN** 对 signer 节点执行 Clique 状态查询
- **THEN** 返回结果 SHALL 包含 genesis 或投票后生效的授权 signer
- **AND** SHALL 能够用于确认当前出块账户被授权

### Requirement: 多节点网络验证

系统 SHALL 提供验证入口，用于确认部署出的私有链网络满足实验前置条件。

#### Scenario: 验证 RPC 和 chain ID

- **WHEN** 验证入口连接配置中的节点 RPC
- **THEN** 每个可验证节点 SHALL 返回与 YAML 配置一致的 chain ID
- **AND** RPC 连接失败 SHALL 被记录为验证失败

#### Scenario: 验证 peer 和出块

- **WHEN** 验证入口等待至少一个 Clique period
- **THEN** signer 节点和 observer 节点 SHALL 达到配置要求的 peer 数量
- **AND** 至少一个节点的区块高度 SHALL 增长

#### Scenario: 验证 cryptoupgrade 基础路径

- **WHEN** 配置启用 cryptoupgrade smoke test
- **THEN** 验证入口 SHALL 通过 RPC 执行一个基础算法上传和调用
- **AND** 返回结果 SHALL 与测试输入的期望输出一致

#### Scenario: 输出可复现验证结果

- **WHEN** 验证完成
- **THEN** 系统 SHALL 输出机器可读验证结果
- **AND** 结果 SHALL 记录配置文件路径、节点 RPC、peer count、起止区块高度和 cryptoupgrade smoke test 状态
