# Cryptoupgrade Scripts

## 第五章单节点实验

第五章的新链上测量使用 `config/local-1node.yaml`，不启动 20 节点网络。单节点启动命令及 PBKDF2 原始数据、曲线、证据矩阵和验收边界见 `experiments/cryptoupgrade/results/chapter-five/20261022-single-node-pbkdf2-rebuilt/report.md`。复现实验先按下文“运行前提”启动单节点，再运行：

```bash
python3 experiments/cryptoupgrade/scripts/chapterfive/run_pbkdf2.py \
  --wasm experiments/cryptoupgrade/results/chapter-five/20261022-single-node-pbkdf2-rebuilt/pbkdf2_sha256.wasm \
  --output experiments/cryptoupgrade/output/results/chapter-five-recheck
python3 experiments/cryptoupgrade/scripts/chapterfive/plot_pbkdf2.py \
  experiments/cryptoupgrade/output/results/chapter-five-recheck
```

本采集入口固定三路径相同输入和七档内部迭代次数，按 3 轮×30 次 `eth_call`、每档每路径 9 笔真实收据报告。调用计费使用单独的 `local-1node-pricing.yaml` 网络及 `Pbkdf2Batch.sol`，计价参数通过算法版本提案发布，不能写入 genesis。

EvoCrypt 的 `WithCloseOnContextDone` 对照补测见 `results/chapter-five/20261023-single-node-pbkdf2-no-termination/report.md`。仅隔离实验节点可设 `EVOCRYPT_EXPERIMENT_DISABLE_WAZERO_TERMINATION=1`，使其值为 `false`；正常节点默认仍为 `true`。关闭后 WASM 的上下文取消和执行超时不能终止失控计算，不应在生产或共享网络使用。补测仅收集 EvoCrypt 的 `eth_call` 往返时间，不测 Gas。

5.2.2 在**独立单节点链**中固定 10 次 PBKDF2 内部迭代，预注册单笔调用次数 `1,2,3,4,5,6,8,12,16`；仅携带计价参数的 PBKDF2 版本在激活块之后采用固定 `g₀=3000`、`N₀=2`、`α=1/2` 的实验计费。父区块倍率恒为 1（暂不实现跨区块自适应计费）。每个目标块只含一笔目标交易，其父块仅含一笔普通自转账；链上 `receipt.gasUsed` 是累计 Gas。先上传算法与部署 Solidity 参考合约（可复用 `benchexecutionefficiency -algorithms Pbkdf2Sha256 -pbkdf-iterations 10 -warmup 2 -n 2 -receipt-samples 0` 的 `setup.contract.address`），再将计价参数与版本 2 一起发布；采集器从链上核对生效版本参数后运行：

```bash
make geth
go run ./experiments/cryptoupgrade/scripts/render -config experiments/cryptoupgrade/config/local-1node-pricing.yaml -geth "$(pwd)/build/bin/geth"
./experiments/cryptoupgrade/output/network/local-1node-pricing/node1/start.sh
go run ./experiments/cryptoupgrade/scripts/benchexecutionefficiency \
  -algorithms Pbkdf2Sha256 -rpc http://127.0.0.1:8761 \
  -pbkdf-wasm experiments/cryptoupgrade/results/chapter-five/20261022-single-node-pbkdf2-rebuilt/pbkdf2_sha256.wasm \
  -pbkdf-iterations 10 -solc /home/liuqi/project/.tools/solc/solc-0.8.26 \
  -warmup 2 -n 2 -receipt-samples 0 \
  -output-json experiments/cryptoupgrade/output/results/billing-setup.json
go run ./experiments/cryptoupgrade/scripts/upload-wasm \
  -rpc http://127.0.0.1:8761 -key-file experiments/cryptoupgrade/config/signer.key \
  -chain-id 11223345 -name Pbkdf2Sha256 -mode immediate -version 2 \
  -wasm experiments/cryptoupgrade/results/chapter-five/20261022-single-node-pbkdf2-rebuilt/pbkdf2_sha256.wasm \
  -itype bytes,bytes,uint256,uint256 -otype bytes -algo-gas 3000 \
  -pricing experiments/cryptoupgrade/config/pbkdf2-pricing.json
REFERENCE_ADDRESS=$(python3 -c 'import json; print(json.load(open("experiments/cryptoupgrade/output/results/billing-setup.json"))["results"][0]["setup"]["contract"]["address"])')
go run ./experiments/cryptoupgrade/scripts/chapterfive/benchbilling \
  -rpc http://127.0.0.1:8761 -reference "$REFERENCE_ADDRESS" \
  -genesis experiments/cryptoupgrade/output/network/local-1node-pricing/genesis.json \
  -output experiments/cryptoupgrade/output/results/billing-recheck
python3 experiments/cryptoupgrade/scripts/chapterfive/plot_billing.py \
  experiments/cryptoupgrade/output/results/billing-recheck
```

