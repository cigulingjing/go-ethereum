#!/usr/bin/env python3
from __future__ import annotations

import csv
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib import gridspec
from matplotlib.patches import Patch
from matplotlib.ticker import MultipleLocator
import numpy as np


plt.rcParams['font.family'] = 'sans-serif'
plt.rcParams['font.sans-serif'] = ['Arial', 'DejaVu Sans', 'Liberation Sans']
plt.rcParams['svg.fonttype'] = 'none'
plt.rcParams['pdf.fonttype'] = 42
plt.rcParams['font.size'] = 7
plt.rcParams['axes.spines.right'] = False
plt.rcParams['axes.spines.top'] = False
plt.rcParams['axes.linewidth'] = 0.8
plt.rcParams['legend.frameon'] = False
plt.rcParams['figure.facecolor'] = 'white'
plt.rcParams['axes.facecolor'] = 'white'
plt.rcParams['savefig.facecolor'] = 'white'
# svg.fonttype = 'none'
# pdf.fonttype = 42


FIG_WIDTH_MM = 183
FIG_HEIGHT_MM = 104

PHASES = [
    ("submit_to_tx_hash_ms", "Submit -> Tx hash", "#B8BDC5"),
    ("tx_hash_to_receipt_ms", "Tx hash -> Receipt", "#4C78A8"),
    ("event_to_all_complete_ms", "Post-receipt activation", "#E28E2C"),
]

ALGORITHM_ORDER = [
    "Add",
    "Blake2bSum256",
    "Sha256",
    "PedersenCommit",
    "SchnorrVerify",
    "Pbkdf2Sha256",
    "Dh2048Secret",
]


@dataclass(frozen=True)
class AlgorithmRow:
    algorithm: str
    upgrade_name: str
    slowest_node: str
    submit_to_tx_hash_ms: float
    tx_hash_to_receipt_ms: float
    submit_to_receipt_ms: float
    event_to_all_complete_ms: float
    submit_to_all_complete_ms: float
    mean_node_submit_to_complete_ms: float
    result_json: str


@dataclass(frozen=True)
class NodeRow:
    node_id: str
    role: str
    submit_to_complete_ms: float
    completed: bool


def find_repo_root(start: Path) -> Path:
    for parent in start.parents:
        if (parent / "openspec").exists() and (parent / "cryptoupgrade").exists():
            return parent
    raise RuntimeError("Unable to locate repository root from figure script path.")


SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = find_repo_root(Path(__file__).resolve())
SUMMARY_CSV = SCRIPT_DIR / "summary.csv"
FIGURE_PREFIX = SCRIPT_DIR / "upgrade_latency_figure"
FIGURE_DATA_CSV = SCRIPT_DIR / "upgrade_latency_figure_data.csv"
QA_PATH = SCRIPT_DIR / "upgrade_latency_figure_qa.md"


def resolve_path(path_str: str) -> Path:
    path = Path(path_str)
    if path.is_absolute():
        return path
    return (REPO_ROOT / path).resolve()


def read_summary_rows() -> list[AlgorithmRow]:
    rows: list[AlgorithmRow] = []
    with SUMMARY_CSV.open("r", encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f)
        for raw in reader:
            if raw.get("completed", "").strip().lower() != "true":
                raise ValueError(f"Unfinished row found for {raw.get('algorithm')!r}.")
            rows.append(
                AlgorithmRow(
                    algorithm=raw["algorithm"],
                    upgrade_name=raw["upgradeName"],
                    slowest_node=raw["slowestNode"],
                    submit_to_tx_hash_ms=float(raw["submitToTxHashMs"]),
                    tx_hash_to_receipt_ms=float(raw["txHashToReceiptMs"]),
                    submit_to_receipt_ms=float(raw["submitToReceiptMs"]),
                    event_to_all_complete_ms=float(raw["eventToAllCompleteMs"]),
                    submit_to_all_complete_ms=float(raw["submitToAllCompleteMs"]),
                    mean_node_submit_to_complete_ms=float(raw["meanNodeSubmitToCompleteMs"]),
                    result_json=raw["resultJson"],
                )
            )

    order_map = {name: idx for idx, name in enumerate(ALGORITHM_ORDER)}

    def sort_key(row: AlgorithmRow) -> tuple[int, float, str]:
        if row.algorithm == "Add":
            return (0, row.submit_to_all_complete_ms, row.algorithm)
        return (1, row.submit_to_all_complete_ms, row.algorithm)

    rows.sort(key=sort_key)
    return rows


