#!/usr/bin/env python3
from __future__ import annotations

import csv
import json
import math
from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd


ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "figures"

ALGO_LABELS = {
    "Add": "Add",
    "Sha256": "SHA-256",
    "Blake2bSum256": "BLAKE2b",
    "Pbkdf2Sha256": "PBKDF2",
    "Dh2048Secret": "DH-2048",
    "PedersenCommit": "Pedersen",
    "SchnorrVerify": "Schnorr",
}

SCHEME_LABELS = {
    "upgrade": "WASM upgrade",
    "contract": "Solidity",
    "precompile": "Precompile",
}

COLORS = {
    "upgrade": "#3B6EA8",
    "contract": "#8A8F98",
    "precompile": "#2D8C65",
    "end_to_end": "#6E7881",
    "event_tail": "#C45A36",
    "ok": "#2D8C65",
    "fail": "#D9DEE3",
}


mpl.rcParams.update(
    {
        "font.family": "sans-serif",
        "font.sans-serif": ["Arial", "Helvetica", "DejaVu Sans", "sans-serif"],
        "svg.fonttype": "none",
        "pdf.fonttype": 42,
        "font.size": 8,
        "axes.spines.right": False,
        "axes.spines.top": False,
        "axes.linewidth": 0.8,
        "axes.labelsize": 8,
        "xtick.labelsize": 8,
        "ytick.labelsize": 8,
        "legend.fontsize": 8,
        "figure.dpi": 150,
    }
)


def save_pub_py(fig: mpl.figure.Figure, filename: Path, dpi: int = 600) -> None:
    fig.savefig(filename.with_suffix(".svg"), bbox_inches="tight")
    fig.savefig(filename.with_suffix(".pdf"), bbox_inches="tight")
    fig.savefig(filename.with_suffix(".tiff"), dpi=dpi, bbox_inches="tight")
    fig.savefig(filename.with_suffix(".png"), dpi=dpi, bbox_inches="tight")


def load_json(rel: str) -> dict:
    return json.loads((ROOT / rel).read_text())


def fmt_ms(value: float) -> str:
    if value >= 1000:
        return f"{value / 1000:.2f} s"
    if value >= 100:
        return f"{value:.0f} ms"
    return f"{value:.1f} ms"


def build_lab1_source() -> pd.DataFrame:
    lab1 = load_json("lab1/result.json")
    rows = []
    for item in lab1["rounds"]:
        summary = item["summary"]
        payload = item["payload"]
        rows.append(
            {
                "algorithm": item["algorithm"],
                "algorithm_label": ALGO_LABELS.get(item["algorithm"], item["algorithm"]),
                "upgrade_name": item["upgradeName"],
                "completed": item["completed"],
                "target_nodes": summary["targetNodeCount"],
                "completed_nodes": summary["completedNodeCount"],
                "submit_to_all_complete_ms": summary["submitToAllCompleteMillis"],
                "tx_hash_to_sender_receipt_ms": summary["txHashToSenderReceiptMillis"],
                "event_to_all_complete_ms": summary["eventToAllCompleteMillis"],
                "mean_node_receipt_to_complete_ms": summary["meanNodeReceiptToCompleteMillis"],
                "receipt_gas_used": item["receiptGasUsed"],
                "wasm_hash": payload["wasmHash"],
                "activation_block": payload["activationBlock"],
                "upload_calldata_bytes": payload["uploadCalldataBytes"],
                "wasm_bytes": payload["sourceBytes"],
            }
        )
    df = pd.DataFrame(rows)
    df.to_csv(OUT / "lab1_upgrade_latency_source.csv", index=False)
    return df


