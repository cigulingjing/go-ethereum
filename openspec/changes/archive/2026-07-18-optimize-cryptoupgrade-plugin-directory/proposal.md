## Why

The current cryptoupgrade runtime stores plugin sources, compiled `.so` files, and `algorithm_info.json` under a hard-coded relative `./plugin` directory. This makes behavior depend on the geth process working directory, which is fragile for multi-node local deployments, service managers, tests, and production nodes started from different paths.

The directory layout should remain compatible by default, but the runtime needs a clear path-management contract so operators can place plugin artifacts under a deterministic node-specific directory.

## What Changes

- Add a centralized cryptoupgrade plugin directory configuration layer that resolves the base directory once and derives `src`, `so`, and `algorithm_info.json` paths from it.
- Preserve the existing `./plugin` layout as the default when no explicit directory is configured.
- Allow the base plugin directory to be overridden through a process-level configuration mechanism suitable for geth startup and tests.
- Harden directory initialization by returning and handling errors instead of ignoring `os.MkdirAll` failures.
- Keep algorithm upload, compile, load, call, and metadata semantics unchanged.
- Document the default layout, override behavior, and migration expectations.

## Capabilities

### New Capabilities
- `cryptoupgrade-plugin-directory-management`: Defines how cryptoupgrade resolves, initializes, and uses its runtime plugin artifact directory.

### Modified Capabilities
- None.

## Impact

- Affected code: `cryptoupgrade/path.go`, `cryptoupgrade/init.go`, and call sites that assume global path constants or fire-and-forget directory initialization.
- Affected runtime behavior: default path remains `./plugin`, but deployments can opt into an explicit plugin base directory.
- Tests: add focused unit tests for default resolution, override resolution, path derivation, directory creation, and error propagation.
- Dependencies: no new external dependencies.
