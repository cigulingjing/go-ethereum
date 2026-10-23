#!/usr/bin/env python3
"""Audit actual receipts and draw the single-transaction PBKDF2 billing comparison."""

import argparse
import csv
import json
import statistics
from collections import defaultdict
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt


PATHS = ("solidityCalls", "evocryptCalls")
COLORS = {"solidityCalls": "#34495e", "evocryptCalls": "#c26b32"}
LABELS = {"solidityCalls": "Solidity (SHA-256 primitive)", "evocryptCalls": "EvoCrypt (configured tariff)"}


def aggregate(directory):
    manifest = json.loads((directory / "manifest.json").read_text(encoding="utf-8"))
    samples = [json.loads(line) for line in (directory / "samples.jsonl").read_text(encoding="utf-8").splitlines()]
    groups = defaultdict(list)
    for sample in samples:
        if (sample["iterations"] != manifest["iterations"] or sample["path"] not in PATHS or
                sample["callsPerTx"] not in manifest["levels"] or sample["round"] < 1 or
                sample["round"] > manifest["rounds"] or sample["repeat"] < 1 or
                sample["repeat"] > manifest["repeats"]):
            raise ValueError("sample does not match frozen manifest")
        groups[(sample["path"], sample["callsPerTx"], sample["round"])].append(sample)
    rows = []
    for count in manifest["levels"]:
        for path in PATHS:
            round_medians = []
            for round_number in range(1, manifest["rounds"] + 1):
                group = groups[(path, count, round_number)]
                if len(group) != manifest["repeats"] or {entry["repeat"] for entry in group} != set(range(1, manifest["repeats"] + 1)):
                    raise ValueError(f"missing/duplicate receipts: {path} n={count} round={round_number}")
                for sample in group:
                    if (sample["status"] != 1 or sample.get("error") or sample["gasUsed"] <= 21000 or
                            sample["parentCalls"] != 0 or not sample["separatorTxHash"] or not sample["blockHash"] or
                            sample["outputHash"] != manifest["expectedOutputHash"]):
                        raise ValueError(f"failed receipt or parent/output check: {path} n={count}")
                    if path == "evocryptCalls":
                        if len(sample.get("quotedCallGas", [])) != count:
                            raise ValueError("missing individual on-chain quote reconstruction")
                        base_gas = manifest["profile"]["algorithms"][0]["bands"][0]["gas"]
                        if count <= manifest["profile"]["threshold"] and any(quote != base_gas for quote in sample["quotedCallGas"]):
                            raise ValueError("unexpected pre-threshold quote")
                round_medians.append(statistics.median(entry["gasUsed"] for entry in group))
            rows.append({"callsPerTx": count, "path": path, "samples": manifest["rounds"] * manifest["repeats"],
                         "medianGas": statistics.median(round_medians), "minRoundMedianGas": min(round_medians),
                         "maxRoundMedianGas": max(round_medians)})
    if len(samples) != len(rows) * manifest["rounds"] * manifest["repeats"]:
        raise ValueError("unexpected additional samples")
    return manifest, rows


def render(directory):
    manifest, rows = aggregate(directory)
    with (directory / "billing-summary.csv").open("w", newline="", encoding="utf-8") as output:
        writer = csv.DictWriter(output, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)

    plt.rcParams["font.family"] = "sans-serif"
    plt.rcParams["font.sans-serif"] = ["DejaVu Sans"]
    plt.rcParams.update({"font.size": 9, "svg.fonttype": "none",
                         "pdf.fonttype": 42, "axes.spines.top": False, "axes.spines.right": False})
    figure, axis = plt.subplots(figsize=(7.2, 4.1))
    for path in PATHS:
        selected = [row for row in rows if row["path"] == path]
        central = [row["medianGas"] for row in selected]
        axis.errorbar([row["callsPerTx"] for row in selected], central,
                      yerr=[[center - row["minRoundMedianGas"] for row, center in zip(selected, central)],
                            [row["maxRoundMedianGas"] - center for row, center in zip(selected, central)]],
                      marker="o", markersize=4, capsize=2, color=COLORS[path], label=LABELS[path])
    axis.axvline(manifest["profile"]["threshold"], color="#888888", linestyle="--", linewidth=0.9)
    threshold = manifest["profile"]["threshold"]
    axis.annotate(f"threshold N₀ = {threshold}", (threshold, 0.93), xycoords=("data", "axes fraction"),
                  xytext=(4, 0), textcoords="offset points", fontsize=8)
    axis.set(xlabel=f"PBKDF2 calls within one transaction (iterations fixed at {manifest['iterations']})",
             ylabel="Mined receipt gasUsed (gas)", xticks=manifest["levels"])
    axis.grid(axis="y", alpha=0.2)
    axis.legend(frameon=False, fontsize=8, loc="lower center", bbox_to_anchor=(0.5, 1.02), ncol=2)
    figure.tight_layout()
    figures = directory / "figures"
    figures.mkdir(exist_ok=True)
    figure.savefig(figures / "calls-vs-cumulative-gas.svg", bbox_inches="tight")
    figure.savefig(figures / "calls-vs-cumulative-gas.pdf", bbox_inches="tight")
    figure.savefig(figures / "calls-vs-cumulative-gas.png", dpi=600, bbox_inches="tight")
    figure.savefig(figures / "calls-vs-cumulative-gas.tiff", dpi=600, bbox_inches="tight",
                   pil_kwargs={"compression": "tiff_lzw"})
    plt.close(figure)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    render(parser.parse_args().directory)