新计费不改变原生预编译，也**不能**把人为配置的较低 Gas 当作执行时间优势，或把单笔交易内递增当作跨交易抗攻击能力。旧插件上传的版本索引在节点重启后可能无法恢复；运行前务必验证已激活版本/重新上传，不能默默将重启前后数据混采。

基于同一 5.2.2 配置的单因素敏感性实验将 `g₀` 设为 1500/3000/6000、`α` 设为 1/4、1/2、3/4，或将阈值 `N₀` 设为 1/2/4，其余参数固定。七个场景（共用一组基线）及选择依据在 `results/chapter-five/20261023-pricing-sensitivity/matrix.json` 中冻结。每个场景在独立单节点链上分别运行并上传相同的归档 PBKDF2 模块，费用变更必须发布新版本，不能原地修改已登记版本。依次运行和汇总：

```bash
go build -o experiments/cryptoupgrade/output/results/pricing-sensitivity/geth ./cmd/geth
python3 experiments/cryptoupgrade/scripts/chapterfive/run_pricing_sensitivity.py \
  --binary experiments/cryptoupgrade/output/results/pricing-sensitivity/geth
```

运行脚本需要 `PyYAML` 和绘图用的 `matplotlib`。每组正式记录 3 轮×每轮 10 张收据×2 路径×9 档调用次数，并在节点停止前重新向 RPC 核验收据、单目标交易区块及中性父块。中断或失败时保留已有链和原始数据，不替换失败档或无声地重试；完整数据合并为 `sensitivity-summary.csv`。开关实验的 `EVOCRYPT_EXPERIMENT_DISABLE_WAZERO_TERMINATION` 在这些计费节点始终不设置。

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

### SHA-256 轻量 WASM 升级 Gas 对照

`algorithm/go/sha256.go` 使用独立 SHA-256 实现，避免 `crypto/sha256` 的传递依赖扩大 WASM。使用 TinyGo 0.42.0（Go 1.25）生成模块，并测试 Go 与 WASM 输出：

```bash
go test ./experiments/cryptoupgrade/algorithm/go -run '^TestSha256' -count=1
go run ./cryptoupgrade/wasmtool/cmd/wasmbuild \
  -source experiments/cryptoupgrade/algorithm/go/sha256.go \
  -out experiments/cryptoupgrade/algorithm/go/wasm/sha256.wasm \
  -function Sha256 -itype bytes -otype bytes
go test ./cryptoupgrade/internal/wasmruntime -run '^TestSha256WasmExecute$' -count=1
```

比较升级 Gas 时，在全新单节点网络（`local-1node.yaml`，Clique `period: 0`）上运行更新后的 `geth`；不要把已上传算法的历史版本原地替换。先从优化前的 Git 快照提取原制品，然后用**长度相同**的名称、同一 ABI 和算法 Gas 配置分别上传：

```bash
git show e72de4db4cee2b02fe5b9ab131e807a7d854363f:experiments/cryptoupgrade/algorithm/go/wasm/sha256.wasm > /tmp/sha256-baseline.wasm
make geth
go run ./experiments/cryptoupgrade/scripts/render \
  -config experiments/cryptoupgrade/config/local-1node.yaml \
  -out "$(pwd)/experiments/cryptoupgrade/output/network/sha256-size-test" \
  -geth "$(pwd)/build/bin/geth"
# 在另一终端运行 CRYPTOUPGRADE_MODULE="$(pwd)" ./experiments/cryptoupgrade/output/network/sha256-size-test/node1/start.sh，宿主机 RPC 为 8545。
KEY=$(cat experiments/cryptoupgrade/config/signer.key)
go run ./experiments/cryptoupgrade/scripts/upload-wasm \
  -wasm /tmp/sha256-baseline.wasm -name Sha256BaselineA \
  -itype bytes -otype bytes -algo-gas 3000 -rpc http://127.0.0.1:8545 -key "$KEY"
go run ./experiments/cryptoupgrade/scripts/upload-wasm \
  -wasm experiments/cryptoupgrade/algorithm/go/wasm/sha256.wasm -name Sha256Optimized \
  -itype bytes -otype bytes -algo-gas 3000 -rpc http://127.0.0.1:8545 -key "$KEY"
```

本次新链收据对照：原 WASM 68,288 字节、编码 52,504 字节、863,152 Gas；优化后 8,167 字节、编码 6,448 字节、126,288 Gas，减少 736,864 Gas（85.37%）。交易数据、制品哈希和结果保存在 `output/results/sha256-optimized-20261022-172110/comparison.json`。小型 WASM 的执行需要更新后的运行时：其输入区位于模块初始化内存之外，避免固定 64 KiB 地址覆盖模块堆。

### 无 `crypto` 导入的实验算法

