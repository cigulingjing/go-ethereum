# Cryptoupgrade Solidity Contracts

本目录存放 cryptoupgrade 实验用的 Solidity 基准合约，用于与 WASM / Precompiled 方案对比。

## 目录结构

- `src/`：合约源码（含 `archive/` 历史基准实现）
- `test/`：Foundry 单元测试
- `script/`：部署脚本
- `lib/`：Foundry 依赖（`forge-std`）

## 部署

模拟（不上链）：

```shell
forge script script/DeployPolynomialMul.s.sol:DeployPolynomialMul \
  --rpc-url http://127.0.0.1:8761 \
  --private-key <PRIVATE_KEY> \
  -vvvv
```

广播部署（私有 Clique 网络推荐 legacy + 显式 gas price，避免 EIP-1559 替换冲突）：

```shell
forge script script/DeployPolynomialMul.s.sol:DeployPolynomialMul \
  --rpc-url http://127.0.0.1:8761 \
  --private-key 0x2aedaed0ff6818dbc349b870e384504aa50dfad34116c089ba58fc28638eb7a2 \
  --legacy \
  --with-gas-price 1000000000 \
  --broadcast \
  -vvvv
```

若出现 `replacement transaction underpriced`，说明 mempool 中已有同 nonce 的 pending 交易；不要重复 `--broadcast`，改用更高 gas price 或 `--legacy` 替换。

## 常用命令

```shell
forge build
forge test
forge fmt
```