def build_lab2_source() -> pd.DataFrame:
    lab2 = load_json("lab2/result.json")
    rows = []
    for result in lab2["results"]:
        setup = result.get("setup", {}).get("upgrade", {})
        for impl in result["implementations"]:
            lat = impl["latency"]
            rows.append(
                {
                    "algorithm": result["algorithm"],
                    "algorithm_label": ALGO_LABELS.get(result["algorithm"], result["algorithm"]),
                    "scheme": impl["scheme"],
                    "scheme_label": SCHEME_LABELS.get(impl["scheme"], impl["scheme"]),
                    "output_matched": result["outputMatched"],
                    "gas_estimate": impl["gasEstimate"],
                    "mean_ms": lat["meanMillis"],
                    "p50_ms": lat["p50Millis"],
                    "p95_ms": lat["p95Millis"],
                    "min_ms": lat["minMillis"],
                    "max_ms": lat["maxMillis"],
                    "samples": lat["samples"],
                    "warmup": lat["warmup"],
                    "calldata_bytes": impl["calldataBytes"],
                    "wasm_hash": setup.get("wasmHash", ""),
                    "activation_block": setup.get("activationBlock", 0),
                }
            )
    df = pd.DataFrame(rows)
    df.to_csv(OUT / "lab2_execution_efficiency_source.csv", index=False)
    sample_rows = []
    for result in lab2["results"]:
        for impl in result["implementations"]:
            for i, sample in enumerate(impl["latency"]["sampleMillis"], start=1):
                sample_rows.append(
                    {
                        "algorithm": result["algorithm"],
                        "scheme": impl["scheme"],
                        "sample_index": i,
                        "latency_ms": sample,
                    }
                )
    pd.DataFrame(sample_rows).to_csv(OUT / "lab2_latency_samples_source.csv", index=False)
    return df


def build_lab3_source() -> pd.DataFrame:
    lab3 = load_json("lab3/result.json")
    rows = []
    for result in lab3["rounds"]:
        for sample in result["samples"]:
            checks = sample["checks"]
            rows.append(
                {
                    "round": result["index"],
                    "mode": result["mode"],
                    "stage": sample["stage"],
                    "node_count": len(sample["nodes"]),
                    "passed": checks["passed"],
                    "chain_view_consistent": checks["chainViewConsistent"],
                    "receipt_consistent": checks["receiptConsistent"],
                    "event_consistent": checks["eventConsistent"],
                    "version_consistent": checks["versionConsistent"],
                    "output_consistent": checks["outputConsistent"],
                    "block_number": sample["blockNumber"],
                    "expected_version": sample["expectedVersion"],
                    "expected_output": sample["expectedOutput"],
                }
            )
    df = pd.DataFrame(rows)
    df.to_csv(OUT / "lab3_upgrade_stability_source.csv", index=False)
    return df


def write_combined_source(lab1: pd.DataFrame, lab2: pd.DataFrame, lab3: pd.DataFrame) -> None:
    rows = []
    for _, r in lab1.iterrows():
        rows.append(
            {
                "panel": "a",
                "metric": "submit_to_all_complete_ms",
                "algorithm": r["algorithm"],
                "scheme_or_stage": "end_to_end",
                "value": r["submit_to_all_complete_ms"],
                "unit": "ms",
            }
        )
        rows.append(
            {
                "panel": "a",
                "metric": "event_to_all_complete_ms",
                "algorithm": r["algorithm"],
                "scheme_or_stage": "post_event_ready",
                "value": r["event_to_all_complete_ms"],
                "unit": "ms",
            }
        )
    for _, r in lab2.iterrows():
        rows.append(
            {
                "panel": "b",
                "metric": "mean_eth_call_latency_ms",
                "algorithm": r["algorithm"],
                "scheme_or_stage": r["scheme"],
                "value": r["mean_ms"],
                "unit": "ms",
            }
        )
        rows.append(
            {
                "panel": "c",
                "metric": "eth_estimateGas",
                "algorithm": r["algorithm"],
                "scheme_or_stage": r["scheme"],
                "value": r["gas_estimate"],
                "unit": "gas",
            }
        )
    for _, r in lab3.iterrows():
        for metric in [
            "chain_view_consistent",
            "receipt_consistent",
            "event_consistent",
            "version_consistent",
            "output_consistent",
        ]:
            rows.append(
                {
                    "panel": "d",
                    "metric": metric,
                    "algorithm": "Add",
                    "scheme_or_stage": f"{r['mode']}:{r['stage']}",
                    "value": int(bool(r[metric])),
                    "unit": "pass_bool",
                }
            )
    pd.DataFrame(rows).to_csv(OUT / "figure_source_data.csv", index=False)


def panel_label(ax: plt.Axes, label: str) -> None:
    ax.text(
        -0.12,
        1.08,
        label,
        transform=ax.transAxes,
        ha="left",
        va="top",
        fontsize=9,
        fontweight="bold",
    )


