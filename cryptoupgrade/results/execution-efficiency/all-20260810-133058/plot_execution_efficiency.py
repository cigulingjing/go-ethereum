#!/usr/bin/env python3

import csv
import re
from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
import numpy as np


BASE_DIR = Path(__file__).resolve().parent
RESULT_TXT = BASE_DIR / "result.txt"
LATENCY_PREFIX = BASE_DIR / "execution_efficiency_latency"
GAS_PREFIX = BASE_DIR / "execution_efficiency_gas_estimate"
CSV_PATH = BASE_DIR / "execution_efficiency_comparison.csv"
QA_PATH = BASE_DIR / "execution_efficiency_comparison_qa.md"
FIG_WIDTH_MM = 183
FIG_HEIGHT_MM = 82

SCHEME_ORDER = ["upgrade", "precompile", "contract"]
SCHEME_LABELS = {
    "upgrade": "Dynamic upgrade",
    "precompile": "Precompile",
    "contract": "Solidity contract",
}
COLORS = {
    "upgrade": "#3A7CA5",
    "precompile": "#8A8F98",
    "contract": "#D17A45",
}
ROW_RE = re.compile(
    r"^\s*(?P<algorithm>[A-Za-z0-9_]+)\s+"
    r"(?P<upgrade_ms>\d+(?:\.\d+)?)\s+ms\s*/\s*(?P<upgrade_gas>\d+)\s+gas\s+"
    r"(?P<precompile_ms>\d+(?:\.\d+)?)\s+ms\s*/\s*(?P<precompile_gas>\d+)\s+gas\s+"
    r"(?P<contract_ms>\d+(?:\.\d+)?)\s+ms\s*/\s*(?P<contract_gas>\d+)\s+gas\s*$"
)


mpl.rcParams.update(
    {
        "font.family": "sans-serif",
        "font.sans-serif": ["Arial", "Helvetica", "DejaVu Sans", "sans-serif"],
        "svg.fonttype": "none",
        "pdf.fonttype": 42,
        "font.size": 7,
        "axes.spines.right": False,
        "axes.spines.top": False,
        "axes.linewidth": 0.8,
        "axes.labelsize": 7,
        "axes.titlesize": 8,
        "xtick.labelsize": 6,
        "ytick.labelsize": 6,
        "legend.fontsize": 7,
        "figure.titlesize": 9,
    }
)


def load_rows():
    algorithms = []
    rows = []
    for line in RESULT_TXT.read_text(encoding="utf-8").splitlines():
        match = ROW_RE.match(line)
        if not match:
            continue
        data = match.groupdict()
        algorithm = data["algorithm"]
        algorithms.append(algorithm)
        for scheme in SCHEME_ORDER:
            rows.append(
                {
                    "algorithm": algorithm,
                    "scheme": scheme,
                    "scheme_label": SCHEME_LABELS[scheme],
                    "mean_ms": float(data[f"{scheme}_ms"]),
                    "gas": int(data[f"{scheme}_gas"]),
                }
            )

    if not rows:
        raise ValueError(f"No benchmark rows parsed from {RESULT_TXT}.")
    return algorithms, rows


def write_source_csv(rows):
    with CSV_PATH.open("w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=[
                "algorithm",
                "scheme",
                "scheme_label",
                "mean_ms",
                "gas",
            ],
        )
        writer.writeheader()
        writer.writerows(rows)


def format_gas(value, _pos=None):
    if value >= 1_000_000:
        return f"{value / 1_000_000:.2f}".rstrip("0").rstrip(".") + "M"
    if value >= 1_000:
        return f"{value / 1_000:.2f}".rstrip("0").rstrip(".") + "k"
    return f"{value:g}"


def annotate_contract_extremes(ax, algorithms, gas_values):
    for idx, algorithm in enumerate(algorithms):
        value = gas_values.get(algorithm)
        if value is None or value < 1_000_000:
            continue
        ax.text(
            idx + 0.24,
            value * 1.08,
            format_gas(value),
            ha="center",
            va="bottom",
            fontsize=6,
            color=COLORS["contract"],
        )


def grouped_rows(rows):
    return {
        scheme: [row for row in rows if row["scheme"] == scheme]
        for scheme in SCHEME_ORDER
    }


def grouped_bar_layout(algorithms):
    x = np.arange(len(algorithms))
    width = 0.25
    offsets = {"upgrade": -width, "precompile": 0.0, "contract": width}
    return x, width, offsets


def new_single_panel_figure():
    fig, ax = plt.subplots(
        1,
        1,
        figsize=(FIG_WIDTH_MM / 25.4, FIG_HEIGHT_MM / 25.4),
        constrained_layout=False,
    )
    return fig, ax


