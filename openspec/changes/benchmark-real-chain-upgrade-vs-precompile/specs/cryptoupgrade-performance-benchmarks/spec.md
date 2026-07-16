## ADDED Requirements

### Requirement: Local Real-Chain Upgrade Vs Precompile Benchmark
系统 SHALL provide a benchmark workflow that compares the current upgrade scheme against the native precompiled contract scheme on a running local geth chain.

#### Scenario: Default local geth endpoint
- **WHEN** the benchmark command is run without an explicit RPC endpoint
- **THEN** it SHALL connect to `http://127.0.0.1:8666`

#### Scenario: Explicit local geth endpoint
- **WHEN** the benchmark command is run with an explicit RPC endpoint
- **THEN** it SHALL use that endpoint for all chain operations
- **AND** the result output SHALL record the endpoint used

#### Scenario: Two-way comparison only
- **WHEN** the real-chain benchmark runs
- **THEN** it SHALL compare `CodeStorage.callFunc` upgrade execution with native precompile execution
- **AND** it SHALL NOT require a Solidity contract comparison for this experiment

### Requirement: Real-Chain Setup Cost Measurement
系统 SHALL measure setup cost for the upgrade path and record the absence of deployment cost for the native precompile path.

#### Scenario: Upgrade setup measurement
- **WHEN** the benchmark uploads an algorithm through `CodeStorage.uploadCode`
- **THEN** it SHALL record transaction hash, elapsed time from `eth_sendTransaction` to receipt, receipt status, receipt gas used, source file path, source size, and compressed payload size

#### Scenario: Precompile setup measurement
- **WHEN** the benchmark selects a registered native precompile address
- **THEN** it SHALL record the precompile address
- **AND** it SHALL record setup transaction count and setup gas used as zero

#### Scenario: Setup failure reporting
- **WHEN** upgrade upload fails before or after transaction submission
- **THEN** the benchmark SHALL return a non-zero exit status
- **AND** it SHALL include enough error context to distinguish RPC failure, gas estimation failure, plugin compilation failure, and failed receipt status

### Requirement: Real-Chain Call Cost Measurement
系统 SHALL measure repeated read-only call cost for both benchmark schemes on the local geth chain.

#### Scenario: Warmed call latency statistics
- **WHEN** the benchmark invokes upgrade and precompile calls
- **THEN** it SHALL run a configurable number of warmup calls before recording samples
- **AND** it SHALL record first, mean, p50, p95, min, and max latency for each scheme

#### Scenario: Gas estimate statistics
- **WHEN** the benchmark prepares upgrade and precompile call data
- **THEN** it SHALL record `eth_estimateGas` results or equivalent call gas estimates for both schemes
- **AND** it SHALL include those values in the result output

#### Scenario: Ratio reporting
- **WHEN** both schemes complete successfully
- **THEN** the benchmark SHALL report upgrade/precompile latency ratios for mean, p50, and p95
- **AND** it SHALL report upgrade/precompile gas ratio when both gas values are non-zero

### Requirement: Real-Chain Result Equivalence
系统 SHALL verify that upgrade and precompile executions produce equivalent results for the configured algorithm and input.

#### Scenario: Matching outputs
- **WHEN** the upgrade call and precompile call return ABI-encoded outputs
- **THEN** the benchmark SHALL compare the decoded or canonical encoded outputs
- **AND** it SHALL mark the run as valid when the outputs match

#### Scenario: Mismatched outputs
- **WHEN** the upgrade call and precompile call return different outputs
- **THEN** the benchmark SHALL fail the run
- **AND** it SHALL include both outputs or output hashes in the error message

### Requirement: Reproducible Real-Chain Result Artifact
系统 SHALL produce reproducible benchmark output for the real-chain upgrade-vs-precompile experiment.

#### Scenario: Machine-readable result
- **WHEN** a benchmark run completes
- **THEN** it SHALL be able to write a JSON result containing endpoint, sender, algorithm name, source path, precompile address, ABI types, input, setup metrics, call metrics, gas metrics, ratios, and validation status

#### Scenario: Human-readable summary
- **WHEN** a benchmark run completes
- **THEN** it SHALL print a concise text summary showing setup cost, call latency, gas estimates, and ratios for both schemes

#### Scenario: Documented command
- **WHEN** benchmark code or result artifacts are updated
- **THEN** documentation SHALL include the exact command used against `http://127.0.0.1:8666`, iteration counts, warmup counts, input values, and expected output
