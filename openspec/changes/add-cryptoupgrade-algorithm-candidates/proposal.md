## Why

The cryptoupgrade flow needs representative cryptographic algorithms that can be compiled as Go plugins for upgrade and benchmark experiments. The source algorithms live in `/home/liuqi/project/blockchain-crypto`, but those sources must remain untouched, so candidates need standalone plugin-friendly Go files in `cryptoupgrade/algorithm`.

## What Changes

- Add one standalone candidate plugin source for each blockchain-crypto category where a pure-Go, single-file adaptation is practical.
- Keep every candidate file as `package main` with exported algorithm entry points.
- Validate each candidate with `go build -buildmode=plugin -tags=urfave_cli_no_docs,ckzg -trimpath`.
- Add Solidity contract counterparts for deterministic or verifiable candidate operations that can be implemented without new dependencies.
- Document categories skipped because their implementations depend on external executables, Rust/C bindings, large curve/proof stacks, or multi-package protocol state that is not reasonable to collapse into one file.

## Capabilities

### New Capabilities
- `cryptoupgrade-algorithm-candidates`: Covers standalone pure-Go candidate algorithm sources and build validation for cryptoupgrade plugins.

### Modified Capabilities
- `cryptoupgrade-performance-benchmarks`: Extends benchmark preparation requirements with a reusable candidate algorithm set compiled through the existing Go plugin path.

## Impact

- Affected code: `cryptoupgrade/algorithm/*.go`, `cryptoupgrade/contracts/*.sol`
- Affected planning artifacts: `openspec/changes/add-cryptoupgrade-algorithm-candidates`
- No dependency changes.
- No changes to `/home/liuqi/project/blockchain-crypto` source files.
