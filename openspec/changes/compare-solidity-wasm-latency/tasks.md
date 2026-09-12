## 1. Fixture and artifact compatibility

- [ ] 1.1 Add archive-path fallback for Solidity and WASM fixture resolution, and verify existing fixture tests still pass when the non-archive path is absent
- [ ] 1.2 Make generated local-network configuration fall back to the archived Add WASM artifact and verify `genlocalnetwork` can generate a smokeTest-disabled config
- [ ] 1.3 Define the supported comparison fixture metadata and validate that the selected algorithm is available in both existing benchmark commands

## 2. Lab 1 comparison runner

- [ ] 2.1 Add `experiments/cryptoupgrade/bench/cmd/benchlab1` flags, node-count parsing, output initialization, and verify defaults are `Sha256`, `1,2,5,10`, and a positive round count
- [ ] 2.2 Implement per-method/per-node-count isolated network generation, rendering, Compose lifecycle, and preflight invocation; verify commands and paths are recorded in a manifest
- [ ] 2.3 Invoke the Solidity and WASM latency benchmarks with identical fixture, timeout, polling, and target-node settings; verify each series preserves its child benchmark result path
- [ ] 2.4 Parse child JSON into normalized round and node observations, compute all-ready time, total latency, and slowest node, and verify incomplete rounds retain errors without being plotted as successful

## 3. Result artifacts

- [ ] 3.1 Emit `summary.csv` with `method`, `nodeCount`, `round`, `submitTime`, `allReadyTime`, `totalLatencyMs`, and `slowestNode`, and verify CSV quoting and failed-round behavior
- [ ] 3.2 Emit node-level CSV and comparison JSON with per-node receipt/ready timestamps, method-specific WASM activation fields, configuration metadata, and raw result paths
- [ ] 3.3 Add a concise Lab 1 usage document with the one-node smoke command, full matrix command, timing definitions, and plotting-oriented field descriptions

## 4. Verification

- [ ] 4.1 Add unit tests for node-count parsing, method selection, timestamp normalization, slowest-node selection, and CSV output, then verify the focused Go test packages pass
- [ ] 4.2 Run `gofmt` and the relevant `go test` commands, then run `openspec validate compare-solidity-wasm-latency`
- [ ] 4.3 Run the one-node two-method integration smoke when Docker, `solc`, and the lab image are available, and verify both rows plus node-level rows are present in the generated CSV
