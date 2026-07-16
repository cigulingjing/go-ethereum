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
go run ./cryptoupgrade/cmd/benchrealchain \
  -rpc http://127.0.0.1:8666 \
  -name Sha256 \
  -source cryptoupgrade/algorithm/sha256.go \
  -itype bytes \
  -otype bytes \
  -input-hex 0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001368656c6c6f2063727970746f7570677261646500000000000000000000000000 \
  -expected-hex 0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000201dfa30acdf489c5a6ad291902fd7c58449fd325e8e64d0d45b704e8f4c453414 \
  -precompile-address 0x0000000000000000000000000000000000000048 \
  -warmup 10 \
  -n 100 \
  -output-json cryptoupgrade/docs/real_chain_upgrade_vs_precompile_result.json
```

`-warmup` 控制每条调用路径正式采样前的预热次数，`-n` 控制正式采样次数。命令会先上传算法，随后对升级路径和预编译路径分别执行预热和采样。

## 输出说明

文本输出包含：

- setup 阶段：升级方案上传交易数量、receipt status、耗时、gas、源码大小和压缩后大小；预编译方案 setup 交易数和 gas 固定为 `0`。
- 调用阶段：两条路径的 first、mean、p50、p95、min、max 延迟，以及 `eth_estimateGas`。
- 比值：`upgrade / precompile` 的 mean、p50、p95、gas 比值，并标记输出是否一致。

JSON 输出包含同样字段，便于后续脚本汇总多组算法实验。

## 常用变体

只测上传成本：

```shell
go run ./cryptoupgrade/cmd/benchrealchain -mode setup -n 1 -warmup 0
```

复用已经上传过的算法，只测调用成本：

```shell
go run ./cryptoupgrade/cmd/benchrealchain -upload=false -n 100 -warmup 10
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

因此本次只能验证 RPC 可达和命令执行路径，不能记录有效性能样本。待本地链完成同步并正常出块后，重新运行上文完整命令即可生成 `cryptoupgrade/docs/real_chain_upgrade_vs_precompile_result.json`。
