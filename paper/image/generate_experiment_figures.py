from __future__ import annotations

import csv
import json
import shutil
from dataclasses import dataclass
from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib.colors import ListedColormap
from matplotlib.patches import Patch


ROOT = Path(__file__).resolve().parents[2]
OUT = Path(__file__).resolve().parent
LAB1 = OUT / "lab1-upgrade-latency-availability"
LAB2 = OUT / "lab2-execution-efficiency"
LAB3 = OUT / "lab3-state-consistency"

FIGURE_EXTENSIONS = (".svg", ".pdf", ".png", ".tiff")

PALETTE = {
    "upgrade": "#4C78A8",
    "contract": "#F58518",
    "precompile": "#54A24B",
    "old": "#6B8DB9",
    "new": "#5FB37A",
    "pass": "#4DAF7C",
    "neutral": "#6E7781",
    "receipt": "#9C755F",
    "activation": "#72B7B2",
}


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
        "xtick.labelsize": 6.5,
        "ytick.labelsize": 6.5,
        "legend.fontsize": 6.5,
        "figure.dpi": 150,
    }
)


def ensure_output_dirs() -> None:
    for path in (LAB1, LAB2, LAB3):
        path.mkdir(parents=True, exist_ok=True)


def save_figure(fig: mpl.figure.Figure, directory: Path, stem: str) -> None:
    target = directory / stem
    fig.savefig(target.with_suffix(".svg"), bbox_inches="tight")
    fig.savefig(target.with_suffix(".pdf"), bbox_inches="tight")
    fig.savefig(target.with_suffix(".png"), dpi=300, bbox_inches="tight")
    fig.savefig(target.with_suffix(".tiff"), dpi=600, bbox_inches="tight")
    plt.close(fig)


def label_bars(ax: mpl.axes.Axes, bars, fmt: str, padding: float = 0.02) -> None:
    _, ymax = ax.get_ylim()
    offset = ymax * padding
    for bar in bars:
        value = bar.get_height()
        ax.text(
            bar.get_x() + bar.get_width() / 2,
            value + offset,
            fmt.format(value),
            ha="center",
            va="bottom",
            fontsize=6,
        )


def copy_existing_figures() -> None:
    sources = {
        LAB2 / "lab2_execution_efficiency_latency": ROOT
        / "cryptoupgrade/results/execution-efficiency/all-20260810-133058/execution_efficiency_latency",
        LAB2 / "lab2_execution_efficiency_gas_estimate": ROOT
        / "cryptoupgrade/results/execution-efficiency/all-20260810-133058/execution_efficiency_gas_estimate",
        LAB1 / "lab1_upgrade_latency_5nodes": ROOT
        / "cryptoupgrade/results/upgrade-latency/lab1-5nodes-combined-20260813/upgrade_latency_figure",
    }
    for target_stem, source_stem in sources.items():
        for ext in FIGURE_EXTENSIONS:
            source = source_stem.with_suffix(ext)
            if source.exists():
                shutil.copy2(source, target_stem.with_suffix(ext))


def plot_deployment_cost() -> None:
    rows = [
        {
            "algorithm": "Add",
            "method": "Upgrade module",
            "elapsed_s": 1.709800323,
            "gas": 85612,
        },
        {
            "algorithm": "Add",
            "method": "Solidity contract",
            "elapsed_s": 1.007160836,
            "gas": 60918,
        },
        {
            "algorithm": "Blake2b-256",
            "method": "Upgrade module",
            "elapsed_s": 1.709387805,
            "gas": 87180,
        },
        {
            "algorithm": "Blake2b-256",
            "method": "Solidity contract",
            "elapsed_s": 1.006741755,
            "gas": 852059,
        },
    ]
    algorithms = ["Add", "Blake2b-256"]
    methods = ["Upgrade module", "Solidity contract"]
    colors = [PALETTE["upgrade"], PALETTE["contract"]]
    width = 0.34
    x = list(range(len(algorithms)))

    fig, axes = plt.subplots(1, 2, figsize=(7.2, 2.65), constrained_layout=True)
    for ax, metric, ylabel, title, fmt in [
        (axes[0], "elapsed_s", "Deployment latency (s)", "a  Deployment latency", "{:.2f}"),
        (axes[1], "gas", "Gas used (thousand)", "b  Deployment gas", "{:.0f}"),
    ]:
        for method_index, method in enumerate(methods):
            values = []
            for algorithm in algorithms:
                value = next(
                    row[metric]
                    for row in rows
                    if row["algorithm"] == algorithm and row["method"] == method
                )
                if metric == "gas":
                    value = value / 1000
                values.append(value)
            positions = [pos + (method_index - 0.5) * width for pos in x]
            bars = ax.bar(
                positions,
                values,
                width=width,
                color=colors[method_index],
                edgecolor="white",
                linewidth=0.6,
                label=method,
            )
            label_bars(ax, bars, fmt)
        ax.set_xticks(x, algorithms)
        ax.set_ylabel(ylabel)
        ax.set_title(title, loc="left", fontweight="bold")
        ax.grid(axis="y", color="#E6E8EB", linewidth=0.7)
        ax.set_axisbelow(True)
    axes[0].set_ylim(0, 2.05)
    axes[1].set_ylim(0, 950)
    axes[0].legend(loc="upper left", bbox_to_anchor=(0, 1.18), ncol=2)
    save_figure(fig, LAB1, "lab1_deployment_upgrade_cost")