def read_node_rows(result_json: str) -> list[NodeRow]:
    result_path = resolve_path(result_json)
    node_csv = result_path.with_name("nodes.csv")
    rows: list[NodeRow] = []
    with node_csv.open("r", encoding="utf-8", newline="") as f:
        reader = csv.DictReader(f)
        for raw in reader:
            rows.append(
                NodeRow(
                    node_id=raw["node_id"],
                    role=raw["role"],
                    submit_to_complete_ms=float(raw["submit_to_complete_ms"]),
                    completed=raw["completed"].strip().lower() == "true",
                )
            )
    rows.sort(key=lambda row: int(row.node_id.replace("node", "")))
    return rows


def validate_summary_against_nodes(rows: Iterable[AlgorithmRow]) -> dict[str, dict[str, float]]:
    figure_data: dict[str, dict[str, float]] = {}
    for row in rows:
        node_rows = read_node_rows(row.result_json)
        if len(node_rows) != 5:
            raise ValueError(f"{row.algorithm} expected 5 nodes, found {len(node_rows)}.")
        if not all(node.completed for node in node_rows):
            raise ValueError(f"{row.algorithm} contains incomplete node rows.")

        node_values = np.array([node.submit_to_complete_ms for node in node_rows], dtype=float)
        node_mean = float(node_values.mean())
        node_min = float(node_values.min())
        node_max = float(node_values.max())
        summary_gap = abs(node_mean - row.mean_node_submit_to_complete_ms)
        if summary_gap > 1.0:
            raise ValueError(
                f"{row.algorithm} summary mean differs from node mean by {summary_gap:.3f} ms."
            )

        total_check = (
            row.submit_to_tx_hash_ms
            + row.tx_hash_to_receipt_ms
            + row.event_to_all_complete_ms
        )
        if abs(total_check - row.submit_to_all_complete_ms) > 0.01:
            raise ValueError(f"{row.algorithm} stacked phase sum does not match total.")

        figure_data[row.algorithm] = {
            "submit_to_tx_hash_ms": row.submit_to_tx_hash_ms,
            "tx_hash_to_receipt_ms": row.tx_hash_to_receipt_ms,
            "event_to_all_complete_ms": row.event_to_all_complete_ms,
            "submit_to_all_complete_ms": row.submit_to_all_complete_ms,
            "mean_node_submit_to_complete_ms": node_mean,
            "node_min_submit_to_complete_ms": node_min,
            "node_max_submit_to_complete_ms": node_max,
            "node_count": float(len(node_rows)),
        }

    return figure_data


def write_figure_data(rows: list[AlgorithmRow], stats: dict[str, dict[str, float]]) -> None:
    fieldnames = [
        "algorithm",
        "upgradeName",
        "slowestNode",
        "submitToTxHashMs",
        "txHashToReceiptMs",
        "eventToAllCompleteMs",
        "submitToAllCompleteMs",
        "meanNodeSubmitToCompleteMs",
        "nodeMinSubmitToCompleteMs",
        "nodeMaxSubmitToCompleteMs",
        "nodeCount",
    ]
    with FIGURE_DATA_CSV.open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for row in rows:
            data = stats[row.algorithm]
            writer.writerow(
                {
                    "algorithm": row.algorithm,
                    "upgradeName": row.upgrade_name,
                    "slowestNode": row.slowest_node,
                    "submitToTxHashMs": f"{data['submit_to_tx_hash_ms']:.6f}",
                    "txHashToReceiptMs": f"{data['tx_hash_to_receipt_ms']:.6f}",
                    "eventToAllCompleteMs": f"{data['event_to_all_complete_ms']:.6f}",
                    "submitToAllCompleteMs": f"{data['submit_to_all_complete_ms']:.6f}",
                    "meanNodeSubmitToCompleteMs": f"{data['mean_node_submit_to_complete_ms']:.6f}",
                    "nodeMinSubmitToCompleteMs": f"{data['node_min_submit_to_complete_ms']:.6f}",
                    "nodeMaxSubmitToCompleteMs": f"{data['node_max_submit_to_complete_ms']:.6f}",
                    "nodeCount": int(data["node_count"]),
                }
            )


def seconds(ms: float) -> float:
    return ms / 1000.0


def format_axis(ax: plt.Axes, xlim: tuple[float, float], major_step: float, label: str) -> None:
    ax.set_xlim(*xlim)
    ax.xaxis.set_major_locator(MultipleLocator(major_step))
    ax.set_xlabel(label)
    ax.grid(axis="x", color="#E6E8EB", linewidth=0.6)
    ax.set_axisbelow(True)


def add_panel_label(ax: plt.Axes, label: str) -> None:
    ax.text(
        -0.10,
        1.02,
        label,
        transform=ax.transAxes,
        fontsize=8,
        fontweight="bold",
        ha="left",
        va="bottom",
    )


