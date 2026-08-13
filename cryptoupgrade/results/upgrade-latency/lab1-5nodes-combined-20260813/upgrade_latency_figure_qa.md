# Upgrade Latency Figure QA

Core conclusion: 5-node Clique upgrade latency is dominated by the post-receipt activation phase, and node-level completion times remain tightly clustered within each algorithm.

Figure archetype: quantitative grid

Target journal/output: manuscript figure / paper result figure

Backend: Python

Final size: 183 mm x 104 mm

Panel map:
- a: stacked phase decomposition of end-to-end upgrade latency
- b: raw node-level submit-to-complete distribution with mean and min-max span

Evidence hierarchy:
- hero evidence: panel a total and phase decomposition
- validation evidence: panel b raw node measurements
- controls/robustness: consistent 5-node layout across all algorithms

Statistics needed:
- n = 5 nodes per algorithm
- center = mean node submit-to-complete time
- spread = min-max across nodes
- no inferential test; this figure is descriptive

Source data needed:
- summary.csv
- one nodes.csv file per algorithm

Image-integrity notes:
- no raster image manipulation; all marks come directly from quantitative CSV outputs
- submit-to-tx-hash is a millisecond-scale stage, so it is annotated separately to avoid misreading it as absent
- event capture and receipt are effectively simultaneous in this listener path, so the dominant post-receipt stage is labeled as activation

Reviewer risk:
- the first phase is visually tiny relative to the full upgrade time, so the callout and phase label prevent over-interpretation
- node-level scatter is raw data; the bar total comes from the summary row and is checked against the component sum

Panel audit:

| Panel | Unique claim | Center/summary | Spread/interval | Replicate unit | Labels/legend | Collision check | Pass |
|---|---|---|---|---|---|---|---|
| a | Which phase dominates upgrade latency across algorithms | stacked phase total per algorithm | none; phase durations are deterministic from the run | one upgrade round per algorithm | phase legend only | clear at final size | yes |
| b | How tightly node completion times cluster within each algorithm | mean node submit-to-complete | min-max across 5 nodes | node | text note only | clear at final size | yes |

Source summary:

| algorithm | total_s | mean_node_s | node_min_s | node_max_s | slowest_node |
|---|---:|---:|---:|---:|---|
| Add | 23.012 | 22.935 | 22.892 | 23.012 | node5 |
| Blake2bSum256 | 20.353 | 20.272 | 20.237 | 20.353 | node4 |
| Sha256 | 24.739 | 24.708 | 24.698 | 24.739 | node1 |
| PedersenCommit | 25.910 | 25.794 | 25.727 | 25.910 | node4 |
| SchnorrVerify | 26.513 | 26.397 | 26.193 | 26.513 | node1 |
| Pbkdf2Sha256 | 27.430 | 27.328 | 27.176 | 27.430 | node5 |
| Dh2048Secret | 31.411 | 31.063 | 30.684 | 31.411 | node3 |
