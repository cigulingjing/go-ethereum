# Figure contract

Core conclusion:
WASM-based cryptographic upgrades complete across a validated five-node Clique network within one block interval, preserve scheduled and immediate upgrade consistency, and expose execution overheads that depend on algorithm complexity.

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
- d: Lab3 scheduled/immediate consistency matrix across chain, receipt, version, and output checks.

Evidence hierarchy:
- Hero evidence: Lab1 all-node completion latency and Lab3 all-pass consistency.
- Validation evidence: Lab2 output-matched execution metrics across all comparable algorithms.
- Controls/robustness: Raw JSON/CSV, node logs, network validation snapshots, per-sample timing arrays.

Statistics needed:
Use observed values only. Report Lab1 per-round five-node completion times, Lab2 n=100 eth_call samples per implementation after n=10 warmup, and Lab3 all-node sample pass counts.

Source data needed:
Derive all plotted values from the raw result files in this run directory without dropping observations.

Reviewer risk:
Single-run Lab1 and Lab3 are reproducible smoke/formal measurements for the current implementation rather than replicated statistical estimates. Lab2 latency includes local RPC overhead and should be interpreted with the recorded network and node context.
