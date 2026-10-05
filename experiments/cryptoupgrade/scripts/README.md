# Cryptoupgrade Scripts

以下命令默认在 go-ethereum 仓库根目录执行。网络 YAML 位于 `experiments/cryptoupgrade/config/`，节点制品、编译缓存、stage log 和新结果默认写入 `experiments/cryptoupgrade/output/`。

## 运行前提

先编译 geth，并根据实验需要启动节点：

```bash
make geth
go run ./experiments/cryptoupgrade/scripts/render \
  -config experiments/cryptoupgrade/config/local-1node.yaml \
  -geth "$(pwd)/build/bin/geth"
./experiments/cryptoupgrade/output/network/local-1node/node1/start.sh
```

涉及 Solidity 对照的脚本还需要 `solc`；Docker 检查脚本需要 Docker Compose。
直接执行节点 `start.sh` 时 RPC 是 `http://127.0.0.1:8545`；使用 Docker Compose 并按 `local-1node.yaml` 暴露时，宿主机映射端口是 `8761`。

### 预编译合约侧调用

`algorithm/contracts/src/CryptoUpgradePrecompileVerify.sol` 是一个通用 Solidity adapter，
把 `(bytes,bytes,bytes)` 编码后转发给客户端预编译。合约按 `paris` EVM 版本编译，适配当前单节点实验链。

```bash
cd experiments/cryptoupgrade/algorithm/contracts
forge build
forge inspect CryptoUpgradePrecompileVerify bytecode
```

部署后可用以下固定地址进行 `eth_call`：`0x5b` Aigis、`0x5c` Dilithium3、`0x5d` ML-DSA-65、
`0x5e` SLH-DSA-SHAKE-192f、`0x5f` Groth16 BLS12-381、`0x60` Groth16 BN254。
调用形式为：

```bash
cast call <ADAPTER_ADDRESS> \
  'verify(address,bytes,bytes,bytes)(bool)' \
  0x000000000000000000000000000000000000005d \
  0x<VERIFICATION_KEY> 0x<PUBLIC_INPUT> 0x<PROOF_OR_SIGNATURE> \
  --rpc-url http://127.0.0.1:8545
```

PQC 的三个参数依次是 `pk`、`message`、`signature`；Groth16 的三个参数依次是 `vk`、`public`、`proof`。

## 网络与升级

### `render` 生成测试目录与文件

根据 YAML 生成 genesis、节点目录、启动脚本、静态节点和 Docker Compose 文件。

```bash
go run ./experiments/cryptoupgrade/scripts/render \
  -config experiments/cryptoupgrade/config/local-1node.yaml \
  -geth "$(pwd)/build/bin/geth"
```

常用参数：`-config` 指定 YAML，`-out` 覆盖输出目录，`-geth` 指定 geth，`-output-json` 输出制品清单。

### `upload-wasm` 完成代码升级

向 CodeStorage 上传 WASM 算法并等待激活。

```bash
go run ./experiments/cryptoupgrade/scripts/upload-wasm \
  -wasm experiments/cryptoupgrade/algorithm/go/wasm/add.wasm \
  -name Add \
  -key 0xYOUR_PRIVATE_KEY \
  -rpc http://127.0.0.1:8761
```

常用参数：`-mode upload|immediate|version`，`-version`，`-activation-block`，`-itype`，`-otype`，`-output-json`。私钥不要提交到 Git。

### `benchupgradelatency` 升级延迟测试

测量多节点网络中 WASM 升级交易从提交、收据确认到各节点完成激活的延迟。需要先用 `render` 生成多节点网络。

```bash
go run ./experiments/cryptoupgrade/scripts/benchupgradelatency \
  -config experiments/cryptoupgrade/output/network/local/network.yaml \
  -source experiments/cryptoupgrade/algorithm/go/wasm/add.wasm \
  -algorithm Add \
  -rounds 1
```

常用参数：`-sender`，`-rounds`，`-poll-interval`，`-timeout`，`-out`，`-skip-preflight`。

## 算法执行效率

### `benchgroth16cgowasm` groth16零知识证明测试

在进程内对比 Groth16 的 native CGO 和 EvoCrypt WASM 执行时间，不包含 RPC 和节点启动时间。

```bash
CGO_ENABLED=1 go run ./experiments/cryptoupgrade/scripts/benchgroth16cgowasm \
  -root "$(pwd)" \
  -algorithms groth16_bls12381_verify \
  -warmup 10 \
  -n 100
```

可选算法：`groth16_bls12381_verify`、`groth16_bn254_verify`。结果默认写入 `experiments/cryptoupgrade/output/results/`。

### `benchpqccgowasm` PQCgo算法测试

在进程内对比 PQC 算法的 native CGO 和 EvoCrypt WASM 执行时间。

```bash
CGO_ENABLED=1 go run ./experiments/cryptoupgrade/scripts/benchpqccgowasm \
  -root "$(pwd)" \
  -algorithms aigis_sig2_verify \
  -warmup 10 \
  -n 100
```

不指定 `-algorithms` 时测量默认的四个 PQC 算法。结果默认写入 `experiments/cryptoupgrade/output/results/`。

### `benchnodeexec`

