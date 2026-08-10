# 设计：cryptoupgrade 基准套件

## 架构

### 升级方案

升级方案通过 `CodeStorage.uploadCode` 接收 gzip + base64 后的 Go 源码，geth 解压、编译为 Go plugin，并记录算法元信息。调用阶段通过 `CodeStorage.callFunc` 进入：

```text
eth_call
  -> EVM CodeStorage address
  -> cryptoupgrade.RunCodeStorageCall
  -> ABI unpack callFunc(name,input)
  -> plugin.Open / Lookup
  -> reflect.Value.Call
  -> ABI pack output
```

### Solidity 方案

Solidity 方案通过 `cryptoupgrade/contracts/Blake2bSolidity.sol` 实现 Blake2b-256。该合约不调用预编译合约，算法主体在 Solidity/EVM 中执行。

### 预编译方案

预编译方案部署一个轻量 EVM adapter，对外暴露 `sum256(bytes) -> bytes32`，内部构造 BLAKE2F 输入并调用地址 `0x09` 的 BLAKE2F 预编译合约。

## Benchmark 命令

### Add 基线

`cryptoupgrade/cmd/benchcall` 用于 Add 基线实验：

- 升级方案：上传并调用 `cryptoupgrade/algorithm/add.go`。
- 传统方案：部署手写最小 EVM add 合约。

### Blake2b 三组 benchmark

`cryptoupgrade/cmd/benchblake2b` 用于 Blake2b Sum256 三组实验：

- `--mode deploy`：部署阶段统计。
- `--mode call`：已部署调用阶段统计。
- `--mode all`：部署和调用一起执行，适合 smoke test。

## 测量定义

| 指标 | 定义 |
| --- | --- |
| Deployment elapsed | 从 `eth_sendTransaction` 到 receipt 返回。 |
| Deployment gas | receipt 中的 `gasUsed`。 |
| Call elapsed | 对已部署目标执行 `eth_call` 的客户端侧耗时。 |
| Call gas | 规范要求通过 `eth_estimateGas` 或等价方式采集；当前结果文档优先记录调用耗时。 |

## 重要设计决策

- 部署测试使用固定 gas limit，不在 `uploadCode` 前执行 `eth_estimateGas`，避免估算触发 Go plugin 编译副作用。
- Go plugin 编译支持 `CRYPTOUPGRADE_GO` 和 `CRYPTOUPGRADE_MODULE`，保证插件可以使用与 geth 主进程一致的构建环境。
- Blake2b Solidity 合约使用 `--via-ir --optimize` 编译，避免复杂局部变量导致 `Stack too deep`。
- 结果文档以用户关注的资源指标为中心，而不是仅输出命令日志。
