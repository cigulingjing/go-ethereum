## Context

geth 的 native precompile 通过 `core/vm.PrecompiledContract` 暴露，`contracts.go` 按分叉规则维护地址到实现的映射，`RunPrecompiledContract` 统一完成 gas 扣减、tracing hook 和执行。当前仓库还存在 `core/vm/contracts_cryptoupgrade.go`，它把 `common.CodeStorageAddress` 上的 `CodeStorage` ABI 调用接入动态上传/插件执行路径。

`cryptoupgrade/algorithm` 中的候选算法文件是 `package main`，用于 `go build -buildmode=plugin`，不能直接被 `core/vm` import。部分函数还会读取系统随机数，例如 `RandomBytes`、`ShamirSplit`、`SchnorrProve`，以及空 seed 下的密钥生成函数；这类行为不能进入共识相关的 precompile 执行路径。

## Goals / Non-Goals

**Goals:**
- 将选定的 deterministic `cryptoupgrade` 算法作为 native precompile 注册到当前客户端。
- 保持现有 Ethereum precompile、`Blake2bSum256Address` 和 `CodeStorageAddress` 行为不变。
- 为每个新增 precompile 定义稳定地址、名称、ABI 输入类型、ABI 输出类型和 deterministic gas 规则。
- 让 precompile 可通过普通 EVM `CALL`、`CALLCODE`、`DELEGATECALL` 和 `STATICCALL` 进入现有 `RunPrecompiledContract` 路径。
- 添加直接 precompile 测试和 EVM call-path 测试，验证输出、gas 扣减和错误行为。

**Non-Goals:**
- 不把运行时上传的 Go plugin 动态转换成 precompile。
- 不在 precompile 中执行任何依赖系统随机数、文件系统、编译器或插件加载器的逻辑。
- 不新增第三方依赖。
- 不替换现有 `CodeStorage` 上传、查询和调用接口。

## Decisions

1. Use a reserved cryptoupgrade address range.

   新增算法 precompile 使用 `common.CodeStorageAddress`、`common.MutiVoucherAddress`、`common.CoinBaseAddress` 和 `common.Blake2bSum256Address` 之后的连续地址，例如从 `0x47` 开始。这样避免与 Ethereum 标准低地址 precompile、现有 cryptoupgrade 系统地址和未来内置地址发生冲突。

   Alternative considered: reuse `CodeStorageAddress` plus method selector dispatch. Rejected because the user-facing target is native precompile behavior, and precompile identity should be address based rather than ABI method based.

2. Keep VM integration as `PrecompiledContract` wrappers.

   `core/vm` should wrap cryptoupgrade algorithm entries with small implementations of `RequiredGas`, `Run`, and `Name`, then merge the resulting map into the active precompile set returned by the current chain rules. This preserves geth's normal gas charging and tracing behavior and keeps `isCryptoUpgradeCall` limited to the existing `CodeStorage` special case.

   Alternative considered: add another special branch in `EVM.Call` similar to `isCryptoUpgradeCall`. Rejected because normal precompiles already have the right dispatch path.

3. Add an importable native algorithm registry outside `cryptoupgrade/algorithm`.

   Because the plugin source files are `package main`, shared deterministic implementations should live in an importable package or root-level cryptoupgrade file, with registry records such as address, algorithm name, input ABI types, output ABI types, gas rule, and handler. Existing plugin candidates can remain in `cryptoupgrade/algorithm`; the implementation can either extract shared code for duplicated algorithms or add thin native wrappers for the selected deterministic functions.

   Alternative considered: compile and load the `cryptoupgrade/algorithm` files as plugins when a precompile runs. Rejected because precompile execution must be deterministic, fast, and independent of local compiler/plugin state.

4. Use direct ABI payloads for each precompile.

   Each algorithm address identifies exactly one entry point. The call input is ABI-encoded arguments only, without a method selector and without `callFunc(string,bytes)` wrapping. The output is ABI-encoded return values using the same `accounts/abi` type handling already used by cryptoupgrade serialization helpers.

   Alternative considered: require the `CodeStorage.callFunc` ABI shape for every precompile. Rejected because it would make the precompile interface depend on dynamic algorithm names even though the address already selects the algorithm.

5. Expose only deterministic, bounded entry points.

   Deterministic functions such as hashing, AES-CBC with caller-supplied IV, PBKDF2 with bounded iteration/key length, public-key derivation, signing with caller-supplied key material, verification, Pedersen commit/verify, Schnorr public/verify, DH public/secret, Shamir recover, and simple arithmetic are eligible. Functions that generate randomness are not registered unless the implementation rejects the random path and requires explicit deterministic inputs.

6. Make gas computation deterministic before execution.

   Each registry entry defines a `RequiredGas(input []byte) uint64` rule. Simple functions can use base plus per-word cost. Parameter-sensitive algorithms such as PBKDF2 and modular arithmetic must decode only the fields needed for gas or fall back to a conservative malformed-input cost. Overflow must saturate to `math.MaxUint64`, causing normal out-of-gas behavior before `Run`.

## Risks / Trade-offs

- Consensus nondeterminism from random or environment-dependent algorithms -> only deterministic entry points are registered, and random fallback inputs are rejected.
- Gas underpricing for CPU-heavy algorithms -> gas formulas are per algorithm and tests cover representative high-cost inputs, malformed inputs, and out-of-gas paths.
- Drift between plugin candidates and native precompile implementations -> deterministic logic should be extracted to shared code where practical, and fixture tests should compare precompile output to the selected reference implementation.
- Address collisions with future precompiles -> all cryptoupgrade algorithm addresses are declared centrally in `common` and tested for uniqueness against active precompile maps.
- VM dependency growth -> `core/vm` only imports a small cryptoupgrade registry/wrapper surface; algorithm-specific code stays outside VM files.

## Migration Plan

1. Add central address constants and a registry for deterministic cryptoupgrade precompile entries.
2. Add VM wrappers and merge the cryptoupgrade precompile map into active precompile sets.
3. Add unit tests for registry uniqueness, gas, ABI decode/encode, and output fixtures.
4. Add EVM call-path tests to ensure normal precompile dispatch reaches the new algorithms.

Rollback is removing the cryptoupgrade precompile map merge and address registrations; existing `CodeStorage` behavior is unaffected.

## Open Questions

- The exact initial algorithm subset should be finalized during implementation based on deterministic behavior and manageable gas formulas.
- If a benchmark requires a randomness-producing operation, it should use an explicit deterministic test seed or remain outside the native precompile set.
