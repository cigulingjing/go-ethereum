## Context

`cryptoupgrade/algorithm` now contains separate Go plugin candidates such as `sha256.go`, `pbkdf2_sha256.go`, `dh2048.go`, `pedersen_commit.go`, and `schnorr_proof.go`. The Solidity side currently groups several candidates in `CandidateAlgorithmsSolidity.sol`, which is convenient for compilation but wrong for deployment benchmarking because each Solidity deployment includes unrelated algorithm code.

The target benchmark measures:
- upgrade transaction gas and elapsed time for each Go plugin candidate,
- elapsed time and gas for calls after the upgrade completes,
- Solidity contract deployment gas and elapsed time for the matching algorithm,
- Solidity call elapsed time and gas after deployment.

## Goals / Non-Goals

**Goals:**
- Split Solidity candidates into per-algorithm deployable contracts that correspond to `cryptoupgrade/algorithm/*.go` files.
- Keep each deployable contract independently selectable by benchmark tooling.
- Keep shared low-level Solidity helpers in library/base contracts only where they are not deployed as the benchmark target.
- Compile all Solidity files with the existing `solc-0.8.26` path and `--optimize --via-ir`.
- Preserve explicit skip documentation for candidates without practical Solidity equivalents.

**Non-Goals:**
- Do not change Go plugin algorithm behavior in this change.
- Do not add Solidity dependencies or external libraries.
- Do not implement insecure on-chain randomness or hand-roll large AES/Ed25519 implementations for benchmark parity.
- Do not modify `/home/liuqi/project/blockchain-crypto`.

## Decisions

- Replace the monolithic Solidity candidate contract with one deployable contract per practical algorithm counterpart:
  - `Sha256Solidity` for `sha256.go`
  - `Pbkdf2Sha256Solidity` for `pbkdf2_sha256.go`
  - `Dh2048Solidity` for `dh2048.go`
  - `PedersenCommitSolidity` for `pedersen_commit.go`
  - `SchnorrProofSolidity` for `schnorr_proof.go`
- Keep `Blake2bSolidity.sol` as its existing standalone benchmark contract.
- Introduce helper libraries or abstract helper contracts for 2048-bit byte arithmetic and `modexp` so deployable contracts stay focused while avoiding copy-paste drift.
- Add a non-benchmark skipped-candidate registry or documentation contract only if useful for discoverability; skipped algorithms must not be deployed as if they were equivalent algorithm benchmarks.
- Treat deployment gas measurement as contract-specific: benchmark tooling must compile/select the exact contract name and deploy only that algorithm contract.

## Risks / Trade-offs

- [Risk] Shared Solidity libraries can complicate deployment if they require linking. -> Mitigation: prefer internal libraries or abstract base contracts whose code is inlined into the deployable contract by solc.
- [Risk] Some Solidity counterparts use EVM precompiles such as `modexp`, while Go plugin code uses native Go big integers. -> Mitigation: document this as the Solidity execution path and compare it consistently as the Solidity pure-contract counterpart for practical benchmarking.
- [Risk] Existing benchmark code may assume a single Solidity contract name. -> Mitigation: tasks include updating contract selection inputs so each algorithm can specify source path, contract name, function, and ABI input.
- [Risk] Algorithms without Solidity equivalents may skew benchmark coverage. -> Mitigation: keep skip reasons explicit and exclude skipped algorithms from Solidity deployment/call result rows rather than reporting misleading zero values.