def style_grouped_axis(ax, algorithms, y_label):
    ax.set_ylabel(y_label)
    ax.set_xticks(np.arange(len(algorithms)))
    ax.set_xticklabels(algorithms, rotation=35, ha="right", rotation_mode="anchor")
    ax.grid(axis="y", color="#E6E8EB", linewidth=0.6)
    ax.set_axisbelow(True)


def add_legend(fig, ax):
    handles, labels = ax.get_legend_handles_labels()
    fig.legend(
        handles,
        labels,
        loc="upper center",
        bbox_to_anchor=(0.52, 1.03),
        ncol=3,
        handlelength=1.5,
        columnspacing=1.3,
        frameon=False,
    )


def save_exports(fig, prefix):
    fig.savefig(f"{prefix}.svg", bbox_inches="tight")
    fig.savefig(f"{prefix}.pdf", bbox_inches="tight")
    fig.savefig(f"{prefix}.png", dpi=300, bbox_inches="tight")
    fig.savefig(f"{prefix}.tiff", dpi=600, bbox_inches="tight")


def plot_latency(algorithms, rows):
    x, width, offsets = grouped_bar_layout(algorithms)
    by_scheme = grouped_rows(rows)
    fig, ax = new_single_panel_figure()

    for scheme in SCHEME_ORDER:
        scheme_rows = by_scheme[scheme]
        values = [row["mean_ms"] for row in scheme_rows]
        ax.bar(
            x + offsets[scheme],
            values,
            width=width,
            linewidth=0.5,
            edgecolor="white",
            color=COLORS[scheme],
            label=SCHEME_LABELS[scheme],
        )
    style_grouped_axis(ax, algorithms, "Latency (ms)")
    latency_max = max(row["mean_ms"] for row in rows)
    ax.set_ylim(0, max(1.2, latency_max * 1.16))
    fig.suptitle(
        "Latency comparison across cryptographic algorithms",
        y=1.1,
        fontweight="bold",
    )
    add_legend(fig, ax)
    save_exports(fig, LATENCY_PREFIX)
    return fig


def plot_gas(algorithms, rows):
    if any(row["gas"] <= 0 for row in rows):
        raise ValueError("Gas values must be strictly positive for log scaling.")

    x, width, offsets = grouped_bar_layout(algorithms)
    by_scheme = grouped_rows(rows)
    fig, ax = new_single_panel_figure()

    contract_gas = {}
    for scheme in SCHEME_ORDER:
        scheme_rows = by_scheme[scheme]
        values = [row["gas"] for row in scheme_rows]
        if scheme == "contract":
            contract_gas = {
                row["algorithm"]: row["gas"]
                for row in scheme_rows
            }
        ax.bar(
            x + offsets[scheme],
            values,
            width=width,
            linewidth=0.5,
            edgecolor="white",
            color=COLORS[scheme],
            label=SCHEME_LABELS[scheme],
        )
    ax.set_yscale("log")
    style_grouped_axis(ax, algorithms, "Gas estimate (log scale)")
    ax.yaxis.set_major_formatter(mpl.ticker.FuncFormatter(format_gas))
    ax.set_ylim(18_000, 14_000_000)
    ax.grid(axis="y", color="#E6E8EB", linewidth=0.6, which="both")
    annotate_contract_extremes(ax, algorithms, contract_gas)
    fig.suptitle(
        "Gas estimate comparison across cryptographic algorithms",
        y=1.1,
        fontweight="bold",
    )
    add_legend(fig, ax)
    save_exports(fig, GAS_PREFIX)
    return fig


def write_qa():
    with QA_PATH.open("w", encoding="utf-8") as f:
        f.write("# Execution Efficiency Figure QA\n\n")
        f.write(
            "Core conclusion: Dynamic upgrade and precompile implementations "
            "are generally close in latency, while Solidity contracts become "
            "substantially slower and more gas-expensive for complex algorithms.\n\n"
        )
        f.write("Archetype: quantitative grid, split into two single-metric figures.\n\n")
        f.write("Source: result.txt summary table.\n\n")
        f.write("Panel audit:\n\n")
        f.write("| Figure | Unique claim | Center/summary | Spread/interval | Replicate unit | Pass |\n")
        f.write("|---|---|---|---|---|---|\n")
        f.write(
            "| latency | Compares runtime latency across schemes and algorithms | mean latency reported in result.txt | none in source table | implementation | yes |\n"
        )
        f.write(
            "| gas estimate | Compares gas estimate across schemes and algorithms | gas estimate reported in result.txt | none, deterministic estimate | implementation | yes |\n"
        )
        f.write(
            "\nExported files: independent latency and gas-estimate SVG/PDF/PNG/TIFF figures, "
            "source CSV, and this QA note.\n"
        )


def main():
    algorithms, rows = load_rows()
    write_source_csv(rows)
    latency_fig = plot_latency(algorithms, rows)
    plt.close(latency_fig)
    gas_fig = plot_gas(algorithms, rows)
    plt.close(gas_fig)
    write_qa()


if __name__ == "__main__":
    main()
