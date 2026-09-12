# Single-node time comparison, 20 rounds

Source JSON: `result.json`
RPC: `http://127.0.0.1:19666`
Timestamp: `2026-09-09T09:31:58.41976019Z`
Warmup calls per implementation: `10`
Measured calls per implementation: `20`

Files:
- `single_node_time_rounds_wide.csv`: one row per algorithm and measured round; columns compare EvoCrypt, Solidity, and Precompile latency in milliseconds.
- `single_node_time_rounds_long.csv`: one row per algorithm, scheme, and measured round.
- `single_node_time_summary.csv`: aggregate latency and gas estimates from the same run.
- `skipped_algorithms.csv`: algorithms skipped because all three comparable implementations are not available.

Scheme mapping: `upgrade` = EvoCrypt, `contract` = Solidity, `precompile` = Precompile.

Comparable algorithms in this run:
- `Add`
- `Sha256`
- `Blake2bSum256`
- `Pbkdf2Sha256`
- `Dh2048Secret`
- `PedersenCommit`
- `SchnorrVerify`

Skipped algorithms:
- `Blake2bSum384`: missing comparable Go and contract implementation
- `Blake2bSum512`: missing comparable Go and contract implementation
- `AesCBCEncrypt`: Go implementation is archived and no comparable contract implementation exists
- `AesCBCDecrypt`: Go implementation is archived and no comparable contract implementation exists
- `Dh2048Private`: key-generation helper without comparable contract algorithm
- `Dh2048Public`: key-generation helper without comparable contract algorithm
- `Ed25519Keygen`: Go implementation is archived and no comparable contract implementation exists
- `Ed25519PublicKey`: Go implementation is archived and no comparable contract implementation exists
- `Ed25519Sign`: Go implementation is archived and no comparable contract implementation exists
- `Ed25519Verify`: Go implementation is archived and no comparable contract implementation exists
- `PedersenVerify`: missing comparable active Go and contract implementation
- `SchnorrPublicKey`: key-generation helper without comparable contract algorithm
