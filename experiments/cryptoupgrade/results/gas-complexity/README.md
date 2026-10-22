# 八算法热执行复杂度与基础 Gas 候选值（2026-10-22）

## Solidity 对照区间

`2026-10-22-solidity-comparison.md` 对照父区块调用量 `C=0..4`、同笔交易同算法第 `i=1..5` 次调用的 25 点动态报价与相同计算输入的 Solidity 实际交易收据。`2026-10-22-solidity-comparison.json` 包含逐点差值及交易哈希，`2026-10-22-solidity-{first,middle,dh,tail}.json` 为四批链上原始记录，`2026-10-22-solidity-range-model.json` 为离线报价。**动态计价尚未接入链上；对照的“新费率交易 Gas”属于固定旧交易开销后的反事实估算，并非真实收据。** Pedersen 的两个已测规模在 Solidity 合约中回退，DH 的公钥 ABI 编码长度不同；参见对照报告的边界说明。

本报告将提案中的 `g₀(a,x)` 称为**算法基础 Gas**，不是 EIP-1559 的区块 `baseFeePerGas`。这里只完成复杂度标定，不启用新计价规则。

## 测量方法

- 在 Linux/amd64、Intel Core i7-10700、Go 1.25.0、TinyGo 0.42.0 上，以 wazero 运行八个重新编译的 WASM；每档预热 10 次，取 51 次调用中位数，独立运行 3 轮，再取三个中位数的中位数。每次包含 WASM 输入写入、模块调用与输出读取，但**不包含** ABI 打包、RPC、区块执行和交易基础 Gas。
- 原仓库 WASM 文件在较高重复次数下出现相同输入返回不同输出；实验使用 `build-fixtures.sh` 从算法源码重新编译，并在 `wasmtool` 生成的 wrapper 中保持输出缓冲区引用。JSON 中记录每个模块的 SHA-256，每个采样点也校验输出哈希一致；有效 Schnorr proof 必须返回 `true`。结果只适用于所列模块哈希。
- 以 64 位 Add 的 3 轮中位数之中位数 `500 ns` 为基准，选择实验锚点 `100 Gas`，按照 `ceil(100 × 测量耗时 / 500 ns)` 得到候选值。这一缩放系数是**实验选择**而非测量推导；若沿用原有 Add=3000 Gas，则有效 Schnorr 验签约需 2.39 亿 Gas，无法纳入当前实验配置的 3000 万 Gas 区块。选择 100 Gas 仅为已测输入留出空间，不保证所有输入或以后 `m_k`、`α` 都可执行。

## 参考输入与建议基础费

| 算法 | 复杂度主变量 | 参考输入 | 三轮中位数的中位数 | `g₀` 候选 Gas |
| --- | --- | ---: | ---: | ---: |
| Add | 操作数位数，EVM 固定 256 位 | 64 bit | 500 ns | 100 |
| Sha256 | 消息字节数 | 256 B | 3,700 ns | 740 |
| Blake2bSum256 | 消息字节数 | 256 B | 1,200 ns | 240 |
| Pbkdf2Sha256 | 迭代次数（32 字节输出） | 500 | 1,503,000 ns | 300,600 |
| Dh2048Secret | 私钥指数位数（固定模数） | 128 bit | 10,390,400 ns | 2,078,080 |
| PedersenCommit | 消息标量位数（盲因子固定） | 128 bit | 10,371,500 ns | 2,074,300 |
| SchnorrVerify | 消息字节数（有效 proof） | 256 B | 39,787,000 ns | 7,957,400 |
| PolynomialMul | 左右多项式项数（等长） | 4×4 | 1,100 ns | 220 |

## 输入规模与计算复杂度

- **Add：** 源码中的 `big.Int.Add` 随整数位数近似线性；ABI 将操作数限制在 256 位，因此链上视为 `O(1)`。测试 8/64/128/254 bit，耗时 500/500/600/600 ns。
- **Sha256 / Blake2bSum256：** 分块哈希，消息长度 `n` 的工作量均为 `O(n)`；32→4096 B 的测量分别为 1,100→43,400 ns 和 800→12,600 ns。
- **Pbkdf2Sha256：** 工作量近似 `O(iterations × ceil(keyLength/32))`，密码及盐长度另有哈希开销；固定 32 B 输出，100→2000 次为 283,600→5,725,900 ns。不能只依据 ABI 输入字节数计费。
- **Dh2048Secret：** 固定 2048 位模数下，模幂随私钥指数位数 `b` 增长，近似 `O(b × M(2048))`；16→256 bit 为 319,500→19,642,800 ns，短指数有明显固定开销。
- **PedersenCommit：** 两次固定模数模幂，近似 `O((b_message+b_blinding) × M(2048))`；固定盲因子，16→256 bit 为 510,600→19,943,500 ns。
- **SchnorrVerify：** 有效证明的两次模幂主导开销，固定群参数和 proof 大小时近似常数，消息哈希另有 `O(n)`；16→1024 B 为约 39.7–40.2 ms，样本波动大于消息长度效应，应对该范围使用不低于最大观测成本的保守费率。
- **PolynomialMul：** 两个长度分别为 `L,R` 的数组使用朴素双重循环，`O(L×R)`；2×2→16×16 为 700→7,400 ns，输入 ABI 解码与固定开销造成小规模时偏离理想平方曲线。

八算法的四档逐点耗时和费用、51 个原始样本、运行环境及模块哈希见 `2026-10-22-run-1.json` 至 `2026-10-22-run-3.json`；`2026-10-22-calibration.json` 包含聚合值。这里只给出参考档候选值，**不能**把四档样本简单扩展到任意输入：PBKDF2 的输出长度、DH 完整密钥位数、Pedersen 盲因子位数，以及任意较大数据都需要额外标定、限制或保守上界。尚未确定 `M(C)`、`α`、`N₀` 和激活高度，也未把候选值写入链共识配置。

## 复现

从仓库根目录运行：

```bash
bash experiments/cryptoupgrade/scripts/benchgascomplexity/build-fixtures.sh /tmp/cryptogas-fixtures
for run in 1 2 3; do
  go run ./experiments/cryptoupgrade/scripts/benchgascomplexity -wasm-dir /tmp/cryptogas-fixtures -warmup 10 -samples 51 -anchor-gas 100 -out "experiments/cryptoupgrade/results/gas-complexity/recheck-$run.json"
done
python3 experiments/cryptoupgrade/scripts/benchgascomplexity/aggregate.py experiments/cryptoupgrade/results/gas-complexity/recheck-{1,2,3}.json --out experiments/cryptoupgrade/results/gas-complexity/recheck-calibration.json
```

同机器复现应比较耗时比例和输出哈希，而非要求每次墙钟纳秒完全一致。不同硬件、WASM 版本、wazero/TinyGo 版本的结果不能直接替代本次链上固定标定。
