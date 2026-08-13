## 1. Runtime Semantics

- [x] 1.1 Inspect `CodeStorage.uploadCode` data flow and confirm where algorithm metadata is stored for `getInfo`.
- [x] 1.2 Remove the direct `ActivateAlgorithm` call from `RunCodeStorageCall` `uploadCode` branch.
- [x] 1.3 Ensure `uploadCode` still records algorithm metadata that `CodeStorage.getInfo(name)` can return.
- [x] 1.4 Ensure `uploadCode` still emits `codeUploaded` only after metadata is available.
- [x] 1.5 Keep `uploadCode` rejected in static context without metadata writes or event emission.

## 2. Event Activation

- [x] 2.1 Audit `BindCodeUploaded` event parsing, `lookupCodeInfo`, duplicate detection and activation error handling against the new spec.
- [x] 2.2 Fix listener robustness issues that would prevent continuous event processing, including nil subscription channels or decode errors.
- [x] 2.3 Confirm geth starts `BindCodeUploaded` in a dedicated goroutine and does not rely on `uploadCode` for local activation.
- [x] 2.4 Preserve `CodeStorage.callFunc` behavior so it only succeeds when the local node has activated the algorithm.

## 3. Benchmarks and Documentation

- [x] 3.1 Update benchmark logic that assumes upload receipt equals activation completion to wait for `callFunc` success.
- [x] 3.2 Update benchmark result notes and docs that describe synchronous `uploadCode` activation.
- [x] 3.3 Rename or clarify timing fields where needed so submit-to-receipt and receipt-to-activation are separate.

## 4. Verification

- [x] 4.1 Add or update unit tests proving `uploadCode` does not call `ActivateAlgorithm` directly.
- [x] 4.2 Add or update tests for event-driven activation using a fake client or focused listener helper.
- [x] 4.3 Run focused `cryptoupgrade` tests.
- [x] 4.4 Run OpenSpec validation for `bind-event`.
