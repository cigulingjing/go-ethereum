## 1. Candidate Selection

- [x] 1.1 Map blockchain-crypto categories to practical single-file pure-Go candidates.
- [x] 1.2 Record skipped categories and reasons for categories that are not practical to convert.

## 2. Candidate Implementation

- [x] 2.1 Add standalone candidate files under `cryptoupgrade/algorithm`.
- [x] 2.2 Keep candidate entry points exported and avoid blockchain-crypto imports.

## 3. Validation

- [x] 3.1 Run `gofmt`/`goimports` on modified Go files.
- [x] 3.2 Build each new candidate with `go build -buildmode=plugin -tags=urfave_cli_no_docs,ckzg -trimpath`.
- [x] 3.3 Validate OpenSpec artifacts.

## 4. Solidity Counterparts

- [x] 4.1 Add Solidity counterparts for practical deterministic or verifiable candidate operations.
- [x] 4.2 Record skipped Solidity operations and reasons.
- [x] 4.3 Compile Solidity counterparts with solc.
