# WASM experiment rerun summary

## Lab1 upgrade latency
- Completed rounds: 7/7 across 5 target nodes.
- Mean submit-to-all-complete latency: 5022.364 ms.
- Range: 4857.450-5342.201 ms.
- Largest post-event readiness tail: 287.298 ms.

## Lab2 execution efficiency
- Comparable algorithms: 7; each implementation uses 10 warmup and 100 measured eth_call samples.
- Output matching: all comparable algorithms matched across upgrade, contract, and precompile.

| Algorithm | WASM mean ms | Contract mean ms | Precompile mean ms | WASM gas | Contract gas | Precompile gas |
|---|---:|---:|---:|---:|---:|---:|
| Add | 1.081 | 0.880 | 0.885 | 25408 | 21736 | 24739 |
| SHA-256 | 1.002 | 0.868 | 0.979 | 25801 | 22846 | 25156 |
| BLAKE2b | 1.046 | 1.418 | 0.868 | 25801 | 113059 | 25156 |
| PBKDF2 | 1.288 | 1.370 | 1.013 | 48522 | 93643 | 52880 |
| DH-2048 | 1.670 | 2.370 | 0.984 | 226586 | 378560 | 251877 |
| Pedersen | 1.876 | 3.549 | 0.885 | 225695 | 853145 | 236863 |
| Schnorr | 188.600 | 6.425 | 1.946 | 229437 | 2301691 | 286970 |

## Lab3 upgrade stability
- Passed sampled mode/stage checks: 3/3.
- Scheduled and immediate modes both passed chain view, receipt, event, version, and output consistency.

## Figure outputs
- figure_wasm_experiments.svg
- figure_wasm_experiments.pdf
- figure_wasm_experiments.tiff
- figure_wasm_experiments.png
