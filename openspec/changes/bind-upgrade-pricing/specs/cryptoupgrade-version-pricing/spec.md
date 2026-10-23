## Purpose

Bind cryptographic invocation pricing to the same immutable algorithm-version proposal as its implementation, so each algorithm can publish and activate its own complete gas parameters without a shared chain-level override.

## ADDED Requirements

### Requirement: Publish independent version pricing
An upgrade proposal SHALL carry its input-size bands, call threshold and growth ratio atomically with the algorithm version. The system SHALL validate these parameters before accepting the proposal and expose them for query and node preparation.

#### Scenario: Different algorithms use different parameters
- **WHEN** two algorithms publish different thresholds, growth ratios and bands
- **THEN** each algorithm is charged using only its own selected version's parameters

#### Scenario: Invalid pricing
- **WHEN** a proposal contains malformed pricing, zero fees, invalid ratios or incomplete bands
- **THEN** it is rejected without replacing existing metadata

### Requirement: Activate and preserve pricing with the version
Pricing SHALL follow the selected version's activation height and survive metadata persistence and reload. Existing fixed-fee proposals SHALL retain fixed-fee behavior.

#### Scenario: Version activation and restart
- **WHEN** a new version activates and the node subsequently reloads persisted metadata
- **THEN** the new version's pricing remains available and querying the old version returns its original parameters

### Requirement: Enforce transaction-local accounting
The system SHALL count charged calls per algorithm and transaction, deduct the quoted gas before execution and reject insufficient gas or arithmetic overflow.

#### Scenario: Interleaved calls and reset
- **WHEN** calls to two algorithms are interleaved and another transaction begins
- **THEN** each algorithm's index progresses independently and both reset for the new transaction

### Requirement: Reject alternative pricing mutations
The system SHALL reject global pricing configurations, conflicting version submissions and in-place gas updates. Changing pricing SHALL require a new algorithm version.

#### Scenario: Attempt to bypass versioned pricing
- **WHEN** a caller resubmits an admitted version with different metadata or invokes the legacy gas update
- **THEN** the request fails and admitted pricing is unchanged

#### Scenario: Old global configuration
- **WHEN** a chain configuration contains the old global pricing profile
- **THEN** configuration validation reports that pricing must be published with an upgrade proposal
