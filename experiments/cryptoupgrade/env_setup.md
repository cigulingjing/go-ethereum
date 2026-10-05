# Cryptoupgrade experiments

本目录按职责划分为四部分：

- `scripts`：测试入口和命令行脚本，每个测试入口目录包含一个 `main.go`；
- `core`：测试运行核心，包括 network 组网、配置加载、节点制品渲染和 CGO/WASM 公共运行器；
- `config`：YAML 网络配置和 signer 私钥等实验输入；
- `algorithm`：Go、Solidity、PQC 和 Groth16 等实验算法实现。

所有测试生成的节点、编译缓存、stage log 和新结果统一写入 `output`；历史 `results` 目录保留为只读快照。

实验代码只通过 `github.com/ethereum/go-ethereum/cryptoupgrade` facade 使用运行时能力。
历史 `cryptoupgrade/results` 是只读运行快照，仍保留原路径且不应改写。

## 环境配置

第一步： Geth编译
```shell
make geth
```

第二步：WASM环境配置
```shell
wget https://github.com/tinygo-org/tinygo/releases/download/v0.42.0/tinygo_0.42.0_amd64.deb

sudo dpkg -i tinygo_0.42.0_amd64.deb
```

第三步：foundry安装
```shell
curl -L https://getfoundry.sh/install | bash
source ~/.bashrc
foundryup
```

## 私有网络搭建

第一步：生成部署目录 render

```shell
go run ./experiments/cryptoupgrade/scripts/render \
  -config /home/liuqi/project/go-ethereum/experiments/cryptoupgrade/config/local.yaml \
  -geth "$(pwd)/build/bin/geth"
```

第二步：启动节点，多个命令行的方式启动节点

```shell
cd ./experiments/cryptoupgrade/output/network/local

export CRYPTOUPGRADE_MODULE=/home/liuqi/project/go-ethereum
export GETH_CRYPTOUPGRADE_NODE_ID=node1   

./node1/start.sh
```

第三步：Docker 部署（多节点推荐）

节点在 Docker bridge 子网内通过 `node1`、`node2` 等主机名互联；只有 YAML 里 `expose` 列出的节点映射 RPC 到宿主机。

```shell
docker build -f /home/liuqi/project/go-ethereum/experiments/cryptoupgrade/scripts/docker/Dockerfile.lab -t cryptoupgrade-geth:lab .
cd /home/liuqi/project/go-ethereum/experiments/cryptoupgrade/output/network/local
docker compose up
```

`local.yaml` 默认 20 节点仅暴露 node1 的 HTTP RPC（`8761 -> 8545`），其余节点不占用宿主机端口。

## 算法准备

准备.go文件编写的算法文件。注意函数名字，输入输出参数

```shell
go run /home/liuqi/project/go-ethereum/cryptoupgrade/wasmtool/cmd/wasmbuild/main.go \
  -source polynomial_mul.go \
  -function PolynomialMul \
  -itype "uint256[],uint256[],uint256" \
  -otype "uint256[]" \
  -out polynomial_mul.wasm
```


## WASM 升级上链

根据UTC文件获取私钥：
```shell
➜  keystore git:(master) ✗ cast wallet decrypt-keystore UTC--2026-08-11T00-00-00.000000000Z--f5f871aa6bd253914705898c66251f994aa426fa --keystore-dir ./
Enter password: 
UTC--2026-08-11T00-00-00.000000000Z--f5f871aa6bd253914705898c66251f994aa426fa's private key is: 
```

向 CodeStorage 提交指定 `.wasm` 升级交易：

```shell
go run /home/liuqi/project/go-ethereum/experiments/cryptoupgrade/scripts/upload-wasm \
  -wasm polynomial_mul.wasm \
  -key  0x2aedaed0ff6818dbc349b870e384504aa50dfad34116c089ba58fc28638eb7a2 \
  -rpc http://127.0.0.1:8761
```

常用参数：

| 参数 | 含义 |
| --- | --- |
| `-mode upload` | 首次上传，异步激活（默认） |
| `-mode immediate` | 当前高度立即生效的新版本 |
| `-mode version` | 预约 `-activation-block` 生效 |
| `-name Add` | 算法名（省略则从文件名推断） |
| `-itype` / `-otype` | ABI 类型，Add 默认 `int256,int256` -> `int256` |

`upload` 模式会在 receipt 成功后轮询 `getActiveVersion`，确认节点已完成 WASM 加载。