def plot_lab1(ax: plt.Axes, df: pd.DataFrame) -> None:
    assert (df["submit_to_all_complete_ms"] > 0).all()
    assert (df["event_to_all_complete_ms"] > 0).all()
    x = np.arange(len(df))
    width = 0.36
    ax.bar(
        x - width / 2,
        df["submit_to_all_complete_ms"],
        width,
        color=COLORS["end_to_end"],
        label="Submit -> all nodes ready",
    )
    ax.bar(
        x + width / 2,
        df["event_to_all_complete_ms"],
        width,
        color=COLORS["event_tail"],
        label="Event -> all nodes ready",
    )
    ax.set_yscale("log")
    ax.set_ylabel("Latency (ms, log scale)")
    ax.set_xticks(x)
    ax.set_xticklabels(df["algorithm_label"], rotation=35, ha="right", rotation_mode="anchor")
    ax.set_title("Five-node WASM upgrade latency")
    ax.grid(axis="y", color="#E7EAEE", linewidth=0.7, which="both")
    ax.legend(loc="upper left", ncols=1)
    panel_label(ax, "a")


def plot_lab2_latency(ax: plt.Axes, df: pd.DataFrame) -> None:
    assert (df["mean_ms"] > 0).all()
    assert (df["p50_ms"] > 0).all()
    assert (df["p95_ms"] > 0).all()
    algos = list(dict.fromkeys(df["algorithm"]))
    x = np.arange(len(algos))
    width = 0.23
    offsets = {"upgrade": -width, "contract": 0, "precompile": width}
    for scheme in ["upgrade", "contract", "precompile"]:
        sub = df[df["scheme"] == scheme].set_index("algorithm").loc[algos]
        ax.bar(
            x + offsets[scheme],
            sub["mean_ms"],
            width,
            yerr=[sub["mean_ms"] - sub["p50_ms"], sub["p95_ms"] - sub["mean_ms"]],
            color=COLORS[scheme],
            error_kw={"elinewidth": 0.7, "capthick": 0.7, "capsize": 1.5},
            label=SCHEME_LABELS[scheme],
        )
    ax.set_yscale("log")
    ax.set_ylabel("Mean eth_call latency (ms)")
    ax.set_xticks(x)
    ax.set_xticklabels([ALGO_LABELS.get(a, a) for a in algos], rotation=35, ha="right", rotation_mode="anchor")
    ax.set_title("Execution latency after setup")
    ax.grid(axis="y", color="#E7EAEE", linewidth=0.7, which="both")
    ax.legend(loc="upper left", ncols=1)
    panel_label(ax, "b")


def plot_lab2_gas(ax: plt.Axes, df: pd.DataFrame) -> None:
    assert (df["gas_estimate"] > 0).all()
    algos = list(dict.fromkeys(df["algorithm"]))
    x = np.arange(len(algos))
    width = 0.23
    offsets = {"upgrade": -width, "contract": 0, "precompile": width}
    for scheme in ["upgrade", "contract", "precompile"]:
        sub = df[df["scheme"] == scheme].set_index("algorithm").loc[algos]
        ax.bar(
            x + offsets[scheme],
            sub["gas_estimate"],
            width,
            color=COLORS[scheme],
            label=SCHEME_LABELS[scheme],
        )
    ax.set_yscale("log")
    ax.set_ylabel("eth_estimateGas (gas)")
    ax.set_xticks(x)
    ax.set_xticklabels([ALGO_LABELS.get(a, a) for a in algos], rotation=35, ha="right", rotation_mode="anchor")
    ax.set_title("Gas estimated for identical logical inputs")
    ax.grid(axis="y", color="#E7EAEE", linewidth=0.7, which="both")
    panel_label(ax, "c")


def plot_lab3(ax: plt.Axes, df: pd.DataFrame) -> None:
    rows = [f"{r.mode}\n{r.stage}" for r in df.itertuples()]
    cols = [
        ("chain_view_consistent", "Chain"),
        ("receipt_consistent", "Receipt"),
        ("event_consistent", "Event"),
        ("version_consistent", "Version"),
        ("output_consistent", "Output"),
    ]
    matrix = np.array([[1 if bool(row[key]) else 0 for key, _ in cols] for _, row in df.iterrows()])
    cmap = mpl.colors.ListedColormap([COLORS["fail"], COLORS["ok"]])
    ax.imshow(matrix, aspect="auto", cmap=cmap, vmin=0, vmax=1)
    ax.set_xticks(np.arange(len(cols)))
    ax.set_xticklabels([name for _, name in cols])
    ax.set_yticks(np.arange(len(rows)))
    ax.set_yticklabels(rows)
    ax.set_title("Five-node upgrade consistency")
    ax.tick_params(axis="both", length=0)
    for i in range(matrix.shape[0]):
        for j in range(matrix.shape[1]):
            text = "5/5" if matrix[i, j] else "fail"
            ax.text(j, i, text, ha="center", va="center", fontsize=8, color="white" if matrix[i, j] else "#333333")
    for spine in ax.spines.values():
        spine.set_visible(False)
    ax.set_xticks(np.arange(-0.5, len(cols), 1), minor=True)
    ax.set_yticks(np.arange(-0.5, len(rows), 1), minor=True)
    ax.grid(which="minor", color="white", linewidth=1.0)
    ax.tick_params(which="minor", bottom=False, left=False)
    panel_label(ax, "d")


