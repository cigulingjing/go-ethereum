# 应用：cryptoupgrade 基准套件

## 实现摘要

本次实现已经完成，并映射到 OpenSpec tasks 中的全部任务。

## 代码制品

| 文件 | 作用 |
| --- | --- |
| `cryptoupgrade/cmd/benchcall/main.go` | Add 算法部署与调用基线实验。 |
| `cryptoupgrade/cmd/benchblake2b/main.go` | Blake2b Sum256 三组实验。 |
| `cryptoupgrade/contracts/Blake2bSolidity.sol` | Solidity 纯 Blake2b-256 实现。 |
| `cryptoupgrade/algorithm/blake2b.go` | 可上传的 standalone Blake2b 算法源文件。 |
| `cryptoupgrade/plugin.go` | Go plugin 编译参数、Go 二进制选择、module 工作目录支持。 |
| `cryptoupgrade/docs/test_result.md` | 实验结果与结论。 |

## 验证命令

```bash
CGO_ENABLED=1 CC=/home/liuqi/project/.tools/zig-cc \
  /home/liuqi/project/.tools/go1.22.5/go/bin/go test \
  -tags urfave_cli_no_docs,ckzg \
  ./cryptoupgrade/cmd/benchblake2b ./cryptoupgrade ./core/vm ./crypto/blake2b
```

## Benchmark 命令

### Blake2b 部署测试

```bash
CGO_ENABLED=1 CC=/home/liuqi/project/.tools/zig-cc \
  /home/liuqi/project/.tools/go1.22.5/go/bin/go run \
  -tags urfave_cli_no_docs,ckzg \
  ./cryptoupgrade/cmd/benchblake2b \
  --mode deploy --deploy-n 5 \
  --rpc http://127.0.0.1:8545 \
  --name Sum256 \
  --input 'Hello world!'
```

### Blake2b 已部署调用测试

```bash
CGO_ENABLED=1 CC=/home/liuqi/project/.tools/zig-cc \
  /home/liuqi/project/.tools/go1.22.5/go/bin/go run \
  -tags urfave_cli_no_docs,ckzg \
  ./cryptoupgrade/cmd/benchblake2b \
  --mode call --upload=false \
  --solidity-address 0xB25655a694886557fE3c52177c70b058b120e2b1 \
  --precompile-address 0x89c4B4c8d829181c03681b7EAa0e4C21d387769d \
  --n 1000 --warmup 100 \
  --rpc http://127.0.0.1:8545 \
  --name Sum256 \
  --input 'Hello world!'
```

## 结果

结果已记录到：

```text
cryptoupgrade/docs/test_result.md
```

核心结论：

- Blake2b 调用阶段升级方案快于 Solidity 纯合约实现。
- Blake2b 调用阶段升级方案接近预编译合约适配。
- 部署阶段升级方案耗时更高，但部署 gas 明显低于 Solidity 纯合约。
