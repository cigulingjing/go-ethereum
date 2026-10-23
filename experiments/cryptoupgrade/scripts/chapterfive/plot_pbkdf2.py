#!/usr/bin/env python3
"""Render PBKDF2 RPC-time and receipt-Gas figures from validated raw samples."""

import argparse
import csv
import json
import statistics
from collections import defaultdict
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt


PALETTE = {"contract": "#34495e", "precompile": "#338a7a", "upgrade": "#c26b32"}
LABELS = {"contract": "Solidity (SHA-256 precompile primitive)",
          "precompile": "Native precompile", "upgrade": "EvoCrypt WASM"}


def load_verified(directory):
    manifest = json.loads((directory / "manifest.json").read_text(encoding="utf-8"))
    samples = [json.loads(line) for line in (directory / "samples.jsonl").read_text(encoding="utf-8").splitlines()]
    with (directory / "summary.csv").open(encoding="utf-8") as handle:
        published = {(int(row["iterations"]), row["path"]): row for row in csv.DictReader(handle)}
    groups = defaultdict(list)
    for sample in samples:
        groups[(sample["iterations"], sample["path"], sample["measurementKind"], sample["round"])].append(sample)
    values = {}
    for iterations in manifest["levels"]:
        for path in PALETTE:
            medians = []
            receipts = []
            for round_number in range(1, manifest["rounds"] + 1):
                timings = groups[(iterations, path, "eth_call_rpc", round_number)]
                gas = groups[(iterations, path, "receipt_gas", round_number)]
                if len(timings) != manifest["ethCallSamplesPerPathPerRound"] or len(gas) != manifest["receiptsPerPathPerRound"]:
                    continue
                medians.append(statistics.median(item["durationNs"] / 1e6 for item in timings))
                receipts.extend(item["gasUsed"] for item in gas if item["status"] == 1)
            summary = published[(iterations, path)]
            if summary["complete"] != "True":
                continue
            if len(medians) != manifest["rounds"] or len(receipts) != manifest["rounds"] * manifest["receiptsPerPathPerRound"]:
                raise ValueError(f"incomplete or inconsistent samples for {iterations} {path}")
            midpoint = statistics.median(medians)
            receipt_median = statistics.median(receipts)
            if abs(midpoint - float(summary["medianRpcMs"])) > 1e-8 or receipt_median != float(summary["medianReceiptGas"]):
                raise ValueError("summary figure points do not match raw samples")
            values[(iterations, path)] = (midpoint, min(medians), max(medians), receipt_median)
    return manifest, values


def draw(directory):
    manifest, values = load_verified(directory)
    figure_dir = directory / "figures"
    figure_dir.mkdir(exist_ok=True)
    plt.rcParams.update({"font.family": "DejaVu Sans", "font.size": 9,
                         "svg.fonttype": "none", "pdf.fonttype": 42,
                         "axes.spines.top": False, "axes.spines.right": False})
    for name, ylabel, index in (("iterations-vs-execution-time", "eth_call RPC latency (ms)", 0),
                                ("iterations-vs-receipt-gas", "Transaction receipt gasUsed (gas)", 3)):
        fig, axis = plt.subplots(figsize=(6.5, 4))
        for path, color in PALETTE.items():
            points = [(level, values[(level, path)]) for level in manifest["levels"] if (level, path) in values]
            x_values = [level for level, _ in points]
            y_values = [point[index] for _, point in points]
            if index == 0:
                axis.errorbar(x_values, y_values, yerr=[
                    [point[0] - point[1] for _, point in points],
                    [point[2] - point[0] for _, point in points],
                ], color=color, marker="o", markersize=4, capsize=2, label=LABELS[path])
            else:
                axis.plot(x_values, y_values, color=color, marker="o", markersize=4, label=LABELS[path])
        axis.set(xlabel="PBKDF2 iterations per call", ylabel=ylabel)
        axis.set_xscale("log")
        axis.set_xlim(0.8, 120)
        axis.set_xticks(manifest["levels"])
        axis.set_xticklabels([str(level) for level in manifest["levels"]])
        axis.minorticks_off()
        if index == 3:
            axis.set_yscale("log")
        axis.grid(axis="y", alpha=0.2)
        axis.legend(frameon=False, fontsize=7.5)
        fig.tight_layout()
        for extension in ("svg", "png", "pdf"):
            fig.savefig(figure_dir / f"{name}.{extension}", dpi=300, bbox_inches="tight")
        plt.close(fig)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    draw(parser.parse_args().directory)
