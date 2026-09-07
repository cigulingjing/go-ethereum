# Figure contract

Core conclusion:
WASM-based cryptographic upgrades complete across a validated five-node Clique network within one block interval and expose execution overheads that depend on algorithm complexity.

Figure archetype:
Quantitative grid.

Target/output:
Paper-ready vector and high-resolution raster exports: SVG, PDF, TIFF, and PNG.

Backend:
Python with matplotlib.

Final size:
7.2 x 6.2 inches, two-column manuscript figure.

Panel map:
- a: Lab1 five-node upgrade latency by algorithm, showing end-to-end completion and post-event WASM readiness.
- b: Lab2 mean eth_call latency for upgrade, contract, and precompile implementations.
- c: Lab2 eth_estimateGas values for the same three implementations.
- d: Historical upgrade-stability panel retained only in the generated artifact; not used as current manuscript experiment content.

Evidence hierarchy:
- Hero evidence: Lab1 all-node completion latency and Lab2 output-matched execution metrics across all comparable algorithms.
- Validation evidence: Lab2 output-matched execution metrics across all comparable algorithms.
- Controls/robustness: Raw JSON/CSV, node logs, network validation snapshots, per-sample timing arrays.

Statistics needed:
Use observed values only. Report Lab1 per-round five-node completion times and Lab2 n=100 eth_call samples per implementation after n=10 warmup. Do not report upgrade-stability or Lab3 pass counts as manuscript experiment statistics.

Source data needed:
Derive all plotted values from the raw result files in this run directory without dropping observations.

Reviewer risk:
Single-run Lab1 measurements are reproducible implementation measurements rather than replicated statistical estimates. Lab2 latency includes local RPC overhead and should be interpreted with the recorded network and node context. Upgrade Consistency is handled as protocol security analysis, not as a manuscript experiment.
