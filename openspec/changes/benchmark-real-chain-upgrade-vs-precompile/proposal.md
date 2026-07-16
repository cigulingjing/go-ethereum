## Why

当前已有升级方案和 native precompile 两条执行路径，但还缺少在真实 geth 链上可复现的时间与执行成本对比实验。需要把实验固定到本地 `http://127.0.0.1:8666` 链，直接量化升级方案与预编译合约方案在部署、调用延迟和 gas 上的差距。

## What Changes

- Add a real-chain benchmark workflow for comparing `CodeStorage.callFunc` upgrade execution with cryptoupgrade native precompile execution.
- Target the local geth execution RPC endpoint `http://127.0.0.1:8666` by default.
- Measure deployment/setup cost for the upgrade path and no-deploy/setup cost for the native precompile path.
- Measure `eth_call` latency with warmup and repeated samples, including first/mean/p50/p95/min/max.
- Measure execution gas through `eth_estimateGas` or an equivalent transaction simulation path for both schemes.
- Validate both schemes return equivalent ABI-encoded outputs for the same algorithm input.
- Emit machine-readable and human-readable benchmark results suitable for reproducible experiment documentation.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: Add a real-chain two-way benchmark requirement for upgrade-vs-precompile experiments on the local geth chain at port 8666.

## Impact

- Affected code: `cryptoupgrade/cmd/benchcandidate` or a new benchmark command under `cryptoupgrade/cmd/`.
- Affected docs/results: `cryptoupgrade/docs/` benchmark command examples and result files.
- Affected specs: `openspec/specs/cryptoupgrade-performance-benchmarks/spec.md`.
- No dependency changes.
