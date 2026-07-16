## 1. Solidity Contract Split

- [x] 1.1 Map each practical `cryptoupgrade/algorithm/*.go` candidate to a deployable Solidity contract name.
- [x] 1.2 Split `CandidateAlgorithmsSolidity.sol` into per-algorithm deployable contracts.
- [x] 1.3 Move shared 2048-bit arithmetic and `modexp` helpers into non-deploy-target helper code.
- [x] 1.4 Preserve or add skip documentation for candidates without practical Solidity equivalents.

## 2. Benchmark Integration

- [x] 2.1 Update Solidity compile selection to accept or resolve the target contract name.
- [x] 2.2 Update benchmark inputs so each algorithm can specify Solidity source, contract, function, and ABI arguments.
- [x] 2.3 Ensure deployment metrics are captured per selected Solidity algorithm contract.
- [x] 2.4 Ensure post-deployment call elapsed time and call gas are captured per selected Solidity algorithm function.

## 3. Validation

- [x] 3.1 Compile all Solidity candidate contracts with `solc-0.8.26 --optimize --via-ir`.
- [x] 3.2 Run focused Go tests or build checks for modified benchmark tooling.
- [x] 3.3 Validate OpenSpec artifacts.