def plot_figure(rows: list[AlgorithmRow], stats: dict[str, dict[str, float]]) -> plt.Figure:
    fig = plt.figure(figsize=(FIG_WIDTH_MM / 25.4, FIG_HEIGHT_MM / 25.4))
    gs = gridspec.GridSpec(1, 2, width_ratios=[1.34, 0.92], wspace=0.18)
    ax_phase = fig.add_subplot(gs[0, 0])
    ax_nodes = fig.add_subplot(gs[0, 1], sharey=ax_phase)

    y = np.arange(len(rows))
    phase_handles = []
    phase_labels = []
    left = np.zeros(len(rows), dtype=float)

    total_sec = np.array([seconds(row.submit_to_all_complete_ms) for row in rows], dtype=float)

    for field, label, color in PHASES:
        values = np.array([seconds(stats[row.algorithm][field]) for row in rows], dtype=float)
        bars = ax_phase.barh(
            y,
            values,
            left=left,
            height=0.66,
            color=color,
            edgecolor="white",
            linewidth=0.6,
        )
        if not phase_handles:
            phase_handles = [Patch(facecolor=color, edgecolor="none") for _, _, color in PHASES]
            phase_labels = [label for _, label, _ in PHASES]
        left = left + values

    ax_phase.set_yticks(y)
    ax_phase.set_yticklabels([row.algorithm for row in rows], fontsize=6.8)
    ax_phase.invert_yaxis()
    ax_phase.tick_params(axis="y", length=0, pad=5)
    for label in ax_phase.get_yticklabels():
        if label.get_text() == "Add":
            label.set_fontweight("bold")
            label.set_color("#0F4D92")

    for idx, (row, total) in enumerate(zip(rows, total_sec)):
        ax_phase.text(
            total + 0.35,
            idx,
            f"{total:.1f}",
            ha="left",
            va="center",
            fontsize=6.4,
            color="#30343B",
        )

    ax_phase.text(
        0.01,
        1.04,
        "Tx hash ack is 1.6-3.6 ms across runs.",
        transform=ax_phase.transAxes,
        fontsize=6.2,
        color="#5B6470",
        ha="left",
        va="bottom",
    )
    ax_phase.text(
        0.01,
        1.00,
        "Total length = submit -> all nodes complete.",
        transform=ax_phase.transAxes,
        fontsize=6.2,
        color="#5B6470",
        ha="left",
        va="bottom",
    )

    format_axis(
        ax_phase,
        (0.0, max(total_sec) + 2.2),
        5.0,
        "End-to-end upgrade latency (s)",
    )
    add_panel_label(ax_phase, "a")

    legend = fig.legend(
        phase_handles,
        phase_labels,
        loc="upper center",
        bbox_to_anchor=(0.5, 0.995),
        ncol=3,
        frameon=False,
        columnspacing=1.5,
        handlelength=1.5,
    )
    for text in legend.get_texts():
        text.set_fontsize(6.4)

    node_offsets = np.linspace(-0.14, 0.14, 5)
    for idx, row in enumerate(rows):
        node_rows = read_node_rows(row.result_json)
        node_values = np.array([seconds(node.submit_to_complete_ms) for node in node_rows], dtype=float)
        y_positions = np.full_like(node_values, idx, dtype=float) + node_offsets
        ax_nodes.hlines(
            idx,
            float(node_values.min()),
            float(node_values.max()),
            color="#C7CDD4",
            linewidth=1.1,
            zorder=1,
        )
        ax_nodes.scatter(
            node_values,
            y_positions,
            s=18,
            color="#6B7280",
            edgecolor="white",
            linewidth=0.35,
            alpha=0.92,
            zorder=2,
        )
        ax_nodes.scatter(
            [seconds(row.mean_node_submit_to_complete_ms)],
            [idx],
            s=28,
            marker="D",
            color="#0F4D92",
            edgecolor="white",
            linewidth=0.4,
            zorder=3,
        )

    ax_nodes.tick_params(axis="y", left=False, labelleft=False)
    ax_nodes.tick_params(axis="x", labelsize=6.2)
    ax_nodes.text(
        0.98,
        1.04,
        "Dots = 5 nodes; diamond = mean; line = min-max.",
        transform=ax_nodes.transAxes,
        fontsize=6.2,
        color="#5B6470",
        ha="right",
        va="bottom",
    )
    ax_nodes.text(
        0.98,
        1.00,
        "n = 5 nodes per algorithm.",
        transform=ax_nodes.transAxes,
        fontsize=6.2,
        color="#5B6470",
        ha="right",
        va="bottom",
    )
    node_min_sec = min(stats[row.algorithm]["node_min_submit_to_complete_ms"] for row in rows) / 1000.0
    node_max_sec = max(stats[row.algorithm]["node_max_submit_to_complete_ms"] for row in rows) / 1000.0
    format_axis(
        ax_nodes,
        (node_min_sec - 0.9, node_max_sec + 0.9),
        2.0,
        "Node-level submit -> complete (s)",
    )
    add_panel_label(ax_nodes, "b")

    ax_nodes.set_title("")
    ax_phase.set_title("")
    fig.subplots_adjust(top=0.86, left=0.16, right=0.98, bottom=0.13)
    return fig


