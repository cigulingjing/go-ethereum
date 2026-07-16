## ADDED Requirements

### Requirement: Standalone Candidate Plugin Sources
The system SHALL provide standalone cryptographic candidate source files under `cryptoupgrade/algorithm` for blockchain-crypto categories that can be represented as pure-Go single-file plugins.

#### Scenario: Candidate source layout
- **WHEN** a candidate algorithm is added
- **THEN** its source file SHALL use `package main`
- **AND** it SHALL expose at least one exported algorithm entry point
- **AND** it SHALL NOT import packages from `/home/liuqi/project/blockchain-crypto`

#### Scenario: Practical category coverage
- **WHEN** a blockchain-crypto category has a practical pure-Go single-file representative
- **THEN** the system SHALL include one candidate source file for that category in `cryptoupgrade/algorithm`

### Requirement: Plugin Build Validation
Candidate algorithm source files SHALL be validated with the cryptoupgrade plugin build command.

#### Scenario: Successful candidate build
- **WHEN** a candidate source file is validated
- **THEN** `go build -buildmode=plugin -tags=urfave_cli_no_docs,ckzg -trimpath -o <outputPath> <srcpath>` SHALL complete successfully

#### Scenario: Build output handling
- **WHEN** plugin validation creates build artifacts
- **THEN** those artifacts SHALL be written outside committed source paths or otherwise excluded from commits

### Requirement: Skipped Category Reporting
The system SHALL report blockchain-crypto categories that are not converted because a pure-Go single-file plugin adaptation is impractical.

#### Scenario: Complex category skipped
- **WHEN** a category depends on external executables, CGO, Rust, generated protocol types, or large multi-package curve/proof stacks
- **THEN** the category SHALL be reported as skipped with a concise reason

#### Scenario: Source repository isolation
- **WHEN** candidates are produced or categories are skipped
- **THEN** files under `/home/liuqi/project/blockchain-crypto` SHALL remain unmodified

### Requirement: Solidity Candidate Counterparts
The system SHALL provide Solidity counterparts for candidate operations that can be implemented as deterministic or verifiable contract logic without adding dependencies.

#### Scenario: Solidity counterpart layout
- **WHEN** a Solidity counterpart is added
- **THEN** its source file SHALL be placed under `cryptoupgrade/contracts`
- **AND** it SHALL compile with Solidity 0.8.26

#### Scenario: Impractical Solidity counterpart
- **WHEN** a Go candidate depends on secure randomness, AES/Ed25519 primitives, or field arithmetic that is not practical in a compact Solidity contract
- **THEN** the Solidity counterpart SHALL report the skipped operation with a concise reason
