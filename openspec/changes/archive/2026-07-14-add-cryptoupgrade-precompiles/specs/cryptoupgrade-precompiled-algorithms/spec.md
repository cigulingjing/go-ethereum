## ADDED Requirements

### Requirement: Cryptoupgrade Precompile Registration
The system SHALL register cryptoupgrade algorithm precompiles in the active geth precompile set without removing or replacing existing precompiled contracts.

#### Scenario: Active precompile map includes cryptoupgrade algorithms
- **WHEN** the EVM builds active precompiled contracts for the current chain rules
- **THEN** the returned precompile map SHALL include every registered cryptoupgrade algorithm address
- **AND** the returned precompile map SHALL still include the existing Ethereum and cryptoupgrade system precompiles

#### Scenario: Address list includes registered algorithms
- **WHEN** active precompile addresses are requested for the current chain rules
- **THEN** the returned address list SHALL include every registered cryptoupgrade algorithm address
- **AND** no registered cryptoupgrade algorithm address SHALL collide with another active precompile address

### Requirement: Deterministic Algorithm Selection
The system SHALL expose only deterministic and bounded `cryptoupgrade` algorithm entry points as native precompiles.

#### Scenario: Deterministic entry point is registered
- **WHEN** a `cryptoupgrade` algorithm entry point has deterministic output for ABI-equivalent input and has a deterministic gas rule
- **THEN** the implementation SHALL be eligible for registration as a cryptoupgrade precompile

#### Scenario: Random entry point is excluded
- **WHEN** a `cryptoupgrade` algorithm entry point depends on system randomness, file system state, local compiler state, or runtime plugin loading
- **THEN** the implementation SHALL NOT register that entry point as a native precompile

#### Scenario: Random fallback is rejected
- **WHEN** an otherwise deterministic algorithm function has an input form that would trigger random behavior
- **THEN** the precompile SHALL reject that input instead of executing the random path

### Requirement: ABI Call Semantics
The system SHALL use a stable ABI for each cryptoupgrade algorithm precompile.

#### Scenario: Successful precompile call
- **WHEN** a caller sends ABI-encoded arguments to a registered cryptoupgrade algorithm precompile address
- **THEN** the precompile SHALL decode the input using that algorithm's configured input ABI types
- **AND** SHALL execute the selected native algorithm entry point
- **AND** SHALL return ABI-encoded output using that algorithm's configured output ABI types

#### Scenario: Invalid ABI input
- **WHEN** a caller sends input that cannot be decoded using the precompile's configured input ABI types
- **THEN** the precompile SHALL return an execution error
- **AND** SHALL NOT call the algorithm handler with partially decoded arguments

#### Scenario: No method selector required
- **WHEN** a caller invokes a registered cryptoupgrade algorithm precompile
- **THEN** the input SHALL be interpreted as algorithm arguments for that address
- **AND** the caller SHALL NOT be required to include a function selector or `CodeStorage.callFunc` wrapper

### Requirement: Gas Accounting
The system SHALL charge deterministic gas for each cryptoupgrade algorithm precompile before executing the algorithm handler.

#### Scenario: Sufficient gas
- **WHEN** a caller invokes a registered cryptoupgrade algorithm precompile with at least the required gas
- **THEN** the EVM SHALL deduct the precompile required gas using the existing precompile gas path
- **AND** SHALL execute the algorithm handler
- **AND** SHALL return the remaining gas budget after the call

#### Scenario: Insufficient gas
- **WHEN** a caller invokes a registered cryptoupgrade algorithm precompile with less than the required gas
- **THEN** the EVM SHALL return out-of-gas through the existing precompile gas path
- **AND** SHALL NOT execute the algorithm handler

#### Scenario: Malformed gas-sensitive input
- **WHEN** a caller sends malformed input to an algorithm whose gas rule depends on decoded parameters
- **THEN** the required gas calculation SHALL remain deterministic
- **AND** the subsequent run SHALL return an input decoding error

### Requirement: Algorithm Result Equivalence
The system SHALL return results equivalent to the selected cryptoupgrade algorithm implementation for supported inputs.

#### Scenario: Fixture output matches
- **WHEN** a registered cryptoupgrade algorithm precompile is called with a supported test fixture input
- **THEN** the returned bytes SHALL match the ABI-encoded result produced by the corresponding native cryptoupgrade algorithm implementation

#### Scenario: Boolean output matches ABI encoding
- **WHEN** a registered verification algorithm returns a boolean result
- **THEN** the precompile SHALL return that boolean using standard ABI encoding

### Requirement: State Isolation
The system SHALL keep cryptoupgrade algorithm precompile execution independent from dynamic code upload state.

#### Scenario: Static call succeeds for pure algorithm
- **WHEN** a caller invokes a registered cryptoupgrade algorithm precompile through `STATICCALL`
- **THEN** the call SHALL execute with the same result as a normal call for the same input and gas

#### Scenario: CodeStorage state is unchanged
- **WHEN** a caller invokes a registered cryptoupgrade algorithm precompile
- **THEN** the call SHALL NOT upload code, modify algorithm metadata, write logs, or depend on previously uploaded `CodeStorage` plugin state