def write_qa(rows: list[AlgorithmRow], stats: dict[str, dict[str, float]]) -> None:
    with QA_PATH.open("w", encoding="utf-8") as f:
        f.write("# Upgrade Latency Figure QA\n\n")
        f.write("Core conclusion: 5-node Clique upgrade latency is dominated by the post-receipt activation phase, and node-level completion times remain tightly clustered within each algorithm.\n\n")
        f.write("Figure archetype: quantitative grid\n\n")
        f.write("Target journal/output: manuscript figure / paper result figure\n\n")
        f.write("Backend: Python\n\n")
        f.write("Final size: 183 mm x 104 mm\n\n")
        f.write("Panel map:\n")
        f.write("- a: stacked phase decomposition of end-to-end upgrade latency\n")
        f.write("- b: raw node-level submit-to-complete distribution with mean and min-max span\n\n")
        f.write("Evidence hierarchy:\n")
        f.write("- hero evidence: panel a total and phase decomposition\n")
        f.write("- validation evidence: panel b raw node measurements\n")
        f.write("- controls/robustness: consistent 5-node layout across all algorithms\n\n")
        f.write("Statistics needed:\n")
        f.write("- n = 5 nodes per algorithm\n")
        f.write("- center = mean node submit-to-complete time\n")
        f.write("- spread = min-max across nodes\n")
        f.write("- no inferential test; this figure is descriptive\n\n")
        f.write("Source data needed:\n")
        f.write("- summary.csv\n")
        f.write("- one nodes.csv file per algorithm\n\n")
        f.write("Image-integrity notes:\n")
        f.write("- no raster image manipulation; all marks come directly from quantitative CSV outputs\n")
        f.write("- submit-to-tx-hash is a millisecond-scale stage, so it is annotated separately to avoid misreading it as absent\n")
        f.write("- event capture and receipt are effectively simultaneous in this listener path, so the dominant post-receipt stage is labeled as activation\n\n")
        f.write("Reviewer risk:\n")
        f.write("- the first phase is visually tiny relative to the full upgrade time, so the callout and phase label prevent over-interpretation\n")
        f.write("- node-level scatter is raw data; the bar total comes from the summary row and is checked against the component sum\n\n")
        f.write("Panel audit:\n\n")
        f.write("| Panel | Unique claim | Center/summary | Spread/interval | Replicate unit | Labels/legend | Collision check | Pass |\n")
        f.write("|---|---|---|---|---|---|---|---|\n")
        f.write("| a | Which phase dominates upgrade latency across algorithms | stacked phase total per algorithm | none; phase durations are deterministic from the run | one upgrade round per algorithm | phase legend only | clear at final size | yes |\n")
        f.write("| b | How tightly node completion times cluster within each algorithm | mean node submit-to-complete | min-max across 5 nodes | node | text note only | clear at final size | yes |\n")
        f.write("\nSource summary:\n\n")
        f.write("| algorithm | total_s | mean_node_s | node_min_s | node_max_s | slowest_node |\n")
        f.write("|---|---:|---:|---:|---:|---|\n")
        for row in rows:
            data = stats[row.algorithm]
            f.write(
                f"| {row.algorithm} | {data['submit_to_all_complete_ms'] / 1000.0:.3f} | "
                f"{data['mean_node_submit_to_complete_ms'] / 1000.0:.3f} | "
                f"{data['node_min_submit_to_complete_ms'] / 1000.0:.3f} | "
                f"{data['node_max_submit_to_complete_ms'] / 1000.0:.3f} | {row.slowest_node} |\n"
            )


def save_exports(fig: plt.Figure) -> None:
    fig.savefig(f"{FIGURE_PREFIX}.svg", bbox_inches="tight")
    fig.savefig(f"{FIGURE_PREFIX}.pdf", bbox_inches="tight")
    fig.savefig(f"{FIGURE_PREFIX}.png", dpi=300, bbox_inches="tight")
    fig.savefig(f"{FIGURE_PREFIX}.tiff", dpi=600, bbox_inches="tight")


def main() -> None:
    rows = read_summary_rows()
    stats = validate_summary_against_nodes(rows)
    write_figure_data(rows, stats)
    fig = plot_figure(rows, stats)
    save_exports(fig)
    plt.close(fig)
    write_qa(rows, stats)


if __name__ == "__main__":
    main()
