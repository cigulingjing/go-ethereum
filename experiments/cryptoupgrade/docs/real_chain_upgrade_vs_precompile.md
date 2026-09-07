# Cryptoupgrade 真实链基准测试

本文档记录升级方案和预编译合约方案在本地 Geth 链上的基准测试方法。测试链默认使用 `http://127.0.0.1:8666` JSON-RPC 端口。

## 测试目标

- 升级方案：通过 `CodeStorage.uploadCode` 上传算法，再通过 `CodeStorage.callFunc(name, encodedInput)` 调用。
- 预编译合约方案：直接向指定的原生预编译地址发送同一份 ABI 编码输入。
- 测量内容：升级方案 setup 交易成本、预编译 setup 零成本、两条调用路径的 `eth_call` 延迟、`eth_estimateGas`、输出一致性和比值。

## 默认 SHA-256 实验

输入明文为：

```text
hello cryptoupgrade
```

ABI 编码输入为 `abi.encode(bytes("hello cryptoupgrade"))`：

```text
0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001368656c6c6f2063727970746f7570677261646500000000000000000000000000
```

期望输出为 `abi.encode(bytes(sha256(input)))`：

```text
0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000201dfa30acdf489c5a6ad291902fd7c58449fd325e8e64d0d45b704e8f4c453414
```

预编译地址为 `0x0000000000000000000000000000000000000048`。

## 执行命令

```shell
go run ./experiments/cryptoupgrade/bench/cmd/benchrealchain \
  -rpc http://127.0.0.1:8666 \
  -name Sha256 \
  -source cryptoupgrade/algorithm/wasm/sha256.wasm \
  -itype bytes \
  -otype bytes \
  -input-hex 0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001368656c6c6f2063727970746f7570677261646500000000000000000000000000 \
  -expected-hex 0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000201dfa30acdf489c5a6ad291902fd7c58449fd325e8e64d0d45b704e8f4c453414 \
  -precompile-address 0x0000000000000000000000000000000000000048 \
  -warmup 10 \
  -n 100
```

`-warmup` 控制每条调用路径正式采样前的预热次数，`-n` 控制正式采样次数。命令会先上传算法，等待事件监听线程完成本地 activation 并通过 `CodeStorage.callFunc` 可调用后，再对升级路径和预编译路径分别执行预热和采样。未显式设置 `-output-json` 时，JSON 结果默认写入 `experiments/cryptoupgrade/results/real-chain/run-<timestamp>/result.json`。

## 输出说明

文本输出包含：

- setup 阶段：升级方案上传交易数量、receipt status、耗时、gas、源码大小和压缩后大小；预编译方案 setup 交易数和 gas 固定为 `0`。
- 调用阶段：两条路径的 first、mean、p50、p95、min、max 延迟，以及 `eth_estimateGas`。
- 比值：`upgrade / precompile` 的 mean、p50、p95、gas 比值，并标记输出是否一致。

JSON 输出包含同样字段，便于后续脚本汇总多组算法实验。

## 常用变体

只测上传成本：

```shell
go run ./experiments/cryptoupgrade/bench/cmd/benchrealchain -mode setup -n 1 -warmup 0
```

复用已经上传过的算法，只测调用成本：

```shell
go run ./experiments/cryptoupgrade/bench/cmd/benchrealchain -upload=false -n 100 -warmup 10
```

更换算法时，需要同时调整 `-name`、`-source`、`-itype`、`-otype`、`-input-hex`、`-expected-hex` 和 `-precompile-address`。

## 本机验证记录

验证时间：2026-07-14。

本地 `http://127.0.0.1:8666` RPC 可访问，`eth_chainId` 返回 `0xab4130`，账户列表返回 `0x71562b71999873db5b286df957af199ec94617f7`。

按本文档的完整命令执行时，上传交易已经发送，但 2 分钟内没有返回 receipt：

```text
timed out waiting for receipt 0xfe45ae319c734c300fde44924a204efb3e18ddd3fd4a8c03ef17fdebff5a5d62
```

进一步检查节点状态：

```text
eth_blockNumber = 0x1
eth_syncing.currentBlock = 0x1
eth_syncing.highestBlock = 0x2
```

