# TinyGo WASM fixtures

This directory holds TinyGo-built guest modules for `cryptoupgrade`.

Build recipe:

```bash
TINYGO=${TINYGO:-tinygo} ./build.sh
```

Guest contract:

- `execute(input_ptr, input_len) -> output_ptr`
- `input_ptr` and `output_ptr` are linear-memory offsets
- returned data is length-prefixed little-endian bytes
- `Add` returns ABI-encoded `int256`
- `Sha256` returns ABI-encoded `bytes`
- `Sum256` returns ABI-encoded `bytes32`
- `Pbkdf2Sha256`, `Dh2048Secret` and `PedersenCommit` return ABI-encoded `bytes`
- `SchnorrVerify` returns ABI-encoded `bool`

The modules are built with TinyGo `wasi` target in reactor mode through
`cryptoupgrade/wasmtool/cmd/wasmbuild`, which adds the `execute` ABI wrapper:

```bash
go run ./cryptoupgrade/wasmtool/cmd/wasmbuild \
  -source cryptoupgrade/algorithm/go/blake2b.go \
  -function Sum256 \
  -itype bytes \
  -otype bytes32 \
  -out cryptoupgrade/algorithm/wasm/blake2b.wasm
```

`build.sh` records the full command list for:

- `add.wasm`
- `sha256.wasm`
- `blake2b.wasm`
- `pbkdf2_sha256.wasm`
- `dh2048.wasm`
- `pedersen_commit.wasm`
- `schnorr_proof.wasm`
