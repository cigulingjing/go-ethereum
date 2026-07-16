## 1. Benchmark Command Structure

- [x] 1.1 Decide whether to extend `cryptoupgrade/cmd/benchcandidate` or add a focused `cryptoupgrade/cmd/benchrealchain` command.
- [x] 1.2 Add default RPC endpoint `http://127.0.0.1:8666` with an overrideable `-rpc` flag.
- [x] 1.3 Add flags for algorithm name, source file, input ABI types, output ABI types, ABI-encoded input, expected output, native precompile address, sender, warmup count, sample count, and output JSON path.
- [x] 1.4 Add validation for supported modes and required parameters without changing existing benchmark commands.

## 2. Upgrade and Precompile Execution Paths

- [x] 2.1 Implement upgrade setup through `CodeStorage.uploadCode` with transaction hash, receipt status, elapsed time, gas used, source size, and compressed size metrics.
- [x] 2.2 Implement upgrade call data generation through `CodeStorage.callFunc(name, encodedInput)`.
- [x] 2.3 Implement native precompile call data generation using ABI-encoded arguments sent directly to the configured precompile address.
- [x] 2.4 Record native precompile setup metrics as zero deployment transactions and zero deployment gas while preserving the selected address.

## 3. Measurement and Validation

- [x] 3.1 Implement warmup and repeated `eth_call` timing for both schemes.
- [x] 3.2 Compute first, mean, p50, p95, min, and max latency statistics for each scheme.
- [x] 3.3 Estimate call gas for both schemes with `eth_estimateGas` or an equivalent simulation path.
- [x] 3.4 Compare upgrade and precompile outputs using decoded or canonical ABI-encoded output and fail on mismatch.
- [x] 3.5 Compute upgrade/precompile ratios for latency and gas where values are non-zero.

## 4. Result Output and Documentation

- [x] 4.1 Print a concise human-readable summary for setup cost, call latency, gas estimates, and ratios.
- [x] 4.2 Write a machine-readable JSON result with endpoint, sender, algorithm, source path, ABI types, input, precompile address, setup metrics, call metrics, gas metrics, ratios, and validation status.
- [x] 4.3 Add documentation under `cryptoupgrade/docs/` with the exact command for the local `8666` geth chain, iteration counts, input values, and expected output.
- [x] 4.4 Add or update a sample result artifact for at least one algorithm fixture if a local chain is available.

## 5. Verification

- [ ] 5.1 Run `gofmt` and `goimports` on modified Go files.
- [ ] 5.2 Run targeted tests or compile checks for the benchmark command package.
- [ ] 5.3 Run the benchmark against `http://127.0.0.1:8666` when the local geth chain is available and record the command/result.
- [ ] 5.4 Run `openspec validate benchmark-real-chain-upgrade-vs-precompile`.
