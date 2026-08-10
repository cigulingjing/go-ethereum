
# 算法实现统计

Go 算法源码统一位于 `cryptoupgrade/algorithm/go`，其中没有 Solidity 对照的 Go-only
算法放在 `cryptoupgrade/algorithm/go/archive`。Solidity 合约统一位于
`cryptoupgrade/algorithm/contracts`。每个 Go 文件只保留一个导出算法入口，每个可部署
Solidity 合约只保留一个 external/public 算法入口。

| Go 文件 | Go 入口 | Solidity 文件 | Solidity 入口 | 状态 |
| --- | --- | --- | --- | --- |
| `add.go` | `Add` | `Add.sol` | `Add` | 匹配 |
| `archive/aes_cbc.go` | `AesCBCEncrypt` | - | - | 仅保留 Go 实现 |
| `blake2b.go` | `Sum256` | `Blake2b.sol` | `Sum256` | 匹配 |
| `dh2048.go` | `Dh2048Secret` | `Dh2048.sol` | `Dh2048Secret` | 匹配 |
| `archive/ed25519.go` | `Ed25519Verify` | - | - | 仅保留 Go 实现 |
| `pbkdf2_sha256.go` | `Pbkdf2Sha256` | `Pbkdf2Sha256.sol` | `Pbkdf2Sha256` | 匹配 |
| `pedersen_commit.go` | `PedersenCommit` | `PedersenCommit.sol` | `PedersenCommit` | 匹配 |
| `archive/random_bytes.go` | `RandomBytes` | - | - | 仅保留 Go 实现 |
| `schnorr_proof.go` | `SchnorrVerify` | `SchnorrProof.sol` | `SchnorrVerify` | 匹配 |
| `sha256.go` | `Sha256` | `Sha256.sol` | `Sha256` | 匹配 |
| `archive/shamir.go` | `ShamirRecover` | - | - | 仅保留 Go 实现 |

运行静态一致性检查：

```shell
bash cryptoupgrade/algorithm/check_interfaces.sh
```
