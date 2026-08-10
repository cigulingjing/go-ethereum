## 1. 候选选择

- [x] 1.1 将 blockchain-crypto 类别映射为实际可行的单文件纯 Go 候选。
- [x] 1.2 记录不适合转换的类别及跳过原因。

## 2. 候选实现

- [x] 2.1 在 `cryptoupgrade/algorithm` 下增加独立候选文件。
- [x] 2.2 保持候选入口为导出函数，并避免 import blockchain-crypto。

## 3. 验证

- [x] 3.1 对修改的 Go 文件运行 `gofmt`/`goimports`。
- [x] 3.2 使用 `go build -buildmode=plugin -tags=urfave_cli_no_docs,ckzg -trimpath` 构建每个新候选。
- [x] 3.3 验证 OpenSpec 制品。

## 4. Solidity 对照

- [x] 4.1 为实际可行的确定性或可验证候选操作增加 Solidity 对照实现。
- [x] 4.2 记录跳过的 Solidity 操作及原因。
- [x] 4.3 使用 solc 编译 Solidity 对照合约。
