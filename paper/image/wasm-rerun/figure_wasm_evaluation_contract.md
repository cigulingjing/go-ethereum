# Figure contract

Core conclusion:
WASM-based cryptographic upgrades complete across a validated five-node Clique network within one block interval and expose execution overheads that depend on algorithm complexity.

Figure archetype:
Quantitative grid.

Target/output:
Paper-ready vector and high-resolution raster exports: SVG, PDF, TIFF, and PNG.

Source data:
`figure_wasm_evaluation_source_data.csv`, filtered to panels a-c only.

Backend:
Python with matplotlib.

Final size:
7.2 x 2.55 inches, two-column manuscript figure.

Panel map:
- a: Lab1 five-node upgrade latency by algorithm, showing end-to-end completion and post-event WASM readiness.
- b: Lab2 mean eth_call latency for upgrade, contract, and precompile implementations.
- c: Lab2 eth_estimateGas values for the same three implementations.

Exclusion:
Upgrade Consistency is not plotted. The current manuscript does not set up a Lab3 consistency experiment and does not cite scheduled/immediate harness rows as Evaluation evidence.
