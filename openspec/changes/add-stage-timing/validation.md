# Validation

Implemented and verified locally; no nodes or containers were started.

- `go test ./cryptoupgrade ./cryptoupgrade/internal/... ./cryptoupgrade/stagelog ./core/vm -count=1`: passed. Fixed the WASM fixture test helper to reference the existing `algorithm/wasm/archive` location.
- `go test ./rpc -parallel 1 -timeout 60s -count=1`: passed. Initial parallel full-package run stalled in TestServerWebsocketReadLimit and was stopped for diagnosis; isolated current and baseline-overlay versions of that test both passed.
- `go test ./internal/ethapi -skip '^TestEIP7910Config$' -count=1`: passed, including RPC-to-transaction-hash correlation.
- `go test ./core -run '^Test(StageTimingCanonicalInclusion|LogReorgs|ExtendCanonicalBlocks|ExtendCanonicalBlocksAfterMerge|CanonicalBlockRetrieval)$' -count=1`: passed.
- `go test -race ./cryptoupgrade/stagelog ./rpc -run 'Test(StageTiming|ConcurrentRecords|TimestampAndDisabled|OutputFailure)' -count=1`: passed.
- `go build -o /tmp/geth-stage-timing ./cmd/geth`: passed.
- `openspec validate add-stage-timing --strict`: passed.

The full ethapi suite has an unrelated existing mismatch in TestEIP7910Config: its expected precompile list omits PolynomialMul (0x5a), which was already present in the working tree before this change. That expectation was not changed as part of timing instrumentation.

Coverage includes successful real WASM output, failed execution, upgrade success/failure, future activationBlock metadata, concurrent line integrity, disabled/unwritable output, batch request isolation, signed RPC transaction linking and canonical inclusion (candidate execution alone does not emit inclusion).
