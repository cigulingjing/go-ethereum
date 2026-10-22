#!/usr/bin/env python3
import argparse
import json
import statistics


REFERENCE_LEVELS = {
    "Add": 64,
    "Sha256": 256,
    "Blake2bSum256": 256,
    "Pbkdf2Sha256": 500,
    "Dh2048Secret": 128,
    "PedersenCommit": 128,
    "SchnorrVerify": 256,
    "PolynomialMul": 4,
}


def aggregate(runs):
    if len(runs) < 2:
        raise ValueError("at least two independent runs are required")
    baseline = runs[0]
    hashes = baseline["wasmSha256"]
    if set(hashes) != set(REFERENCE_LEVELS):
        raise ValueError("incomplete algorithm coverage")
    anchor_gas = baseline["anchorGas"]
    for run in runs:
        if (run["wasmSha256"] != hashes or run["anchorGas"] != anchor_gas
                or run["anchorAlgorithm"] != "Add"
                or len(run["measurements"]) != 32):
            raise ValueError("runs use different artifacts or settings")
        for setting in ("goVersion", "os", "architecture", "runtime", "cpuCount", "warmup", "samples"):
            if run.get(setting) != baseline.get(setting):
                raise ValueError(f"runs differ in {setting}")
    anchor_ns = int(statistics.median(run["anchorMedianNs"] for run in runs))
    if anchor_ns <= 0:
        raise ValueError("invalid anchor median")
    algorithms = {}
    for index in range(32):
        measurements = [run["measurements"][index] for run in runs]
        first = measurements[0]
        for measured in measurements:
            for key in ("algorithm", "metric", "value", "inputBytes", "outputSha256"):
                if measured[key] != first[key]:
                    raise ValueError(f"fixture mismatch for {first['algorithm']}")
        name = first["algorithm"]
        median_ns = int(statistics.median(item["medianNs"] for item in measurements))
        proportional = (anchor_gas * median_ns + anchor_ns - 1) // anchor_ns
        entry = algorithms.setdefault(name, {
            "metric": first["metric"],
            "referenceValue": REFERENCE_LEVELS[name],
            "wasmSha256": hashes[name],
            "measurements": [],
        })
        entry["measurements"].append({
            "value": first["value"],
            "inputBytes": first["inputBytes"],
            "medianNs": median_ns,
            "rawProportionalGas": proportional,
        })
    for name, entry in algorithms.items():
        reference = next((item for item in entry["measurements"]
                          if item["value"] == entry["referenceValue"]), None)
        if reference is None:
            raise ValueError(f"missing reference level for {name}")
        entry["referenceBaseGas"] = reference["rawProportionalGas"]
    return {
        "method": "ceil(anchorGas * median(run medians) / median(anchor medians))",
        "runs": len(runs),
        "anchorAlgorithm": "Add",
        "anchorGas": anchor_gas,
        "anchorMedianNs": anchor_ns,
        "algorithms": algorithms,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("runs", nargs="+", help="input JSON reports")
    parser.add_argument("--out", required=True, help="aggregate JSON path")
    args = parser.parse_args()
    with open(args.runs[0], encoding="utf-8") as first:
        runs = [json.load(first)]
    for path in args.runs[1:]:
        with open(path, encoding="utf-8") as source:
            runs.append(json.load(source))
    result = aggregate(runs)
    with open(args.out, "w", encoding="utf-8") as destination:
        json.dump(result, destination, indent=2, ensure_ascii=False)
        destination.write("\n")


if __name__ == "__main__":
    main()