通过单节点 `eth_call` 执行 EvoCrypt，并对照 native CGO 结果。`evocrypt` 使用节点 stage log 中的 `coprocessor_exit` 时间，`ethCallMeanMillis` 表示 RPC 往返时间。

```bash
CGO_ENABLED=1 go run ./experiments/cryptoupgrade/scripts/benchnodeexec \
  -root "$(pwd)" \
  -rpc http://127.0.0.1:8761 \
  -algorithms groth16_bls12381_verify \
  -warmup 10 \
  -n 100
```

常用参数：`-key-file`，`-stage-log`，`-output`，`-chain-id`。默认 stage log 位于 `output/network/local-1node/node1/plugin/stage_timing.jsonl`。

### `benchgroth16bls12381`

启动一个新客户端后，对 Groth16/BLS12-381 的 EvoCrypt WASM 和客户端 precompile 进行同输入、多次采样测试。
结果目录必须显式指定，便于把一次本地实验的输入和结果集中保存：

```bash
CODE=groth16-bls12381-evocrypt-precompile-$(date +%Y%m%d-%H%M%S)
CGO_ENABLED=1 go run ./experiments/cryptoupgrade/scripts/benchgroth16bls12381 \
  -root "$(pwd)" \
  -rpc http://127.0.0.1:8545 \
  -stage-log "$(pwd)/experiments/cryptoupgrade/output/network/<client-code>/node1/plugin/stage_timing.jsonl" \
  -output "$(pwd)/experiments/cryptoupgrade/results/$CODE" \
  -warmup 10 -n 100
```

结果目录包含 `inputs/`、`result.json` 和 `result.txt`；节点 datadir、WASM runtime 缓存和原始 stage log 仍留在 `output/network/`，不会混入实验结果。

### `benchprecompile` 六算法 EvoCrypt/precompile 对照测试

在同一个新启动的本地单节点上，依次测试 Aigis-Sig2、Dilithium3、ML-DSA-65、SLH-DSA-SHAKE-192f、Groth16/BLS12-381 和 Groth16/BN254。每个算法使用相同的输入向量分别调用 EvoCrypt WASM 和客户端 native precompile，并记录节点内部执行时间与 `eth_call` 往返时间。

```bash
RUN_CODE="pqc-groth16-precompile-$(date +%Y%m%d-%H%M%S)"
export RUN_CODE
CGO_ENABLED=1 go run ./experiments/cryptoupgrade/scripts/benchprecompile \
  -root "$(pwd)" \
  -rpc http://127.0.0.1:8545 \
  -stage-log "$(pwd)/experiments/cryptoupgrade/output/network/<client-code>/node1/plugin/stage_timing.jsonl" \
  -output-root "$(pwd)/experiments/cryptoupgrade/results" \
  -run-code "$RUN_CODE" \
  -warmup 10 -n 50
```

结果分别写入 `experiments/cryptoupgrade/results/<algorithm>/<run-code>/`，每个目录包含 `inputs/`、`result.json` 和 `result.txt`。可用 `-algorithms` 指定逗号分隔的算法名；省略时默认执行上述六个算法。

## EVM 与交易测试

### `benchexecutionefficiency`

通过 EVM `eth_call` 对比 WASM upgrade、Solidity contract 和 precompile 等方案的执行效率。

```bash
go run ./experiments/cryptoupgrade/scripts/benchexecutionefficiency \
  -rpc http://127.0.0.1:8761 \
  -algorithms Add \
  -schemes upgrade,contract,precompile \
  -warmup 10 \
  -n 100
```

常用参数：`-upload=false`，`-solc`，`-evm-version`，`-output-json`。默认结果目录为 `experiments/cryptoupgrade/output/results/execution-efficiency/`。

### `benchtxgas`

在单节点网络中比较不同实现方案的交易 Gas 消耗。

```bash
go run ./experiments/cryptoupgrade/scripts/benchtxgas \
  -rpc http://127.0.0.1:8761 \
  -algorithms Add
```

常用参数：`-from`，`-solc`，`-poly-profile P1`，`-output-json`。默认结果目录为 `experiments/cryptoupgrade/output/results/tx-gas/`。

### `benchthroughput`

测量多个交易数量下的算法吞吐量，支持 WASM upgrade 和 Solidity contract 方案。

```bash
go run ./experiments/cryptoupgrade/scripts/benchthroughput \
  -rpc http://127.0.0.1:8761 \
  -algorithm PolynomialMul \
  -tx-counts 10,20,30,40,50 \
  -schemes upgrade,contract
```

可选算法：`PolynomialMul`、`Blake2bSum256`。常用参数：`-upload=false`，`-poly-left-len`，`-poly-right-len`，`-output-json`。

## Docker 检查

### `docker/check.sh`

构建实验镜像、渲染网络，并检查 Docker Compose 配置。

```bash
./experiments/cryptoupgrade/scripts/docker/check.sh
```

可通过环境变量覆盖默认值：

```bash
CONFIG=experiments/cryptoupgrade/config/local.yaml \
OUT=experiments/cryptoupgrade/output/docker-check \
IMAGE=cryptoupgrade-geth:lab \
./experiments/cryptoupgrade/scripts/docker/check.sh
```

未安装 Docker 时脚本会跳过检查并正常退出。
