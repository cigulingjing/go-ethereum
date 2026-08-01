## 1. Path Resolver

- [x] 1.1 Add a centralized plugin path resolver in `cryptoupgrade/path.go` with base, source, shared-object, and metadata paths.
- [x] 1.2 Preserve the default `./plugin` layout while resolving the startup working-directory location deterministically.
- [x] 1.3 Add a process-level override for the plugin base directory and document the selected environment variable name in code.
- [x] 1.4 Update `gofilePath` and `sofilePath` to derive algorithm-specific paths from the resolver.

## 2. Runtime Integration

- [x] 2.1 Replace direct uses of `algoInfoPath` and `compressedPath` with resolver-backed helpers.
- [x] 2.2 Change directory initialization to return errors from `os.MkdirAll`.
- [x] 2.3 Propagate directory initialization errors from `ActivateAlgorithm` before source/plugin writes.
- [x] 2.4 Propagate directory initialization errors from `Store` before metadata writes.
- [x] 2.5 Keep package initialization tolerant of absent metadata while logging non-missing load errors with path context.

## 3. Tests and Documentation

- [x] 3.1 Add unit tests for default path derivation.
- [x] 3.2 Add unit tests for absolute and relative override path derivation.
- [x] 3.3 Add unit tests for directory creation and error propagation.
- [x] 3.4 Add or update cryptoupgrade documentation describing default layout, override usage, node-local isolation, and migration expectations.

## 4. Verification

- [x] 4.1 Run `gofmt` and `goimports` on modified Go files.
- [x] 4.2 Run targeted tests for the cryptoupgrade package.
- [x] 4.3 Run `go run ./build/ci.go test -short` if the implementation changes shared runtime behavior beyond the path helpers.
- [x] 4.4 Run `openspec validate optimize-cryptoupgrade-plugin-directory`.
