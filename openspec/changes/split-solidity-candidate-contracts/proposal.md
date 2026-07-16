## Why

The benchmark flow needs per-algorithm Solidity contracts so deployment gas, deployment elapsed time, post-deployment call time, and call gas can be measured independently against the matching `cryptoupgrade/algorithm/*.go` plugin candidates. The current combined Solidity candidate contract makes per-algorithm deployment measurements noisy because unrelated code is deployed together.

## What Changes

- Split `cryptoupgrade/contracts/CandidateAlgorithmsSolidity.sol` into multiple Solidity contracts aligned with the candidate Go files under `cryptoupgrade/algorithm`.
- Keep each Solidity contract independently compilable and deployable for benchmark runs.
- Preserve a shared helper/library boundary only when it avoids duplicated low-level code without forcing all algorithms into one deployable contract.
- Record algorithms that cannot be represented as practical Solidity contracts, such as secure randomness, AES-CBC, Ed25519, and Shamir over the 521-bit field.
- Prepare benchmark tooling expectations for measuring each Solidity contract deployment and each post-deployment call separately.

## Capabilities

### New Capabilities
- `cryptoupgrade-solidity-candidate-contracts`: Covers per-algorithm Solidity contract sources corresponding to `cryptoupgrade/algorithm/*.go` candidates.

### Modified Capabilities
- `cryptoupgrade-performance-benchmarks`: Benchmark requirements shall support per-algorithm Solidity deployment and post-deployment call measurements.

## Impact

- Affected code: `cryptoupgrade/contracts/*.sol`
- Affected benchmark tooling expectations: Solidity contract selection, deployment, and call measurement per algorithm.
- Affected planning artifacts: `openspec/changes/split-solidity-candidate-contracts`
- No dependency changes.
- No changes to `/home/liuqi/project/blockchain-crypto` source files.
