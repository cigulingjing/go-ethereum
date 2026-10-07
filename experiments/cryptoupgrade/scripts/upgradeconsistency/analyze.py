#!/usr/bin/env python3
"""Analyze one UpgradeConsistencyProbe run without filling missing observations."""
import argparse, csv, json, math, os
from collections import defaultdict


def output_value(raw):
    if not raw:
        return ""
    try:
        return str(int(raw[2:] if raw.startswith("0x") else raw, 16))
    except ValueError:
        return raw


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--out-dir", required=True)
    args = parser.parse_args()
    os.makedirs(args.out_dir, exist_ok=True)
    run = json.load(open(args.input, encoding="utf-8"))
    H = int(run["activation_block"])
    rows = run.get("observations", [])
    with open(os.path.join(args.out_dir, "observations.csv"), "w", newline="", encoding="utf-8") as f:
        fields = ["node_id", "block_number", "block_hash", "tx_hash", "selected_version", "module_hash", "output", "gas_used", "receipt_status", "state_root", "local_state_root", "pending", "node_error"]
        writer = csv.DictWriter(f, fieldnames=fields)
        writer.writeheader()
        for row in rows:
            writer.writerow({field: row.get(field, "") for field in fields})

    by_tx = defaultdict(list)
    for row in rows:
        by_tx[row["tx_hash"]].append(row)
    checks = []
    metrics = {"early_switch_count": 0, "post_activation_old_version_count": 0, "output_gas_status_divergence_count": 0, "state_root_divergence_count": 0}
    complete_samples = 0
    expected_samples = 0
    for tx_hash, sample in by_tx.items():
        observed = [r for r in sample if not r.get("pending") and r.get("selected_version")]
        if not observed:
            continue
        block = int(observed[0].get("block_number", 0))
        expected = 1 if block < H else 2
        if block in range(max(0, H - 3), H + 4):
            expected_samples += 1
        versions = {int(r["selected_version"]) for r in observed}
        outputs = {output_value(r.get("output", "")) for r in observed}
        gas_status = {(r.get("gas_used"), r.get("receipt_status")) for r in observed}
        roots = {r.get("local_state_root") for r in observed if r.get("local_state_root")}
        complete = len(observed) == 20
        if complete and block in range(max(0, H - 3), H + 4):
            complete_samples += 1
        if block < H and any(v != 1 for v in versions):
            metrics["early_switch_count"] += 1
        if block >= H and any(v != 2 for v in versions):
            metrics["post_activation_old_version_count"] += 1
        if len(outputs) > 1 or len(gas_status) > 1:
            metrics["output_gas_status_divergence_count"] += 1
        if len(roots) > 1:
            metrics["state_root_divergence_count"] += 1
        checks.append({"tx_hash": tx_hash, "block_number": block, "expected_version": expected, "observed_nodes": len(observed), "coverage": len(observed) / 20, "complete": complete, "versions": sorted(versions), "outputs": sorted(outputs), "gas_status_pairs": len(gas_status), "local_state_roots": sorted(roots), "same_block_hash": len({r.get("block_hash") for r in observed}) <= 1})
    coverage = (complete_samples / expected_samples) if expected_samples else 0.0
    summary = {
        "algorithm": run["algorithm"], "activation_block": H, "sample_count": len(checks),
        "comparison_window": [max(0, H - 3), H + 3], "window_samples": expected_samples,
        "complete_window_samples": complete_samples, "sample_coverage": coverage,
        "checks": {
            "pre_activation_v1_output_1": all(c["block_number"] >= H or c["expected_version"] == 1 and c["outputs"] == ["1"] for c in checks),
            "post_activation_v2_output_2": all(c["block_number"] < H or c["expected_version"] == 2 and c["outputs"] == ["2"] for c in checks),
            "same_tx_consistent": all(c["complete"] and len(c["versions"]) == 1 and c["gas_status_pairs"] == 1 and c["same_block_hash"] for c in checks),
            "local_state_roots_consistent": all(len(c["local_state_roots"]) <= 1 and c["complete"] for c in checks),
            "all_selected_samples_cover_20_nodes": complete_samples == expected_samples and expected_samples > 0,
        }, "metrics": metrics, "samples": checks,
        "limitations": ["pending observations are excluded from consistency counts", "this experiment reports only the observed chain and does not establish universal consensus safety"],
    }
    json.dump(summary, open(os.path.join(args.out_dir, "summary.json"), "w", encoding="utf-8"), indent=2)
    with open(os.path.join(args.out_dir, "summary.csv"), "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f); writer.writerow(["check", "result"])
        for key, value in summary["checks"].items(): writer.writerow([key, value])
        for key, value in metrics.items(): writer.writerow([key, value])
        writer.writerow(["sample_coverage", coverage])
    try:
        import matplotlib.pyplot as plt
        nodes = [f"node{i}" for i in range(1, 21)]
        blocks = sorted({int(r["block_number"]) for r in rows if not r.get("pending") and r.get("block_number")})
        blocks = [b for b in blocks if H - 3 <= b <= H + 3]
        index = {b: j for j, b in enumerate(blocks)}
        values = [[math.nan for _ in blocks] for _ in nodes]
        for row in rows:
            if row.get("pending") or int(row.get("block_number", 0)) not in index or not row.get("selected_version"):
                continue
            values[nodes.index(row["node_id"])][index[int(row["block_number"])]] = int(row["selected_version"])
        fig, ax = plt.subplots(figsize=(max(7, len(blocks) * .8), 6))
        cmap = plt.matplotlib.colors.ListedColormap(["#3b82f6", "#ef4444"])
        masked = __import__("numpy").ma.masked_invalid(values)
        ax.imshow(masked, cmap=cmap, vmin=1, vmax=2, aspect="auto")
        ax.set_xticks(range(len(blocks)), [str(b) for b in blocks]); ax.set_yticks(range(20), nodes)
        ax.axvline(index[H], color="black", linewidth=1.5) if H in index else None
        ax.set_xlabel("Block height (H marked by black line)"); ax.set_ylabel("Node")
        ax.set_title("UpgradeConsistencyProbe: executed version")
        fig.tight_layout(); fig.savefig(os.path.join(args.out_dir, "version_heatmap.png"), dpi=180); plt.close(fig)
    except Exception as exc:
        summary["heatmap_error"] = str(exc)
    json.dump(summary, open(os.path.join(args.out_dir, "summary.json"), "w", encoding="utf-8"), indent=2)
    print(json.dumps({"summary": os.path.join(args.out_dir, "summary.json"), "coverage": coverage, "metrics": metrics}, indent=2))


if __name__ == "__main__":
    main()