`PedersenCommit`、`Pbkdf2Sha256`、`Ed25519Verify` 分别在各自的独立 Go 源码中实现所需的 SHA-256、HMAC-SHA256 或 SHA-512/Ed25519 计算，不依赖 `crypto` 标准库。使用 TinyGo 0.42.0 / Go 1.25.0 和当前 `wasmbuild` 配置时，原始 WASM 大小对比如下（此处不是上链 Gas 测量）：

| 算法 | 原始制品 | 新制品 |
| --- | ---: | ---: |
| PedersenCommit | 95,780 B | 48,068 B |
| Pbkdf2Sha256 | 72,580 B | 12,191 B |
| Ed25519Verify | 无 | 39,440 B |

重建和验证命令：

```bash
go run ./cryptoupgrade/wasmtool/cmd/wasmbuild -source experiments/cryptoupgrade/algorithm/go/pedersen_commit.go -out experiments/cryptoupgrade/algorithm/go/wasm/pedersen_commit.wasm -function PedersenCommit -itype bytes,bytes -otype bytes
go run ./cryptoupgrade/wasmtool/cmd/wasmbuild -source experiments/cryptoupgrade/algorithm/go/pbkdf2_sha256.go -out experiments/cryptoupgrade/algorithm/go/wasm/pbkdf2_sha256.wasm -function Pbkdf2Sha256 -itype bytes,bytes,uint256,uint256 -otype bytes
go run ./cryptoupgrade/wasmtool/cmd/wasmbuild -source experiments/cryptoupgrade/algorithm/go/ed25519.go -out experiments/cryptoupgrade/algorithm/go/wasm/ed25519.wasm -function Ed25519Verify -itype bytes,bytes,bytes -otype bool
go test ./experiments/cryptoupgrade/algorithm/go ./cryptoupgrade/internal/wasmruntime
```

Ed25519 验签使用实验性 `math/big` 运算，并非恒定时间实现或经审计的生产密码库；制品缩小也不等于 Gas 必然同比减少，升级成本应使用实际交易收据测量。

### Schnorr 验签 WASM

`SchnorrVerify` 在算法源码中计算 SHA-256 挑战值，不再导入 `crypto/sha256`；原始 WASM 从 95,960 B 降为 48,630 B（TinyGo 0.42.0 / Go 1.25.0）。这只是制品体积对比，并非升级交易的 Gas 实测。重建与验证：

```bash
go run ./cryptoupgrade/wasmtool/cmd/wasmbuild -source experiments/cryptoupgrade/algorithm/go/schnorr_proof.go -out experiments/cryptoupgrade/algorithm/go/wasm/schnorr_proof.wasm -function SchnorrVerify -itype bytes,bytes -otype bool
go test ./experiments/cryptoupgrade/algorithm/go ./cryptoupgrade/internal/wasmruntime -run 'TestSchnorr' -count=1
```

新构建的 WASM wrapper 暴露 `allocate`，执行器使用模块自身的可复用输入缓冲区，避免连续调用时原先固定输入地址与模块堆重叠。未导出 `allocate` 的旧制品仍沿用兼容路径，已有链上版本不自动替换。

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

### 20 节点升级激活一致性实验

该实验只验证一次 `UpgradeConsistencyProbe` 的 v1→v2 版本选择和状态一致性，不测吞吐量/RPC 性能。详见 `experiments/cryptoupgrade/scripts/upgradeconsistency/README.md`。

## 随升级提案发布 Gas 参数

`upload-wasm -pricing <schedule.json>` 支持 `-mode version`（配合 `-activation-block`）和 `-mode immediate`。
文件是单个算法版本的完整配置，例如 `config/pbkdf2-pricing.json`：

```json
{
  "threshold": 2,
  "alpha": {"numerator": 1, "denominator": 2},
  "bands": [{"maxMetric": 18446744073709551615, "gas": 3000}]
}
```

每个算法、每个版本可使用不同参数。输入指标为 ABI 参数编码字节数，费用为
`ceil(B(inputBytes) * (1 + alpha)^max(0, index - threshold))`。
计数按交易和算法隔离。分档必须覆盖 uint64 大小范围，边界递增、Gas 为正且不递减，最多 64 档；alpha 分母为 1–10000。
参数由工具编码成 ABI bytes，同一笔升级交易提交，并通过 `getVersionPricing(name, version)` 查询。
未指定 `-pricing` 的提案继续按其 `-algo-gas` 固定收费；指定后使用该版本 Schedule，不叠加固定 Gas。

迁移旧实验时移除 YAML `cryptoupgrade.pricing` / genesis `config.cryptoUpgradePricing`，在新实验链发布带参数的算法版本。
旧全局配置会明确报错；`updataGas` 和修改已登记版本的提交也会被拒绝（相同提案允许幂等重放）。归档实验结果仍描述当时实现，不应混用为新方案的实测结果。
