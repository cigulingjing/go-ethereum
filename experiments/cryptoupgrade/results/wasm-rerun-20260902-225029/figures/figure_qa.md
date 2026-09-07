# Figure QA

Backend:
Python / matplotlib only.

Exports:
- `figure_wasm_experiments.svg`
- `figure_wasm_experiments.pdf`
- `figure_wasm_experiments.tiff`
- `figure_wasm_experiments.png`

Automated checks:
- `validate_figure.py --backend python`: ready for visual QA, 18 pass, 2 warn, 0 fail.
- `audit_pdf_text.py --min-pt 5`: pass, minimum PDF text run 5.6 pt.

Warnings reviewed:
- Raster DPI is passed as the `dpi` argument to the export helper. The rendered PNG and TIFF are both 4389 x 3909 px from a 7.2 x 6.4 inch figure, matching 600 dpi export.
- Log-scale panels assert positive plotted values before applying log axes; the validator does not identify these dataframe positivity guards.

Panel audit:

| Panel | Unique claim | Source data | Labels/legend | Collision check | Pass |
|---|---|---|---|---|---|
| a | Five-node WASM upgrades complete within roughly one block interval, with post-event readiness tails separated from end-to-end latency. | Lab1 raw JSON/CSV, 7 algorithms, 5 nodes each. | Direct algorithm tick labels and shared two-item legend. | No value-label crowding after removing dense top annotations. | yes |
| b | Upgrade, Solidity, and precompile call latency differ by algorithm after setup. | Lab2 raw JSON, 100 measured eth_call samples per implementation. | Shared method legend with consistent colors. | Rotated labels anchored; error bars clear bars. | yes |
| c | Gas estimates show different resource profiles for identical logical inputs. | Lab2 raw JSON gasEstimate fields. | Same method colors as panel b. | Log-scale bars clear axes and labels. | yes |
| d | Scheduled and immediate WASM upgrades preserve chain, receipt, event, version, and output consistency across all five nodes. | Lab3 raw JSON/CSV. | Matrix cells show 5/5 passing nodes. | Row labels wrap correctly and do not overlap cells. | yes |
