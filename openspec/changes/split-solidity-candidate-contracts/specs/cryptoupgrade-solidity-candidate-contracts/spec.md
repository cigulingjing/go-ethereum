## ADDED Requirements

### Requirement: Per-Algorithm Solidity Contracts
The system SHALL provide per-algorithm Solidity candidate contracts corresponding to practical Go plugin candidates under `cryptoupgrade/algorithm`.

#### Scenario: Go candidate mapping
- **WHEN** a Go candidate has a practical Solidity counterpart
- **THEN** the Solidity counterpart SHALL be implemented as an independently deployable contract under `cryptoupgrade/contracts`
- **AND** the contract name SHALL identify the algorithm it benchmarks

#### Scenario: Independent deployment unit
- **WHEN** a Solidity candidate is selected for deployment benchmarking
- **THEN** only the selected algorithm contract SHALL be deployed as the benchmark target
- **AND** unrelated candidate algorithm entry points SHALL NOT be included in that deployable contract

### Requirement: Solidity Compilation
Solidity candidate contracts SHALL compile with the repository's Solidity compiler settings.

#### Scenario: Compile candidate contracts
- **WHEN** Solidity candidates are validated
- **THEN** each deployable candidate contract SHALL compile with `solc-0.8.26 --optimize --via-ir`
- **AND** the compiled bytecode SHALL be non-empty

#### Scenario: Shared helper code
- **WHEN** Solidity candidates need shared arithmetic or encoding helpers
- **THEN** helper code SHALL be included through internal libraries or abstract base contracts
- **AND** helpers SHALL NOT force all algorithms into one deployable benchmark contract

### Requirement: Solidity Skip Reporting
The system SHALL report Go candidate algorithms that are not practical Solidity benchmark targets.

#### Scenario: Missing Solidity primitive
- **WHEN** a candidate requires secure randomness, AES, Ed25519, or large-field operations that are impractical without new dependencies
- **THEN** the candidate SHALL be listed as skipped for Solidity benchmarking
- **AND** the skip reason SHALL be documented near the Solidity candidate contracts or OpenSpec artifacts

#### Scenario: Benchmark exclusion
- **WHEN** an algorithm is skipped for Solidity benchmarking
- **THEN** benchmark reports SHALL NOT include a Solidity deployment or call measurement for that skipped algorithm as if it were implemented
