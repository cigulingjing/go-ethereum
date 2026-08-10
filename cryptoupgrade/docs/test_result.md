# Cryptoupgrade 性能测试结果

## 测试概述

本次实验包含两类结果：

- Add 基线实验：对比升级方案和传统最小 EVM 合约。
- Blake2b Sum256 三组实验：对比 Solidity 纯合约实现、升级方案 Go plugin 实现、客户端内置预编译合约实现。

部署耗时从 `eth_sendTransaction` 开始统计，到交易 receipt 返回为止。调用耗时是在目标算法或合约已经部署完成后，通过 `eth_call` 统计。

调用阶段 gas 通过 `eth_estimateGas` 统计。该值包含交易固有 gas、calldata gas 和实际执行路径的 gas 估算。

## Blake2b Sum256 三组实验

### 实现方式

| 实现方式 | 说明 | 是否存在部署阶段 |
| --- | --- | --- |
| Solidity 纯合约 | 通过 `cryptoupgrade/algorithm/contracts/Blake2b.sol` 在 Solidity 中实现 Blake2b-256，测试输入限制为不超过 128 字节。 | 是 |
| 升级方案 | 通过 `CodeStorage.uploadCode` 上传 Go wrapper，wrapper 调用 `github.com/ethereum/go-ethereum/crypto/blake2b.Sum256`。 | 是 |
| 预编译合约 | 在 geth 客户端中内置 `Blake2bSum256` 预编译合约，地址为 `0x0000000000000000000000000000000000000046`，直接调用 `go-ethereum/crypto/blake2b.Sum256`。 | 否 |

测试输入：

```text
Hello world!
```

期望 Blake2b-256 摘要：

```text
3fbc092db9350757e2ab4f7ee9792bfcd2f5220ada5a4bc684487f60c6034369
```

### 部署阶段

预编译合约已经内置在客户端中，不存在部署阶段。因此部署阶段只比较 Solidity 纯合约和升级方案。

| 算法 | 实现方式 | Mean | P50 | P95 | Gas Mean |
| --- | --- | ---: | ---: | ---: | ---: |
| Blake2b Sum256 | Solidity 纯合约 | 1.006741755s | 1.006716617s | 1.008646414s | 852059 |
| Blake2b Sum256 | 升级方案 | 1.709387805s | 1.51039862s | 2.011335634s | 87180 |

部署阶段比值：

| 比值 | Mean | P50 | P95 |
| --- | ---: | ---: | ---: |
| 升级方案 / Solidity 纯合约 | 1.70x | 1.50x | 1.99x |

### 调用阶段

调用阶段三组实现都参与对比：Solidity 纯合约、升级方案和预编译合约。

| 算法 | 实现方式 | Mean | P50 | P95 | Gas Estimate | 返回值 |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| Blake2b Sum256 | Solidity 纯合约 | 1.021828ms | 883.1us | 1.7781ms | 112127 | `3fbc092db9350757e2ab4f7ee9792bfcd2f5220ada5a4bc684487f60c6034369` |
| Blake2b Sum256 | 升级方案 | 533.229us | 470.9us | 895us | 24779 | `3fbc092db9350757e2ab4f7ee9792bfcd2f5220ada5a4bc684487f60c6034369` |
| Blake2b Sum256 | 预编译合约 | 479.634us | 413.2us | 875.1us | 21646 | `3fbc092db9350757e2ab4f7ee9792bfcd2f5220ada5a4bc684487f60c6034369` |

调用阶段比值：

| 比值 | Mean | P50 | P95 |
| --- | ---: | ---: | ---: |
| 升级方案 / Solidity 纯合约 | 0.52x | 0.53x | 0.50x |
| 升级方案 / 预编译合约 | 1.11x | 1.14x | 1.02x |
| Solidity 纯合约 / 预编译合约 | 2.13x | 2.14x | 2.03x |

## Add 基线实验

| 算法 | 测试阶段 | 输入 | 升级方案 Mean | 升级方案 P50 | 升级方案 P95 | 升级方案 Gas | 传统方案 Mean | 传统方案 P50 | 传统方案 P95 | 传统方案 Gas | Mean 比值 |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Add | 部署 | `100 + 100` | 1.709800323s | 1.512630436s | 2.008988498s | 85612 | 1.007160836s | 1.005923499s | 1.0113675s | 60918 | 1.70x |
| Add | 已部署调用 | `100 + 100` | 450.997us | 414us | 660.3us | - | 416.851us | 388.1us | 626.5us | - | 1.08x |

## 结论

- 部署阶段只比较 Solidity 纯合约和升级方案；预编译合约属于客户端内置能力，不存在部署阶段。
- Blake2b Sum256 部署阶段中，升级方案耗时高于 Solidity 纯合约，Mean 约为 `1.70x`，主要原因是上传阶段包含 Go plugin 编译和激活。
- Blake2b Sum256 部署 gas 中，升级方案为 `87180`，Solidity 纯合约为 `852059`，升级方案部署 gas 明显更低。
- Blake2b Sum256 调用阶段中，升级方案明显快于 Solidity 纯合约，Mean 约为 Solidity 的 `0.52x`，gas 估算约为 Solidity 的 `0.22x`。
- Blake2b Sum256 调用阶段中，升级方案与预编译合约接近，Mean 约为预编译合约的 `1.11x`，gas 估算略高于预编译合约。
- 预编译合约 gas 规则使用与 SHA256 预编译相同的 base/per-word 形式；本次 12 字节输入下，`eth_estimateGas` 为 `21646`。