def plot_real_chain_precompile() -> None:
    result_path = ROOT / "experiments/cryptoupgrade/docs/real_chain_upgrade_vs_precompile_result.json"
    data = json.loads(result_path.read_text())
    upgrade_call = data["upgradeCall"]
    precompile_call = data["precompileCall"]
    setup = {
        "Upgrade module": data["upgradeSetup"],
        "Precompile": data["precompileSetup"],
    }
    methods = ["Upgrade module", "Precompile"]
    colors = [PALETTE["upgrade"], PALETTE["precompile"]]

    fig, axes = plt.subplots(2, 2, figsize=(7.2, 4.6), constrained_layout=True)

    setup_elapsed = [setup[method]["elapsedMillis"] for method in methods]
    bars = axes[0, 0].bar(methods, setup_elapsed, color=colors, edgecolor="white", linewidth=0.6)
    axes[0, 0].set_ylabel("Setup latency (ms)")
    axes[0, 0].set_title("a  Setup latency", loc="left", fontweight="bold")
    axes[0, 0].set_ylim(0, max(setup_elapsed) * 1.25 + 1)
    label_bars(axes[0, 0], bars, "{:.0f}")

    setup_gas = [setup[method]["gasUsed"] / 1000 for method in methods]
    bars = axes[0, 1].bar(methods, setup_gas, color=colors, edgecolor="white", linewidth=0.6)
    axes[0, 1].set_ylabel("Setup gas (thousand)")
    axes[0, 1].set_title("b  Setup gas", loc="left", fontweight="bold")
    axes[0, 1].set_ylim(0, max(setup_gas) * 1.25 + 1)
    label_bars(axes[0, 1], bars, "{:.1f}")

    latency_metrics = [
        ("Mean", upgrade_call["meanMillis"], precompile_call["meanMillis"]),
        ("P50", upgrade_call["p50Millis"], precompile_call["p50Millis"]),
        ("P95", upgrade_call["p95Millis"], precompile_call["p95Millis"]),
    ]
    width = 0.34
    x = list(range(len(latency_metrics)))
    for method_index, method in enumerate(methods):
        values = [row[method_index + 1] for row in latency_metrics]
        positions = [pos + (method_index - 0.5) * width for pos in x]
        bars = axes[1, 0].bar(
            positions,
            values,
            width=width,
            color=colors[method_index],
            edgecolor="white",
            linewidth=0.6,
            label=method,
        )
        label_bars(axes[1, 0], bars, "{:.2f}", padding=0.025)
    axes[1, 0].set_xticks(x, [row[0] for row in latency_metrics])
    axes[1, 0].set_ylabel("Call latency (ms)")
    axes[1, 0].set_title("c  Call latency", loc="left", fontweight="bold")
    axes[1, 0].set_ylim(0, max(row[1] for row in latency_metrics) * 1.55)
    axes[1, 0].legend(loc="upper left", ncol=2)

    call_gas = [
        upgrade_call["gasEstimate"] / 1000,
        precompile_call["gasEstimate"] / 1000,
    ]
    bars = axes[1, 1].bar(methods, call_gas, color=colors, edgecolor="white", linewidth=0.6)
    axes[1, 1].set_ylabel("Call gas estimate (thousand)")
    axes[1, 1].set_title("d  Call gas", loc="left", fontweight="bold")
    axes[1, 1].set_ylim(0, max(call_gas) * 1.25)
    label_bars(axes[1, 1], bars, "{:.1f}")

    for ax in axes.flat:
        ax.grid(axis="y", color="#E6E8EB", linewidth=0.7)
        ax.set_axisbelow(True)
    save_figure(fig, LAB2, "lab2_real_chain_precompile_comparison")


@dataclass(frozen=True)
class StabilitySample:
    round_index: int
    mode: str
    stage: str
    block_number: int
    node_id: str
    output: str
    ok: bool


def read_stability_samples() -> list[StabilitySample]:
    path = ROOT / "experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/samples.csv"
    with path.open(newline="") as f:
        reader = csv.DictReader(f)
        return [
            StabilitySample(
                round_index=int(row["round"]),
                mode=row["mode"],
                stage=row["stage"],
                block_number=int(row["block_number"]),
                node_id=row["node_id"],
                output=row["output"],
                ok=row["ok"].lower() == "true",
            )
            for row in reader
        ]


