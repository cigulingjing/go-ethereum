## Context

`cryptoupgrade/path.go` currently derives source and plugin paths from the package-level `compressedPath` value, and `cryptoupgrade/init.go` defines both `compressedPath = "./plugin"` and `algoInfoPath = "./plugin/algorithm_info.json"`. Runtime state therefore depends on the process working directory and path strings are split across multiple files.

The directory contains runtime artifacts rather than static source assets:

- `src/<Algorithm>.go`: decompressed uploaded algorithm source.
- `so/<Algorithm>.so`: compiled Go plugin.
- `algorithm_info.json`: persisted algorithm metadata.

These files are node-local execution artifacts. They should be deterministic for an operator, isolated for tests, and explicit enough to debug, while keeping the current default path for existing development workflows.

## Goals / Non-Goals

**Goals:**

- Centralize all cryptoupgrade plugin artifact path derivation in one small API.
- Keep `./plugin/src`, `./plugin/so`, and `./plugin/algorithm_info.json` as the default layout.
- Support an explicit base directory override for deployments and tests.
- Return and propagate directory creation errors.
- Add tests for path resolution, directory initialization, and compatibility behavior.

**Non-Goals:**

- Do not change `CodeStorage` ABI, algorithm upload encoding, plugin compile flags, gas accounting, or precompiled contract behavior.
- Do not migrate or rewrite existing plugin artifact files automatically.
- Do not add new dependencies.
- Do not add a geth CLI flag in this change; that can be layered on later if needed.

## Decisions

1. Use a centralized `pluginPaths` resolver.

   `path.go` should own a small struct or equivalent helpers for base directory, source directory, shared-object directory, and algorithm info path. Call sites should ask the resolver for paths instead of joining `compressedPath` and `algoInfoPath` separately.

   Alternative considered: keep the current package globals and only add comments. This does not fix split ownership or error handling, and it keeps future changes brittle.

2. Resolve the base directory once from a process-level override.

   The resolver should use a named environment variable when set, otherwise default to `./plugin`. Relative values should be resolved against the current working directory at initialization time so later working-directory changes do not silently move artifacts.

   Alternative considered: derive the directory from geth `--datadir`. That is attractive for node-local storage, but it requires wiring node or command configuration into `cryptoupgrade`, which is a broader cross-package change than necessary for this cleanup.

3. Preserve default compatibility but make custom locations explicit.

   Existing users that do nothing should continue using the same `plugin` directory under the startup working directory. Users that set an override get an isolated directory and are responsible for copying any existing `algorithm_info.json` or artifacts if they want to reuse them.

   Alternative considered: automatically copy from `./plugin` to the override path. That creates unclear ownership and can copy stale compiled plugins between incompatible binaries, so it is excluded.

4. Make initialization failures visible.

   `directoryInit` should return an error. `ActivateAlgorithm` and `Store` should propagate it. Package initialization should log failures with enough path context instead of ignoring them.

   Alternative considered: panic during package init when directories cannot be created. That is too disruptive for read-only code paths and makes node startup failure modes harder to control.

## Risks / Trade-offs

- Existing tests may assume `./plugin` relative paths -> keep the default layout and add isolated temporary-directory tests for the new override path.
- Environment variable configuration must be set before package initialization to affect initial metadata loading -> document the startup requirement and keep tests focused on resolver helpers.
- Operators may point multiple nodes at the same plugin directory -> document that plugin directories are node-local and should not be shared between concurrently running nodes.
- Relative override paths can still depend on startup directory -> resolve to absolute paths at initialization and expose resolved paths in tests/logging.
