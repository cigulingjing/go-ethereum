## Context

`cryptoupgrade` compiles uploaded algorithm source files as Go plugins. Candidate files therefore need to be standalone `package main` files and expose exported entry points that the plugin loader can resolve by name. The reference crypto implementations are in `/home/liuqi/project/blockchain-crypto`, but those files are out of scope for modification.

The blockchain-crypto repository contains both compact primitives and complex systems. Compact primitives such as hashes, AES, PBKDF2, Diffie-Hellman, Ed25519, Shamir sharing, Pedersen commitments, and Schnorr proofs can be represented by single-file pure-Go candidates. Systems such as VDF runners, VRFs, verifiable draw, shuffle, voting, BLS/KZG proof stacks, and oblivious transfer depend on external executables, CGO/Rust, generated protobufs, or many curve/protocol packages, so they are not good single-file plugin candidates for this pass.

## Goals / Non-Goals

**Goals:**
- Add standalone candidate source files under `cryptoupgrade/algorithm`.
- Use only Go standard library packages in the new candidates.
- Keep entry points exported and ABI-friendly where practical: `[]byte`, `bool`, and `*big.Int`.
- Verify each new source file with the same plugin build command used by cryptoupgrade.
- Record skipped blockchain-crypto categories and the reason they were not converted.

**Non-Goals:**
- Do not modify `/home/liuqi/project/blockchain-crypto`.
- Do not add dependencies.
- Do not port large multi-package implementations into cryptoupgrade.
- Do not claim simplified benchmark candidates are drop-in replacements for full protocol implementations.

## Decisions

- Implement candidates as independent files rather than importing blockchain-crypto packages. This keeps the plugin build self-contained and honors the no-source-modification constraint.
- Prefer standard-library implementations when they match the candidate category. Examples: SHA-256, AES-CBC, Ed25519, HMAC-SHA256 for PBKDF2, and `math/big` modular arithmetic for DH/Pedersen/Schnorr/Shamir.
- Use compact encoded byte formats for multi-value algorithms such as Shamir shares and Schnorr proofs. This keeps exported signatures simple for ABI encoding and plugin lookup.
- Treat complex protocol categories as skipped when a faithful single-file implementation would require copying large dependency trees or external binaries.

Candidate mapping:
- `rand`: `RandomBytes`
- `hash`: `Sha256`
- `encrypt`: `AesCBCEncrypt`, `AesCBCDecrypt`
- `signature`: `Ed25519Keygen`, `Ed25519PublicKey`, `Ed25519Sign`, `Ed25519Verify`
- `key_exchange`: `Dh2048Private`, `Dh2048Public`, `Dh2048Secret`
- `kdf`: `Pbkdf2Sha256`
- `share`: `ShamirSplit`, `ShamirRecover`
- `commit`: `PedersenCommit`, `PedersenVerify`
- `proof`: `SchnorrPublicKey`, `SchnorrProve`, `SchnorrVerify`

Solidity counterpart mapping:
- Implemented: `Sha256`, `Pbkdf2Sha256`, `Dh2048Public`, `Dh2048Secret`, `PedersenCommit`, `PedersenVerify`, `SchnorrPublicKey`, `SchnorrVerify`
- Skipped: `RandomBytes` and private/proof generation routines that require secure randomness; AES-CBC and Ed25519 routines that lack native Solidity/precompile support; Shamir split/recover because the Go candidate uses a 521-bit field and compact Solidity support would require a much larger arbitrary-precision field library.

Skipped categories:
- `vdf`: current implementations call external VDF executables or Rust-backed runners.
- `vrf`: implementations depend on multi-file Edwards25519/secp256k1 VRF stacks.
- `protocols`: oblivious transfer and transmission evidence depend on Ristretto/BLS protocol packages and multi-round state.
- `shuffle`, `verifiable_draw`, and `vote`: implementations depend on BLS/KZG/Caulk+/DLEQ/Schnorr proof trees, SRS objects, protobufs, or preloaded vote flows.
- `types`, `utils`, and `change`: support packages rather than standalone cryptographic algorithm categories for plugin candidates.

## Risks / Trade-offs

- Simplified candidates are benchmark representatives, not complete replacements for the original blockchain-crypto protocols. Mitigation: record skipped categories and keep file names/category mapping explicit.
- Some candidates use benchmark-friendly byte encodings rather than typed Go structs. Mitigation: exported functions return deterministic byte formats documented in code comments.
- Single-file candidates duplicate small constants such as the RFC 3526 group prime. Mitigation: duplication keeps each file independently buildable through `go build ... srcpath`.
- Solidity counterparts for 2048-bit modular operations rely on the EVM `modexp` precompile plus compact byte-array arithmetic. Mitigation: keep these contracts as benchmark counterparts and compile-check them with solc.