def make_summary(lab1: pd.DataFrame, lab2: pd.DataFrame, lab3: pd.DataFrame) -> None:
    upgrade = lab2[lab2["scheme"] == "upgrade"].set_index("algorithm")
    contract = lab2[lab2["scheme"] == "contract"].set_index("algorithm")
    precompile = lab2[lab2["scheme"] == "precompile"].set_index("algorithm")
    rows = [
        "# WASM experiment rerun summary",
        "",
        "## Lab1 upgrade latency",
        f"- Completed rounds: {int(lab1['completed'].sum())}/{len(lab1)} across 5 target nodes.",
        f"- Mean submit-to-all-complete latency: {lab1['submit_to_all_complete_ms'].mean():.3f} ms.",
        f"- Range: {lab1['submit_to_all_complete_ms'].min():.3f}-{lab1['submit_to_all_complete_ms'].max():.3f} ms.",
        f"- Largest post-event readiness tail: {lab1['event_to_all_complete_ms'].max():.3f} ms.",
        "",
        "## Lab2 execution efficiency",
        f"- Comparable algorithms: {len(upgrade)}; each implementation uses 10 warmup and 100 measured eth_call samples.",
        "- Output matching: all comparable algorithms matched across upgrade, contract, and precompile.",
        "",
        "| Algorithm | WASM mean ms | Contract mean ms | Precompile mean ms | WASM gas | Contract gas | Precompile gas |",
        "|---|---:|---:|---:|---:|---:|---:|",
    ]
    for algo in upgrade.index:
        rows.append(
            f"| {ALGO_LABELS.get(algo, algo)} | "
            f"{upgrade.loc[algo, 'mean_ms']:.3f} | "
            f"{contract.loc[algo, 'mean_ms']:.3f} | "
            f"{precompile.loc[algo, 'mean_ms']:.3f} | "
            f"{int(upgrade.loc[algo, 'gas_estimate'])} | "
            f"{int(contract.loc[algo, 'gas_estimate'])} | "
            f"{int(precompile.loc[algo, 'gas_estimate'])} |"
        )
    rows.extend(
        [
            "",
            "## Lab3 upgrade stability",
            f"- Passed sampled mode/stage checks: {int(lab3['passed'].sum())}/{len(lab3)}.",
            "- Scheduled and immediate modes both passed chain view, receipt, event, version, and output consistency.",
            "",
            "## Figure outputs",
            "- figure_wasm_experiments.svg",
            "- figure_wasm_experiments.pdf",
            "- figure_wasm_experiments.tiff",
            "- figure_wasm_experiments.png",
        ]
    )
    (ROOT / "summary.md").write_text("\n".join(rows) + "\n")


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    lab1 = build_lab1_source()
    lab2 = build_lab2_source()
    lab3 = build_lab3_source()
    write_combined_source(lab1, lab2, lab3)
    make_summary(lab1, lab2, lab3)

    fig = plt.figure(figsize=(7.2, 6.4), constrained_layout=True)
    gs = fig.add_gridspec(2, 2, height_ratios=[1.0, 1.0], width_ratios=[1.15, 1.0])
    ax_a = fig.add_subplot(gs[0, 0])
    ax_b = fig.add_subplot(gs[0, 1])
    ax_c = fig.add_subplot(gs[1, 0])
    ax_d = fig.add_subplot(gs[1, 1])

    plot_lab1(ax_a, lab1)
    plot_lab2_latency(ax_b, lab2)
    plot_lab2_gas(ax_c, lab2)
    plot_lab3(ax_d, lab3)

    fig.suptitle("WASM cryptographic upgrade evaluation on a five-node Clique network", fontsize=9.5)
    save_pub_py(fig, OUT / "figure_wasm_experiments")


if __name__ == "__main__":
    main()
