## 1. Experiment Command

- [x] 1.1 Add `benchsolidityupgradelatency` command with multi-node config, sender/target selection, result path initialization, and verify `go test ./experiments/cryptoupgrade/bench/cmd/benchsolidityupgradelatency` passes.
- [x] 1.2 Add Solidity algorithm fixtures for Add, Sha256, Blake2bSum256, Pbkdf2Sha256, Dh2048Secret, PedersenCommit, and SchnorrVerify, and verify fixture selection tests pass.
- [x] 1.3 Implement solc compilation and deployment transaction submission, and verify build/test covers ABI and bytecode metadata handling.

## 2. Observation And Outputs

- [x] 2.1 Implement per-node concurrent receipt polling and contract validation calls, and verify unit tests cover summary timing aggregation.
- [x] 2.2 Write `result.json`, `rounds.csv`, and `nodes.csv` with Lab1-compatible field names plus Solidity contract metadata, and verify CSV tests cover required columns.
- [x] 2.3 Run `gofmt`, `go test ./experiments/cryptoupgrade/bench/cmd/benchsolidityupgradelatency`, and `openspec validate benchmark-solidity-upgrade`.

## 3. 20-Node Data Collection

- [x] 3.1 Run the Solidity deployment/upgrade benchmark against the 20-node network with `-algorithms all -rounds 1`, and verify every algorithm has 20 completed nodes.
- [x] 3.2 Save the raw result directory under `experiments/cryptoupgrade/results/` and report Solidity per-algorithm upgrade time and Gas so it can be merged with the existing Lab1 WASM data.