在该状态下，直接 `eth_call` 预编译地址也会失败：

```text
historical state 44558bbcebd211d272df58f6c7d2ff3b5569ff24af2678fd1a2ac343f1450e9f is not available
```

因此本次只能验证 RPC 可达和命令执行路径，不能记录有效性能样本。待本地链完成同步并正常出块后，重新运行上文完整命令即可在 `experiments/cryptoupgrade/results/real-chain/run-<timestamp>/result.json` 生成结果。

## 本机验证记录 2026-08-08

本地 `geth` 进程已监听 `0.0.0.0:8666`，启动参数与 `experiments/cryptoupgrade/docs/chain_setup.md` 中的 `--dev --datadir chain/node1 --http.port 8666` 形式一致。

当前 shell 配置了 HTTP 代理，访问本地 RPC 时需要绕过代理：

```shell
NO_PROXY=127.0.0.1,localhost no_proxy=127.0.0.1,localhost \
ALL_PROXY= HTTP_PROXY= HTTPS_PROXY= all_proxy= http_proxy= https_proxy= \
go run ./experiments/cryptoupgrade/bench/cmd/benchrealchain ...
```

绕过代理后，`web3_clientVersion`、`eth_chainId` 和 `eth_accounts` 均可正常返回：

```text
web3_clientVersion = Geth/v1.17.5-unstable-82c7b26c-20260801/linux-amd64/go1.25.0
eth_chainId = 0xab4130
eth_accounts = ["0xf5f871aa6bd253914705898c66251f994aa426fa"]
```

按 SHA-256 完整命令执行时，`CodeStorage.uploadCode` 上传交易未能发送成功：

```text
send upload transaction: insufficient funds for gas * price + value: balance 0, tx cost 16000000008000000, overshot 16000000008000000
```

进一步检查链状态：

```text
eth_blockNumber = 0x0
eth_getBalance(0xf5f871aa6bd253914705898c66251f994aa426fa) = 0x0
debug_accountRange = {}
```

该状态说明当前 `--dev` 启动没有使用带账户 `alloc` 的 `chain/genesis.json` 状态，链上没有可支付上传交易 gas 的账户。因此本次仍不能生成有效性能样本。后续需要使用包含 funded sender 的链状态启动节点，或通过 `-from` 指定链上已有余额且节点可签名的账户后重新运行完整命令。

随后使用不带 `--dev` 的私链启动命令重新验证：

```shell
./geth \
  --datadir chain/node1 \
  --password chain/password.txt \
  --networkid 11223344 \
  --http \
  --http.addr "0.0.0.0" \
  --http.port 8666 \
  --http.corsdomain "*" \
  --http.vhosts "*" \
  --http.api "web3,eth,debug,net,admin"
```

本次 `CodeStorage` 预部署合约已经存在，`chain/genesis.json` 中的 funded account 也已经生效：

```text
eth_getCode(0x0000000000000000000000000000000000000043) != 0x
eth_getBalance(0xc4e5e9b04769b82ce36f4aef4ead980e2ff82dc4) = 0x65a4da25d3016c00000
eth_accounts = ["0xf5f871aa6bd253914705898c66251f994aa426fa"]
eth_getBalance(0xf5f871aa6bd253914705898c66251f994aa426fa) = 0x0
```

用默认账户 `0xf5f...26fa` 运行完整 benchmark 仍会因为余额为 0 失败；改用 funded account `0xc4e5...2dc4` 作为 `-from` 时，节点返回：

```text
send upload transaction: unknown account
```

因此不带 `--dev` 后链状态已经正确加载了预部署合约和 funded account，但 funded account 不在当前节点 keystore 中，当前节点 keystore 中的账户又没有余额。完整 benchmark 需要满足同一个地址同时具备两项条件：链上有余额，并且节点本地可签名。

再次重新初始化并启动后，账户与 signer 已经统一为 `0xf5f871aa6bd253914705898c66251f994aa426fa`，并且该账户已有余额：

```text
eth_accounts = ["0xf5f871aa6bd253914705898c66251f994aa426fa"]
eth_getBalance(0xf5f871aa6bd253914705898c66251f994aa426fa) = 0x65a4da25d3016c00000
genesis signer = 0xf5f871aa6bd253914705898c66251f994aa426fa
```

