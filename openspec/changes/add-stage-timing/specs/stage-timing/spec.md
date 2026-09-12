## Purpose
Expose correlated node-local timestamps for RPC acceptance, WASM coprocessor execution, canonical transaction inclusion, and upgrade preparation so experiments can measure individual stages without changing execution semantics.
## ADDED Requirements
### Requirement: Independent structured log
The node SHALL append structured stage events to a separate configurable file, including nanosecond timestamps and node/process identity. Logging failures MUST NOT change execution results and logging MUST be disableable.
#### Scenario: Concurrent writes
- **WHEN** multiple executions emit events concurrently
- **THEN** complete JSON lines retain distinct correlation identifiers.
#### Scenario: Unwritable output
- **WHEN** the configured log cannot be written
- **THEN** business execution continues without a logging error being returned.
### Requirement: RPC and execution correlation
The node SHALL capture decoded RPC acceptance, attach unique request identifiers including batch members, and link submitted transaction hashes. WASM execution SHALL record entry and exit for each attempt including errors.
#### Scenario: Read-only call
- **WHEN** eth_call executes a WASM algorithm
- **THEN** RPC and execution events share a request identifier and no transaction inclusion is fabricated.
#### Scenario: Submitted transaction
- **WHEN** a signed transaction executes WASM
- **THEN** execution events carry its transaction hash and separate attempts have separate execution identifiers.
### Requirement: Canonical inclusion timestamp
The node SHALL record a local timestamp after a transaction-containing block becomes canonical, including transaction and block hashes and block number.
#### Scenario: Imported block
- **WHEN** a peer block becomes the local canonical head
- **THEN** inclusion events report local adoption time rather than the block header timestamp.
### Requirement: Upgrade boundaries
The node SHALL log actual WASM upgrade start and successful load completion with algorithm, version and originating transaction metadata when available. Failed upgrades MUST NOT emit successful completion.
#### Scenario: Activation failure
- **WHEN** compilation or loading fails
- **THEN** the log records failure and no successful load event.
#### Scenario: Scheduled activation
- **WHEN** a future version finishes loading
- **THEN** the log preserves activationBlock and identifies completion as local loading, not premature activation.
