## 上下文

`cryptoupgrade/algorithm/go` 当前已经包含独立 Go plugin 候选，例如 `sha256.go`、`pbkdf2_sha256.go`、`dh2048.go`、`pedersen_commit.go` 和 `schnorr_proof.go`。Solidity 侧此前将多个候选聚合在单体候选合约中；这种方式便于编译，但不适合部署 benchmark，因为每次 Solidity 部署都会包含无关算法代码。

目标 benchmark 需要测量：

- 每个 Go plugin 候选的升级交易 gas 和耗时；
- 升级完成后的调用耗时和 gas；
- 匹配算法的 Solidity 合约部署 gas 和耗时；
- Solidity 合约部署后的调用耗时和 gas。

## 目标 / 非目标

**目标：**

- 将 Solidity 候选拆分为按算法独立部署的合约，并与 `cryptoupgrade/algorithm/go/*.go` 文件对应。
- 保持每个可部署合约都能被 benchmark tooling 独立选择。
- 只有在 helper 不是 benchmark 部署目标时，才将共享底层 Solidity helper 放在 library/base contract 中。
- 使用既有 `solc-0.8.26` 路径和 `--optimize --via-ir` 编译全部 Solidity 文件。
- 为没有实际可行 Solidity 对照的候选保留明确的跳过文档。

**非目标：**

- 不在本次 change 中改变 Go plugin 算法行为。
- 不新增 Solidity 依赖或外部 library。
- 不为了 benchmark 对齐而实现不安全的链上随机数，或手写大型 AES/Ed25519 实现。
- 不修改 `/home/liuqi/project/blockchain-crypto`。

## 设计决策

- 用每个实际可行算法一个可部署合约的方式替换单体 Solidity 候选合约：
  - `Sha256.sol` 对应 `sha256.go`
  - `Pbkdf2Sha256.sol` 对应 `pbkdf2_sha256.go`
  - `Dh2048.sol` 对应 `dh2048.go`
  - `PedersenCommit.sol` 对应 `pedersen_commit.go`
  - `SchnorrProof.sol` 对应 `schnorr_proof.go`
- 保留并重命名 `Blake2b.sol` 作为既有独立 benchmark 合约。
- 为 2048-bit 字节算术和 `modexp` 引入 helper library 或 abstract helper contract，让可部署合约保持聚焦，同时避免复制粘贴导致的漂移。
- 只有在有助于发现性时，才增加非 benchmark 的 skipped-candidate registry 或文档合约；被跳过算法不得作为等价算法 benchmark 被部署。
- 将部署 gas 视为合约专属指标：benchmark tooling 必须编译/选择精确的 contract name，并只部署该算法合约。

## 风险 / 权衡

- [风险] 共享 Solidity library 可能因需要链接而增加部署复杂度。缓解方式：优先使用 internal library 或 abstract base contract，让 solc 将代码内联到可部署合约中。
- [风险] 部分 Solidity 对照使用 `modexp` 等 EVM precompile，而 Go plugin 代码使用 native Go big integer。缓解方式：将其文档化为 Solidity 执行路径，并一致地作为实际可行 benchmark 的 Solidity 纯合约对照来比较。
- [风险] 现有 benchmark 代码可能假定只有一个 Solidity contract name。缓解方式：任务包含更新合约选择输入，使每个算法可以指定 source path、contract name、function 和 ABI input。
- [风险] 缺少 Solidity 对照的算法可能影响 benchmark 覆盖。缓解方式：明确保留跳过原因，并从 Solidity 部署/调用结果行中排除被跳过算法，避免记录误导性的零值。