预编译路径可通过 `eth_call` 直接返回期望 SHA-256 输出，但完整 benchmark 的上传交易仍失败：

```text
send upload transaction: authentication needed: password or unlock
```

该状态说明账户已有余额但未解锁。由于当前 HTTP API 未暴露可用的在线解锁接口，后续启动节点时需要同时解锁 signer/sender，例如：

```shell
./geth \
  --datadir chain/node1 \
  --password chain/password.txt \
  --networkid 11223344 \
  --unlock 0xF5F871aA6Bd253914705898c66251f994aa426FA \
  --allow-insecure-unlock \
  --mine \
  --http \
  --http.addr "0.0.0.0" \
  --http.port 8666 \
  --http.corsdomain "*" \
  --http.vhosts "*" \
  --http.api "web3,eth,debug,net,admin"
```

其中 `--unlock` 解决 `eth_sendTransaction` 本地签名问题，`--mine` 用于让 Clique 私链在收到上传交易后出块并返回 receipt。

## Dev 模式验证记录 2026-08-08

考虑到本阶段实验核心是验证升级机制本身，而不是验证 Clique 私链账户解锁流程，本次使用独立 dev 链完成完整 benchmark。当前 `geth` 二进制不支持 `--unlock` 参数；dev 模式会自动创建一个预分配且已解锁的 developer account，适合本地迭代测试上传交易。

从 `build/bin` 目录启动 dev 节点：

```shell
./geth \
  --dev \
  --datadir chain/dev-bench-8666 \
  --networkid 11223344 \
  --http \
  --http.addr 0.0.0.0 \
  --http.port 8666 \
  --http.corsdomain "*" \
  --http.vhosts "*" \
  --http.api "web3,eth,debug,net,admin"
```

RPC 状态：

```text
eth_chainId = 0x539
eth_accounts = ["0x71562b71999873db5b286df957af199ec94617f7"]
eth_getBalance(0x71562b71999873db5b286df957af199ec94617f7) = 0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7
```

执行 benchmark 时继续显式绕过本机代理：

```shell
NO_PROXY=127.0.0.1,localhost no_proxy=127.0.0.1,localhost \
ALL_PROXY= HTTP_PROXY= HTTPS_PROXY= all_proxy= http_proxy= https_proxy= \
go run ./experiments/cryptoupgrade/bench/cmd/benchrealchain \
  -rpc http://127.0.0.1:8666 \
  -name Sha256 \
  -source cryptoupgrade/algorithm/wasm/sha256.wasm \
  -itype bytes \
  -otype bytes \
  -input-hex 0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001368656c6c6f2063727970746f7570677261646500000000000000000000000000 \
  -expected-hex 0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000201dfa30acdf489c5a6ad291902fd7c58449fd325e8e64d0d45b704e8f4c453414 \
  -precompile-address 0x0000000000000000000000000000000000000048 \
  -warmup 10 \
  -n 100 \
  -output-json experiments/cryptoupgrade/results/real-chain/sha256-dev/result.json
```

本次完整结果已写入 `experiments/cryptoupgrade/docs/real_chain_upgrade_vs_precompile_result.json`。该文件保留为历史运行记录；新的运行结果默认写入 `experiments/cryptoupgrade/results/real-chain/run-<timestamp>/result.json`，也可以通过 `-output-json` 显式指定到 `experiments/cryptoupgrade/results/real-chain/<run-id>/result.json`。关键结果：

| 指标 | upgrade | precompile |
| --- | ---: | ---: |
| setup txs | 1 | 0 |
| setup gas | 87180 | 0 |
| setup elapsed | 410.855ms | 0ms |
| call mean | 1.123ms | 0.726ms |
| call p50 | 1.047ms | 0.671ms |
| call p95 | 1.700ms | 1.065ms |
| call gas estimate | 25801 | 25156 |

比值：

```text
mean upgrade/precompile = 1.55x
p50 upgrade/precompile = 1.56x
p95 upgrade/precompile = 1.60x
gas upgrade/precompile = 1.03x
outputMatched = true
```
