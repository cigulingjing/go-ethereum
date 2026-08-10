## 上下文

当前候选 Go 源码已经迁移到 `cryptoupgrade/algorithm/go`，Solidity 合约已经迁移到 `cryptoupgrade/algorithm/contracts`，但迁移后仍存在三个问题：

- 部分 Go 文件暴露多个导出函数，例如 `blake2b.go` 同时暴露 `Sum256`、`Sum384`、`Sum512`。
- 部分 Solidity 文件名、contract name 和函数名仍带有 `Solidity` 后缀，无法和 Go 文件形成稳定映射。
- `CandidateAlgorithmsSkipped.sol` 这类占位合约会让“合约侧是否实现了算法”统计产生歧义。

## 目标 / 非目标

**目标：**

- 统一算法源码目录：Go 在 `cryptoupgrade/algorithm/go`，没有 Solidity 对照的 Go-only 算法在 `cryptoupgrade/algorithm/go/archive`，Solidity 在 `cryptoupgrade/algorithm/contracts`。
- 统一命名规则：Go 文件使用现有 snake_case 名称，Solidity 文件使用对应 PascalCase 名称，例如 `add.go` 对应 `Add.sol`。
- 每个 Go 文件只保留一个导出的算法入口，每个 Solidity 算法合约只暴露一个外部算法入口。
- 删除难以实现或尚未实现的 Solidity 占位合约，并将对应 Go-only 文件移动到 `cryptoupgrade/algorithm/go/archive`。
- 更新 benchmark 默认路径，保证既有工具仍能按默认参数找到算法源码和合约源码。
- 增加静态一致性检查，输出 Go/Solidity 算法映射和操作语义是否一致。

**非目标：**

- 不改变 cryptoupgrade plugin 加载、ABI 编码、gas 计费或 native precompile 注册逻辑。
- 不为 AES、Ed25519、RandomBytes、Shamir 等困难算法补 Solidity 实现。
- 不在本 change 中追求所有 Go 算法都有 Solidity 对照。
- 不新增外部依赖。

## 设计决策

1. 使用单文件单导出入口。

   每个 Go 文件保留一个代表性导出函数，内部 helper 保持小写非导出。这样 `go build -buildmode=plugin` 和 benchmark 选择算法时不会面对同一文件多个候选入口。

2. 对多入口文件选择稳定代表操作。

   本次选择如下：

   | Go 文件 | 保留导出入口 | Solidity 对照 |
   | --- | --- | --- |
   | `add.go` | `Add` | `Add.sol` |
   | `archive/aes_cbc.go` | `AesCBCEncrypt` | 无 |
   | `blake2b.go` | `Sum256` | `Blake2b.sol` |
   | `dh2048.go` | `Dh2048Secret` | `Dh2048.sol` |
   | `archive/ed25519.go` | `Ed25519Verify` | 无 |
   | `pbkdf2_sha256.go` | `Pbkdf2Sha256` | `Pbkdf2Sha256.sol` |
   | `pedersen_commit.go` | `PedersenCommit` | `PedersenCommit.sol` |
   | `archive/random_bytes.go` | `RandomBytes` | 无 |
   | `schnorr_proof.go` | `SchnorrVerify` | `SchnorrProof.sol` |
   | `sha256.go` | `Sha256` | `Sha256.sol` |
   | `archive/shamir.go` | `ShamirRecover` | 无 |

   选择原则是：优先保留确定性、可 benchmark、与现有 Solidity 实现最接近的入口；涉及安全随机数或复杂链上 primitive 的入口不作为 Solidity 对照，并放入 `archive` 子目录避免被默认 Solidity 对照流程选中。

3. Solidity 文件名去掉 `Solidity` 后缀。

   文件名使用 PascalCase 算法名。外部函数名使用与 Go 导出入口相同的名称。由于 Solidity 0.8.x 不允许函数名与 contract name 完全相同，当算法入口名与文件主名相同时，contract name 使用稳定的 `Contract` 后缀，例如 `Sha256.sol` 中声明 `Sha256Contract` 并暴露 `Sha256`。内部 helper 或 abstract helper 合约不计入算法入口统计。

4. 保留 shared helper，但排除在算法统计之外。

   `CandidateAlgorithmsBigMod.sol` 仅为 DH、Pedersen、Schnorr 提供大整数 helper。它不是可部署 benchmark 算法目标，一致性检查应排除 abstract contract 和 internal/private helper。

5. 用静态脚本完成一致性检查。

   新增或使用一个仓库内脚本扫描 `cryptoupgrade/algorithm/go`、`cryptoupgrade/algorithm/go/archive` 和 `cryptoupgrade/algorithm/contracts`，验证每个 Go 文件最多一个导出入口、每个算法 Solidity 合约最多一个外部入口，并报告 Solidity 子集是否和 Go 操作同名、同语义。

## 风险 / 权衡

- [风险] 删除多余导出入口会影响手工 benchmark 命令。缓解方式：同步更新默认命令和文档路径，并在一致性报告中列出保留入口。
- [风险] Solidity 对照只覆盖 Go 子集，可能被误读为实现缺失。缓解方式：将 Go-only 文件放入 `archive`，并在报告中明确区分“匹配”和“仅 Go 保留”。
- [风险] 选择代表入口可能不符合后续某个实验。缓解方式：后续如需恢复某个入口，应新增独立文件和对应 change，而不是在同一文件重新暴露多个入口。

## 迁移计划

1. 清理 Go 文件导出函数，使每个文件只保留一个导出算法入口。
2. 重命名 Solidity 文件与 contract name，并只保留一个外部算法入口；当入口名与算法名相同时，contract name 使用稳定后缀避免 Solidity 语言冲突。
3. 删除占位 Solidity 合约，保留必要 abstract helper。
4. 更新 benchmark 默认路径和文档示例。
5. 运行 Go plugin 构建检查、Solidity 编译检查和一致性检查。
