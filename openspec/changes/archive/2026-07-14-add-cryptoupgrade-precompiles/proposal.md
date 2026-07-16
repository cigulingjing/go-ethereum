## Why

`cryptoupgrade/algorithm` 已经包含一组可用于升级和基准实验的密码算法实现，但这些实现目前还不能像 geth 内置预编译合约一样直接通过 EVM `CALL` 访问。将这些算法接入当前客户端的 precompile 路径，可以补齐性能基准中的“预编译合约实现”对照组，并提供与 Solidity/升级方案语义等价的 native 执行入口。

## What Changes

- Add native precompiled contract bindings for selected `cryptoupgrade/algorithm` implementations.
- Register the cryptoupgrade precompiles in the active geth precompile set without replacing existing Ethereum precompiles.
- Define deterministic input/output ABI handling and deterministic gas charging for each exposed algorithm entry point.
- Add tests that call the new precompile addresses through the EVM precompile path and compare outputs with the underlying algorithm behavior.
- Keep the existing plugin upload/call flow available; the new precompiles are an additional execution path.

## Capabilities

### New Capabilities

- `cryptoupgrade-precompiled-algorithms`: Covers exposing cryptoupgrade candidate algorithms as geth native precompiled contracts, including address registration, call semantics, gas accounting, and validation.

### Modified Capabilities

- None.

## Impact

- Affected code: `core/vm/contracts.go`, `core/vm/contracts_cryptoupgrade.go`, `core/vm/contracts_test.go`, and new cryptoupgrade precompile adapter code as needed.
- Affected algorithms: selected deterministic and benchmark-safe entry points under `cryptoupgrade/algorithm`.
- Affected planning artifacts: `openspec/changes/add-cryptoupgrade-precompiles`.
- No dependency changes.
