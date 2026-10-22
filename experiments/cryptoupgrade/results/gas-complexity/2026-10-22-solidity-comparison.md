# 动态 Gas 与 Solidity 对照（固定输入，有限范围）

**实验性质**：Solidity 与旧版 WASM 是隔离单节点的真实 `receipt.gasUsed`；新费率尚未接入链上，比较列是固定其他开销后的**离线反事实估算**，不是新费率交易收据。

## 范围和方法

- 八算法报价取标定参考输入；Blake2b 对照改取已测的 32 B 档，因为 Solidity 合约拒绝 256 B。DH 公钥对 Solidity 左填充至 256 B，标定使用 1 B，逻辑整数同为 4，但 ABI 输入不完全相同。
- 固定 `α=1/4`、`N₀=2`、`C∈{0,1,2,3,4}`（`m=1.0…1.4`）、`i∈{1,2,3,4,5}`（单笔交易内同算法调用序号），共 25 个报价点/算法；`i>1` 不能当作 5 笔独立交易的序号。
- `G_proj(C,i) = G_WASM,legacy(receipt) − G_alg,legacy + Quote(C,i)`，与**相同算法及输入**的 `G_Solidity(receipt)` 对比。这里假设替换算法费用不改变固有 Gas、calldata 或封装费用；对 `i>1` 只是以单次调用交易封装估算边际费用，并非批处理总 Gas。
- 区块 Gas 上限 30,000,000；历史 WASM 费用只是扣费差分基准，不等于新计价已启用。DH 仅为相同计算值的示意性比较。

## 结果

| 算法 | 输入档 | 基础费 | Solidity 实收 | 新费率估算 C=0,i=1 | 新费率估算 C=4,i=5 | 低于 Solidity / 25 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Add | 64 | 100 | 21,738 | 22,336 | 22,510 | 0/25 |
| Sha256 | 256 | 740 | 26,365 | 26,952 | 28,236 | 0/25 |
| Blake2bSum256 | 32 | 160 | 113,670 | 22,776 | 23,054 | 25/25 |
| Pbkdf2Sha256 | 500 | 300,600 | 12,474,307 | 323,748 | 845,102 | 25/25 |
| Dh2048Secret† | 128 | 2,078,080 | 410,905 | 2,101,736 | 5,705,906 | 0/25 |
| PedersenCommit | 128 | 2,074,300 | — | — | — | — |
| SchnorrVerify | 256 | 7,957,400 | 2,056,684 | 7,987,456 | 21,788,572 | 0/25 |
| PolynomialMul | 4 | 220 | 33,745 | 23,972 | 24,354 | 25/25 |

† DH 输入编码不同；不可作为严格同一 ABI 输入的对照。Pedersen 的已标定 128 bit 和 16 bit 输入在 Solidity 合约执行时回退，不能填造对照收据。

每个算法 25 个逐点报价、真实交易哈希、旧费用和对照差值见同名 JSON。重复调用报价超过单次交易上限或累计超过区块上限时，交易会失败；表中的端点只是数学估算，不保证可执行。

## 复现

在隔离链上使用 `benchtxgas -reference-inputs -algorithms Add,Sha256`、`-algorithms Blake2bSum256,Pbkdf2Sha256`、`-algorithms Dh2048Secret`、`-algorithms SchnorrVerify,PolynomialMul` 分批采集，对应原始 JSON 文件为 `2026-10-22-solidity-{first,middle,dh,tail}.json`。Pedersen 的 128/16 bit 两档 `eth_call` 均回退，故不纳入收据比较。用 `go run ./experiments/cryptoupgrade/scripts/benchcallgas -blake32` 生成 25 点报价，再执行 `python3 experiments/cryptoupgrade/scripts/benchcomparegas/compare.py --help` 查看对照命令参数；不得用历史默认输入的收据直接替换。需要动态计价完成链上集成后，才能测量真实的新费率 `receipt.gasUsed`。
