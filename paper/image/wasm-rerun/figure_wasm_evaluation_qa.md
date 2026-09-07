# Figure QA

Backend:
Python / matplotlib.

Exports:
- `figure_wasm_evaluation.svg`
- `figure_wasm_evaluation.pdf`
- `figure_wasm_evaluation.tiff`
- `figure_wasm_evaluation.png`

Source data:
- `figure_wasm_evaluation_source_data.csv`

Panel audit:

| Panel | Unique claim | Source data | Labels/legend | Exclusion check | Pass |
|---|---|---|---|---|---|
| a | Five-node WASM upgrades complete within roughly one block interval, with post-event readiness tails separated from end-to-end latency. | Lab1 raw JSON/CSV, 7 algorithms, 5 nodes each. | Direct algorithm tick labels and shared two-item legend. | No consistency metrics plotted. | yes |
| b | Upgrade, Solidity, and precompile call latency differ by algorithm after setup. | Lab2 raw JSON, 100 measured eth_call samples per implementation. | Shared method legend with consistent colors. | No consistency metrics plotted. | yes |
| c | Gas estimates show different resource profiles for identical logical inputs. | Lab2 raw JSON gasEstimate fields. | Same method colors as panel b. | No consistency metrics plotted. | yes |

Current boundary:
This figure is the manuscript Evaluation figure. It does not include Lab3, scheduled/immediate baseline rows, chain-view checks, receipt/event checks, version checks, output checks, or State Root claims.