def plot_upgrade_stability() -> None:
    samples = read_stability_samples()
    result_path = ROOT / "experiments/cryptoupgrade/results/upgrade-stability/local-5nodes-formal-20260813/result.json"
    result = json.loads(result_path.read_text())

    node_ids = sorted({sample.node_id for sample in samples})
    sample_keys = []
    for sample in samples:
        key = (sample.round_index, sample.mode, sample.stage, sample.block_number)
        if key not in sample_keys:
            sample_keys.append(key)
    sample_keys.sort(key=lambda item: (item[0], 0 if item[2] == "before" else 1))

    matrix = []
    for key in sample_keys:
        row = []
        for node_id in node_ids:
            output = next(
                sample.output
                for sample in samples
                if (sample.round_index, sample.mode, sample.stage, sample.block_number) == key
                and sample.node_id == node_id
            )
            row.append(0 if output == "200" else 1)
        matrix.append(row)

    checks = ["chain view", "receipt", "event", "version", "output"]
    check_keys = [
        "chainViewConsistent",
        "receiptConsistent",
        "eventConsistent",
        "versionConsistent",
        "outputConsistent",
    ]
    check_matrix = [
        [1 if round_data["checks"][key] else 0 for key in check_keys]
        for round_data in result["rounds"]
    ]
    gas_by_mode = {}
    for round_data in result["rounds"]:
        gas_by_mode.setdefault(round_data["mode"], []).append(round_data["receipt"]["gasUsed"] / 1000)

    fig = plt.figure(figsize=(7.2, 3.6), constrained_layout=True)
    grid = fig.add_gridspec(1, 3, width_ratios=[1.45, 1.25, 0.9])
    ax_output = fig.add_subplot(grid[0, 0])
    ax_checks = fig.add_subplot(grid[0, 1])
    ax_gas = fig.add_subplot(grid[0, 2])

    ax_output.imshow(matrix, cmap=ListedColormap([PALETTE["old"], PALETTE["new"]]), vmin=0, vmax=1)
    labels = [
        f"R{round_index} {stage}\n{mode}\nB{block_number}"
        for round_index, mode, stage, block_number in sample_keys
    ]
    ax_output.set_xticks(range(len(node_ids)), node_ids)
    ax_output.set_yticks(range(len(labels)), labels)
    ax_output.set_title("a  Versioned output by node", loc="left", fontweight="bold")
    ax_output.tick_params(length=0)
    for y, row in enumerate(matrix):
        for x, value in enumerate(row):
            ax_output.text(x, y, "205" if value else "200", ha="center", va="center", fontsize=6, color="white")
    ax_output.legend(
        handles=[
            Patch(facecolor=PALETTE["old"], label="Old output 200"),
            Patch(facecolor=PALETTE["new"], label="New output 205"),
        ],
        loc="lower left",
        bbox_to_anchor=(0, -0.32),
        ncol=2,
    )

    ax_checks.imshow(check_matrix, cmap=ListedColormap(["#D95F5F", PALETTE["pass"]]), vmin=0, vmax=1)
    ax_checks.set_xticks(range(len(checks)))
    ax_checks.set_xticklabels(checks, rotation=35, ha="right", rotation_mode="anchor")
    ax_checks.set_yticks(range(len(result["rounds"])), [f"R{round_data['index']}\n{round_data['mode']}" for round_data in result["rounds"]])
    ax_checks.set_title("b  Consistency checks", loc="left", fontweight="bold")
    ax_checks.tick_params(length=0)
    for y, row in enumerate(check_matrix):
        for x, value in enumerate(row):
            ax_checks.text(x, y, "pass" if value else "fail", ha="center", va="center", fontsize=6, color="white")

    mode_order = ["scheduled", "immediate"]
    for index, mode in enumerate(mode_order):
        values = gas_by_mode[mode]
        ax_gas.scatter(
            [index] * len(values),
            values,
            s=26,
            color=PALETTE["upgrade"] if mode == "scheduled" else PALETTE["precompile"],
            zorder=3,
        )
        mean_value = sum(values) / len(values)
        ax_gas.plot([index - 0.18, index + 0.18], [mean_value, mean_value], color="#20262E", linewidth=1.1)
        ax_gas.text(index, mean_value + 0.18, f"{mean_value:.1f}", ha="center", va="bottom", fontsize=6)
    ax_gas.set_xticks(range(len(mode_order)))
    ax_gas.set_xticklabels(["Scheduled", "Immediate"], rotation=25, ha="right", rotation_mode="anchor")
    ax_gas.set_ylabel("Receipt gas used (thousand)")
    ax_gas.set_title("c  Upgrade transaction gas", loc="left", fontweight="bold")
    ax_gas.set_ylim(84, 88)
    ax_gas.grid(axis="y", color="#E6E8EB", linewidth=0.7)
    ax_gas.set_axisbelow(True)

    save_figure(fig, LAB3, "lab3_upgrade_state_consistency_5nodes")


def main() -> None:
    ensure_output_dirs()
    copy_existing_figures()
    plot_deployment_cost()
    plot_real_chain_precompile()
    plot_upgrade_stability()


if __name__ == "__main__":
    main()
