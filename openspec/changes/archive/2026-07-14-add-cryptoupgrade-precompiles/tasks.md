## 1. Registry and Addressing

- [x] 1.1 Audit `cryptoupgrade/algorithm` exported entry points and choose the deterministic, bounded subset to expose as precompiles.
- [x] 1.2 Add central cryptoupgrade algorithm precompile address constants after the existing `0x43` to `0x46` system addresses.
- [x] 1.3 Add an importable native algorithm registry with address, name, input ABI types, output ABI types, gas rule, and handler metadata.
- [x] 1.4 Ensure the registry rejects or omits entry points that depend on randomness, runtime plugin loading, compiler state, or file system state.

## 2. Native Algorithm Adapters

- [x] 2.1 Add native implementations or shared wrappers for the selected deterministic algorithms without importing `cryptoupgrade/algorithm` package-main files.
- [x] 2.2 Implement ABI encode/decode helpers for direct precompile inputs and outputs.
- [x] 2.3 Implement deterministic gas formulas for each registered algorithm, including overflow-safe saturation.
- [x] 2.4 Add input validation that rejects random fallback forms such as empty deterministic seeds where applicable.

## 3. VM Precompile Integration

- [x] 3.1 Add a `core/vm` wrapper that implements `PrecompiledContract` for cryptoupgrade registry entries.
- [x] 3.2 Merge cryptoupgrade algorithm precompiles into `activePrecompiledContracts` without mutating existing fork maps.
- [x] 3.3 Include cryptoupgrade algorithm addresses in `ActivePrecompiles` results for the current chain rules.
- [x] 3.4 Keep the existing `CodeStorageAddress` special-case path unchanged for upload/query/plugin calls.

## 4. Tests

- [x] 4.1 Add registry tests for address uniqueness, name uniqueness, deterministic subset selection, and ABI metadata validity.
- [x] 4.2 Add direct `RunPrecompiledContract` fixture tests for successful outputs, boolean ABI outputs, invalid ABI input, and out-of-gas behavior.
- [x] 4.3 Add EVM call-path tests showing cryptoupgrade precompiles execute through normal precompile dispatch, including `STATICCALL`.
- [x] 4.4 Add regression tests proving existing Ethereum precompiles, `Blake2bSum256Address`, and `CodeStorageAddress` behavior remain available.

## 5. Verification

- [x] 5.1 Run `gofmt` and `goimports` on modified Go files.
- [x] 5.2 Run targeted package tests for `cryptoupgrade` and `core/vm`.
- [x] 5.3 Run `go run ./build/ci.go test -short`.
- [x] 5.4 If preparing a commit, run the full AGENTS pre-commit checklist.
