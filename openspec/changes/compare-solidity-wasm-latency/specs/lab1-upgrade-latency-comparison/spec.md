## Purpose

为论文 Lab 1 提供可复现的 Solidity 合约部署与 WASM Coprocessor 动态升级端到端延迟对比，确保不同节点规模、重复轮次和逐节点 ready 时间能够由同一份机器可读结果直接分析。

## ADDED Requirements

### Requirement: Fixed comparison fixture

The experiment SHALL use the same algorithm fixture, ABI input values, expected output, sender account, consensus parameters, and readiness polling policy for the Solidity and WASM methods within a comparison run. The default fixture SHALL be a SHA-256 computation over the fixed input `hello cryptoupgrade`, while the fixture and input SHALL be configurable as a named supported option.

#### Scenario: Matching fixture metadata

- **WHEN** a comparison run starts for one node count
- **THEN** both method records SHALL contain the same algorithm identifier, input summary, expected output, chain ID, and target node IDs

### Requirement: Node-count matrix and repetitions

The experiment SHALL support the node-count matrix `1`, `2`, `5`, and `10`, and SHALL execute a positive number of repeated rounds for every selected node count and method. A failed round SHALL remain in the result with its failure reason instead of being silently omitted.

#### Scenario: Complete matrix

- **WHEN** the default comparison command is run with `rounds = R`
- **THEN** the summary result SHALL contain `2 * 4 * R` method/node-count/round records, subject to explicit failed records being retained

### Requirement: End-to-end readiness timing

For the Solidity method, `submitTime` SHALL be recorded immediately before the deployment transaction RPC is sent and `allReadyTime` SHALL be the control-side observation time when every target node can read the deployment receipt and successfully call the deployed contract with the fixed input. For the WASM method, `submitTime` SHALL be recorded immediately before the WASM upgrade transaction RPC is sent and `allReadyTime` SHALL be the control-side observation time when every target node has observed the upgrade receipt, completed the upgrade event handling and WASM loading path, and successfully called the upgraded algorithm with the fixed input. `totalLatencyMs` SHALL be the non-negative elapsed time between these two timestamps.

#### Scenario: Solidity all-node readiness

- **WHEN** a Solidity deployment receipt is visible on all target nodes and each target node returns the expected SHA-256 output
- **THEN** the round SHALL be marked ready and `totalLatencyMs` SHALL span transaction submission through the latest target-node contract-ready observation

#### Scenario: WASM all-node readiness

- **WHEN** each target node has a visible upgrade receipt and `callFunc` returns the expected SHA-256 output after local activation
- **THEN** the round SHALL be marked ready and `totalLatencyMs` SHALL span upgrade transaction submission through the latest target-node WASM-ready observation

#### Scenario: Partial readiness timeout

- **WHEN** at least one target node does not reach the method-specific ready condition before the round timeout
- **THEN** the round SHALL be marked incomplete, SHALL retain completed-node timestamps, SHALL identify the missing node error, and SHALL not report the incomplete round as a successful all-node latency

### Requirement: Slowest-node and phase observations

Each round SHALL record every target node's receipt observation time, method-specific ready time, completion status, and error. For a successful round, `slowestNode` SHALL identify the target node whose ready time equals `allReadyTime`; the result SHALL also retain the submission and all-ready wall-clock timestamps and the underlying method result path.

#### Scenario: Slowest node selection

- **WHEN** all target nodes complete at different times
- **THEN** `slowestNode` SHALL be the node with the maximum completed-at timestamp, not the node with the maximum RPC latency for an earlier phase

### Requirement: Comparison CSV output

The experiment SHALL emit a summary CSV with at least the columns `method`, `nodeCount`, `round`, `submitTime`, `allReadyTime`, `totalLatencyMs`, and `slowestNode`. It SHALL also emit a node-level CSV containing method, node count, round, node ID, receipt time, ready time, completed status, and error so later plots and audits do not require parsing logs.

#### Scenario: Plot-ready rows

- **WHEN** a successful comparison run completes
- **THEN** each method/node-count/round combination SHALL produce one summary CSV row with numeric `totalLatencyMs` and a non-empty `slowestNode`

#### Scenario: Reproducible raw artifacts

- **WHEN** a round succeeds or fails
- **THEN** the output directory SHALL retain the comparison JSON, summary CSV, node CSV, network metadata, and each method's underlying benchmark JSON path or embedded result

### Requirement: Isolated clean baselines

The experiment SHALL prevent a deployed contract, a previously loaded WASM module, a preflight crypto smoke test, or a prior method's state from satisfying a later round's readiness check. Each method/node-count series SHALL use a clean network state or an equivalent explicit pollution check and SHALL record the network/configuration identity used for that series.

#### Scenario: Method isolation

- **WHEN** the Solidity series and WASM series are executed for the same node count
- **THEN** each series SHALL start from an isolated clean network state and neither series SHALL reuse the other series' contract address, algorithm name, plugin directory, or datadir

#### Scenario: Preflight does not contaminate baseline

- **WHEN** network preflight is executed before a measured round
- **THEN** preflight SHALL validate RPC, peer connectivity, chain ID, and block growth without deploying the comparison contract or uploading the comparison WASM module
