# cryptoupgrade 多节点私链网络

本文档说明如何使用一个 YAML 配置管理多个 geth Docker 节点，并组成 cryptoupgrade 私有链实验网络。

## 目录约定

源码内只提交配置、模板、命令和文档：

```text
cryptoupgrade/
├── cmd/multinode/              # render/init/validate 命令
├── network/                    # YAML 解析和制品生成逻辑
├── docker/Dockerfile.lab       # 支持 Go plugin 编译的实验镜像
├── examples/networks/          # 示例网络配置
└── docs/multi_node_network.md
```

运行时数据默认输出到：

```text
build/cryptoupgrade-networks/<network-name>/
├── network.yaml
├── genesis.json
├── docker-compose.yml
├── artifacts.json
├── validate-result.json
└── nodes/
    ├── node1/
    │   ├── datadir/
    │   ├── plugin/
    │   ├── config.toml
    │   ├── static-nodes.json
    │   ├── nodekey
    │   ├── password.txt
    │   └── start.sh
    └── node2/
        ├── datadir/
        ├── plugin/
        ├── config.toml
        ├── static-nodes.json
        ├── nodekey
        ├── password.txt
        └── start.sh
```

每个节点必须使用独立 `datadir` 和 `plugin` 目录，避免链数据库、`algorithm_info.json` 和 Go plugin `.so` 文件互相污染。

## 配置文件

默认示例：

```bash
cryptoupgrade/examples/networks/local-2nodes.yaml
```

一个 YAML 管理整个网络：

- `network`: chain ID、network ID、gas limit。
- `consensus`: Clique period、epoch 和 signer 地址。
- `docker`: 实验镜像和 Docker network 名称。
- `accounts`: 账户地址、keystore 路径、password 文件和初始余额。
- `nodes`: 每个节点的 ID、角色、host、端口、目录、nodekey 和账户引用。

YAML 不保存账户私钥明文。账户私钥应通过 encrypted keystore 文件提供，P2P nodekey 通过文件路径引用。

## 生成制品

```bash
go run ./cryptoupgrade/cmd/multinode \
  -mode render \
  -config cryptoupgrade/examples/networks/local-2nodes.yaml \
  -out build/cryptoupgrade-networks/local-2nodes \
  -output-json build/cryptoupgrade-networks/local-2nodes/render-result.json
```

该命令会生成：

- `genesis.json`
- 每个节点的 `config.toml` 和归档用 `static-nodes.json`
- 每个节点的 `start.sh`
- `docker-compose.yml`
- `artifacts.json`

当前 geth 已忽略 datadir 下的 `static-nodes.json`，因此渲染器会把静态节点写入 `config.toml` 的 `Node.P2P.StaticNodes`。`static-nodes.json` 保留在节点目录下用于实验归档和兼容旧版本节点。

如果要直接初始化本机 datadir：

```bash
go run ./cryptoupgrade/cmd/multinode \
  -mode init \
  -config cryptoupgrade/examples/networks/local-2nodes.yaml \
  -out build/cryptoupgrade-networks/local-2nodes \
  -geth ./build/bin/geth
```

## Docker 镜像

cryptoupgrade 的动态升级路径会在节点运行时执行 Go plugin 编译，所以实验镜像必须保留 Go toolchain、CGO 依赖和源码模块。

```bash
docker build \
  -f cryptoupgrade/docker/Dockerfile.lab \
  -t cryptoupgrade-geth:lab \
  .
```

也可以运行检查脚本：

```bash
sh cryptoupgrade/docker/check.sh
```

如果本机没有 Docker 或 Docker Compose，脚本会输出 skip 原因。

## 单服务器启动

进入 render 输出目录：

```bash
cd build/cryptoupgrade-networks/local-2nodes
docker compose up -d
docker compose logs -f node1
```

宿主机 RPC 端口来自 YAML 的 `httpHostPort`：

```bash
curl -s http://127.0.0.1:8661 \
  -H 'content-type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}'
```

## 验证网络

```bash
go run ./cryptoupgrade/cmd/multinode \
  -mode validate \
  -config cryptoupgrade/examples/networks/local-2nodes.yaml \
  -output-json build/cryptoupgrade-networks/local-2nodes/validate-result.json
```

验证项包括：

- RPC 是否可达；
- chain ID 是否和 YAML 一致；
- peer count 是否达到配置要求；
- 等待至少一个 Clique period 后区块高度是否增长；
- `clique_getSigners` 是否能返回授权 signer。

可选 Add smoke test：

```bash
go run ./cryptoupgrade/cmd/multinode \
  -mode validate \
  -config cryptoupgrade/examples/networks/local-2nodes.yaml \
  -crypto-smoke \
  -output-json build/cryptoupgrade-networks/local-2nodes/validate-result.json
```

该测试要求 signer 账户已解锁，并会通过 RPC 执行 `CodeStorage.uploadCode` 和 `CodeStorage.callFunc("Add", ...)`。

## 多服务器部署

多服务器部署仍使用同一个 YAML，但需要调整：

- `host` 和 `advertiseHost` 使用服务器内网 IP 或可互通 IP；
- `p2pHostPort` 和 `httpHostPort` 使用服务器开放端口；
- 每台服务器保存对应节点的 nodekey、keystore、password 文件；
- 防火墙放行 P2P TCP/UDP 端口和需要暴露的 RPC 端口。

多服务器场景不要依赖 Docker Compose 服务名解析。`config.toml` 的 `Node.P2P.StaticNodes` 和归档用 `static-nodes.json` 都会使用 `advertiseHost` 生成 enode。

## Clique 注意事项

Clique signer 是 geth 节点内部的共识服务，不是单独运行的外部出块进程。`role: signer` 节点会通过生成的启动参数解锁 signer 账户并启用本地 Clique sealing；`role: observer` 或 `role: rpc` 节点不解锁账户，只通过 P2P 同步并验证 signer 节点产生的区块。

`validate` 的区块高度增长和 `clique_getSigners` 检查用于确认 signer 出块、observer 同步和 genesis signer 授权是否一致。

## Go Plugin 注意事项

容器内必须满足：

- `CRYPTOUPGRADE_MODULE=/go-ethereum`
- `GETH_CRYPTOUPGRADE_PLUGIN_DIR=/plugin`
- Go toolchain 可用；
- CGO 编译依赖可用；
- geth 二进制和 plugin 编译使用同一份源码模块。

如果 `CodeStorage.uploadCode` 失败，优先检查容器日志中的 Go build 错误、`/plugin` 是否可写，以及 `CRYPTOUPGRADE_MODULE` 是否指向包含 `go.mod` 的源码目录。
