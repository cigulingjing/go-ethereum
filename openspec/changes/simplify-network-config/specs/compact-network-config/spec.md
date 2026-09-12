## Purpose
Provide one readable local private-network configuration that controls node count and keeps generated node state together, without requiring users to maintain repeated per-node YAML files.
## ADDED Requirements
### Requirement: Compact node configuration
The loader SHALL accept local.nodeCount and expand a signer followed by observers with unique sequential ports, without writing files during loading. It MUST reject nonpositive counts, port overflow and simultaneous local and nodes definitions.
#### Scenario: Twenty nodes
- **WHEN** local.nodeCount is 20
- **THEN** twenty nodes are available to deployment and benchmarks with one signer and nineteen observers.
#### Scenario: Invalid count
- **WHEN** local.nodeCount is zero or negative
- **THEN** loading fails with a configuration error.
### Requirement: Centralized deployment
The networks directory SHALL contain only local.yaml. Rendering SHALL place generated node resources in one output directory and produce a reloadable snapshot of the actual deployment.
#### Scenario: Render and reload
- **WHEN** a compact configuration is rendered to an output directory
- **THEN** its snapshot points to the rendered node directories and keys and reloads successfully.
### Requirement: Legacy compatibility
The loader SHALL continue accepting explicit nodes configurations.
#### Scenario: Historical configuration
- **WHEN** an existing expanded YAML is loaded
- **THEN** its nodes and network settings remain usable.
