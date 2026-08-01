# cryptoupgrade-plugin-directory-management Specification

## Purpose
Define how cryptoupgrade resolves, initializes, and uses node-local runtime directories for uploaded algorithm source files, compiled Go plugins, and persisted algorithm metadata.

## Requirements
### Requirement: Centralized Plugin Artifact Paths
The cryptoupgrade runtime SHALL derive the algorithm source path, compiled plugin path, and algorithm metadata path from a single resolved plugin base directory.

#### Scenario: Default plugin paths are derived from one base directory
- **WHEN** no plugin directory override is configured
- **THEN** the source directory SHALL resolve under `plugin/src`, the compiled plugin directory SHALL resolve under `plugin/so`, and the metadata file SHALL resolve to `plugin/algorithm_info.json` relative to the process startup directory

#### Scenario: Algorithm-specific paths use the centralized resolver
- **WHEN** the runtime needs paths for algorithm `Sha256`
- **THEN** the source path SHALL be derived as `<base>/src/Sha256.go` and the compiled plugin path SHALL be derived as `<base>/so/Sha256.so`

### Requirement: Configurable Plugin Base Directory
The cryptoupgrade runtime SHALL support an explicit process-level plugin base directory override while preserving `./plugin` as the default.

#### Scenario: Override directory is configured
- **WHEN** the configured plugin base directory is `/tmp/geth-node-a/cryptoupgrade-plugin`
- **THEN** the runtime SHALL use `/tmp/geth-node-a/cryptoupgrade-plugin/src`, `/tmp/geth-node-a/cryptoupgrade-plugin/so`, and `/tmp/geth-node-a/cryptoupgrade-plugin/algorithm_info.json`

#### Scenario: Relative override directory is configured
- **WHEN** the configured plugin base directory is relative
- **THEN** the runtime SHALL resolve it against the process startup working directory before deriving child paths

### Requirement: Directory Initialization Error Handling
The cryptoupgrade runtime SHALL create required plugin artifact directories before writing source, compiled plugin, or metadata files, and SHALL return any directory creation failure to the caller.

#### Scenario: Directory creation succeeds
- **WHEN** an algorithm is activated or algorithm metadata is stored
- **THEN** the runtime SHALL ensure the source and compiled plugin directories exist before writing files

#### Scenario: Directory creation fails
- **WHEN** the source or compiled plugin directory cannot be created
- **THEN** algorithm activation or metadata storage SHALL fail with an error that identifies the failing directory operation

### Requirement: Backward-Compatible Metadata Loading
The cryptoupgrade runtime SHALL load `algorithm_info.json` from the resolved plugin base directory without changing the persisted metadata schema.

#### Scenario: Existing default metadata is present
- **WHEN** no plugin directory override is configured and `./plugin/algorithm_info.json` exists
- **THEN** package initialization SHALL load the existing metadata file using the current JSON schema

#### Scenario: Metadata file is absent
- **WHEN** the resolved metadata file does not exist
- **THEN** package initialization SHALL continue without treating the missing file as an error

### Requirement: Execution Semantics Remain Unchanged
The cryptoupgrade runtime SHALL keep algorithm upload, compilation, loading, invocation, ABI packing, and gas behavior unchanged while changing only path management.

#### Scenario: Algorithm activation uses the new path resolver
- **WHEN** an algorithm upload succeeds
- **THEN** the algorithm SHALL still be decompressed, compiled, registered, persisted, and callable through the existing `CodeStorage` behavior
