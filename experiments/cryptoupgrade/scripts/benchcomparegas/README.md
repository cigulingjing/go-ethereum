# 固定输入动态 Gas / Solidity 对照

从仓库根目录在隔离环境运行；需要当前项目的 `build/bin/geth` 和 `solc-0.8.26`。**不得连接生产节点**。先生成单节点实验链（示例端口 8545），在另一终端启动并保持运行：

```bash
go run ./experiments/cryptoupgrade/scripts/render -config experiments/cryptoupgrade/config/local-1node.yaml -out /tmp/cryptogas-reference-chain -geth "$(pwd)/build/bin/geth"
CRYPTOUPGRADE_MODULE="$(pwd)" /tmp/cryptogas-reference-chain/node1/start.sh
```

新的数据目录可避免旧上传版本影响结果。运行真实交易（Solidity 和旧版 WASM 同一组计算输入；仅提交成功的分组）：

```bash
RESULTS=experiments/cryptoupgrade/results/gas-complexity
SOLC=/home/liuqi/project/.tools/solc/solc-0.8.26
for entry in 'first Add,Sha256' 'middle Blake2bSum256,Pbkdf2Sha256' 'dh Dh2048Secret' 'tail SchnorrVerify,PolynomialMul'; do
  read -r label algorithms <<< "$entry"
  go run ./experiments/cryptoupgrade/scripts/benchtxgas -rpc http://127.0.0.1:8545 -reference-inputs -algorithms "$algorithms" -solc "$SOLC" -output-json "$RESULTS/2026-10-22-solidity-$label.json"
done
go run ./experiments/cryptoupgrade/scripts/benchcallgas -blake32 -out "$RESULTS/2026-10-22-solidity-range-model.json"
python3 experiments/cryptoupgrade/scripts/benchcomparegas/compare.py \
  --calibration "$RESULTS/2026-10-22-calibration.json" \
  --model "$RESULTS/2026-10-22-solidity-range-model.json" \
  --baseline "$RESULTS/2026-10-22-solidity-first.json" \
  --baseline "$RESULTS/2026-10-22-solidity-middle.json" \
  --baseline "$RESULTS/2026-10-22-solidity-dh.json" \
  --baseline "$RESULTS/2026-10-22-solidity-tail.json" \
  --out "$RESULTS/2026-10-22-solidity-comparison.json" \
  --report "$RESULTS/2026-10-22-solidity-comparison.md"
```

单节点交易和模拟价格不可混为一谈：执行新费率所需的链上集成尚未完成。`i=2..5` 是**同一交易中**的重复调用序号，当前链上四组测试实际只进行单次调用；差分后的报价仅用于比较边际费用。`C=0..4` 是假设的父区块计数，未由本次链上区块自动观测。Blake2b 的合约限制输入至 128 B，故使用标定记录中的 32 B 档；DH 的公钥必须编码成 256 B，和标定时的 1 B 形式表示同一个整数；Pedersen 的标定 128 bit 与 16 bit 输入在当前 Solidity 合约 `eth_call` 中回退，缺失基线不强行填充。

可运行 `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s experiments/cryptoupgrade/scripts/benchcomparegas` 复核比较脚本。
