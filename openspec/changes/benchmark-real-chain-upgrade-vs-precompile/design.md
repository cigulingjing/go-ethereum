## Context

The repository already contains benchmark tooling under `cryptoupgrade/cmd/` and specs for performance benchmarking. Existing tools cover upgrade calls, Solidity comparisons, and at least one Blake2b precompile comparison, but there is no focused experiment that runs on a live local geth chain and compares only the current upgrade scheme against the newly added cryptoupgrade native precompile path.

The target test chain is the geth execution RPC endpoint at `http://127.0.0.1:8666`. The benchmark must use real RPC calls so it measures client-side EVM execution, transaction submission/receipt timing for upgrade setup, and `eth_call`/`eth_estimateGas` behavior as seen by a caller.

## Goals / Non-Goals

**Goals:**
- Provide a reproducible benchmark workflow for upgrade-vs-precompile experiments on the local geth chain at port 8666.
- Compare the same algorithm and same logical input across `CodeStorage.callFunc` and native precompile calls.
- Measure upgrade setup cost: upload transaction elapsed time, receipt gas used, uploaded source size, and compressed size.
- Measure call cost: warmup and repeated `eth_call` latency samples, first/mean/p50/p95/min/max, and gas estimates for both paths.
- Emit structured output that can be saved as a result artifact and checked into `cryptoupgrade/docs/`.
- Fail the benchmark when outputs are not equivalent.

**Non-Goals:**
- Do not benchmark Solidity in this change.
- Do not start or configure the geth node automatically.
- Do not add dependencies.
- Do not change native precompile semantics or gas formulas.
- Do not require the benchmark to submit state-changing algorithm calls; use `eth_call` for repeatable invocation timing.

## Decisions

1. Add or adapt a dedicated benchmark command.

   The benchmark should live under `cryptoupgrade/cmd/`, either by extending `benchcandidate` with a precompile mode or by adding a focused command such as `benchrealchain`. A dedicated command is preferable if extending `benchcandidate` would make the existing Solidity-oriented flags ambiguous.

2. Default RPC target to `http://127.0.0.1:8666`.

   The command should default to the user-specified local geth endpoint but keep `-rpc` configurable for repeated experiments against another endpoint.

3. Use address-selected precompile calls.

   Upgrade calls use `CodeStorage.callFunc(name, encodedInput)` sent to `common.CodeStorageAddress`. Precompile calls use the registered native precompile address directly, with ABI-encoded arguments only and no method selector.

4. Report setup and call phases separately.

   The upgrade path has a real setup phase because it uploads and activates code. The native precompile path has no deploy transaction, so its setup result should explicitly report zero deploy tx count and zero deploy gas, while still recording the precompile address used.

5. Emit both text and JSON.

   Text output is useful during experiments; JSON is needed for reproducible result docs. The JSON should include chain endpoint, sender, algorithm, input, output hash or raw output, setup metrics, call metrics, gas metrics, ratios, and validation status.

## Risks / Trade-offs

- Local node account is locked or missing -> require `eth_accounts[0]` or a `-from` address and surface a clear error.
- Upgrade upload can fail if geth cannot compile plugins -> include the same actionable error hint used by existing upload tooling.
- RPC noise can skew timing -> use warmup, repeated samples, and percentile reporting; record iteration counts with the result.
- Existing algorithm ABI may differ from precompile ABI -> require explicit input/output ABI type flags and fail on output mismatch.
- Precompile gas and upgrade gas are not charged through identical internal code paths -> report both gas estimates and call latency instead of collapsing them into one score.

## Migration Plan

1. Implement the benchmark command or extend an existing command with a clear upgrade-vs-precompile mode.
2. Add command examples and a result template under `cryptoupgrade/docs/`.
3. Validate against a running local geth node on `http://127.0.0.1:8666`.
4. Keep existing benchmark commands compatible.

## Open Questions

- Which algorithm should be the default fixture: `Sha256` is simplest for ABI bytes output; `Add` is useful for tiny-call overhead; both can be supported through flags.
- Whether the result artifact should store full raw outputs or output hashes for large byte results.
