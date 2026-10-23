#!/usr/bin/env python3
"""Collect one-node, same-input PBKDF2 three-path call and receipt measurements."""

import argparse
import csv
import hashlib
import json
import platform
import random
import shutil
import statistics
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from urllib.request import Request, urlopen


ROOT = Path(__file__).resolve().parents[4]
LEVELS = (1, 2, 5, 10, 20, 50, 100)
SCHEMES = ("upgrade", "contract", "precompile")


def rpc_query(url, method, params):
    request = Request(url, data=json.dumps({"jsonrpc": "2.0", "id": 1,
                                            "method": method, "params": params}).encode(),
                      headers={"Content-Type": "application/json"})
    with urlopen(request, timeout=10) as response:
        payload = json.load(response)
    if "error" in payload:
        raise ValueError(f"{method}: {payload['error']}")
    return payload["result"]


def version(*command):
    return subprocess.check_output(command, text=True, stderr=subprocess.STDOUT).splitlines()[0]


def run_command(command, timeout):
    try:
        completed = subprocess.run(command, cwd=ROOT, capture_output=True, text=True,
                                   timeout=timeout, check=False)
    except subprocess.TimeoutExpired as error:
        return {"error": f"timeout after {timeout}s", "stdout": str(error.stdout),
                "stderr": str(error.stderr)}
    return {"returncode": completed.returncode, "stdout": completed.stdout,
            "stderr": completed.stderr}


def inspect_result(path, iterations, receipt_count):
    result = json.loads(path.read_text(encoding="utf-8"))["results"][0]
    reference = "0x" + hashlib.pbkdf2_hmac("sha256", b"password", b"salt", iterations, 32).hex()
    implementations = result["implementations"]
    if result["outputMatched"] is not True or {entry["scheme"] for entry in implementations} != set(SCHEMES):
        raise ValueError("three successful comparable implementations are required")
    for entry in implementations:
        if entry["outputValue"] != reference:
            raise ValueError(f"incorrect {entry['scheme']} PBKDF2 output at {iterations}")
        if len(entry.get("gasReceipts", [])) != receipt_count:
            raise ValueError(f"missing {entry['scheme']} transaction receipts")
        for receipt in entry.get("gasReceipts", []):
            if receipt["status"] != 1 or not receipt["blockHash"] or receipt["gasUsed"] <= 21000:
                raise ValueError("failed or incomplete receipt")
    return result


def collect(rpc, output, wasm, solc, rounds, samples, warmup, receipts, timeout):
    output.mkdir(parents=True, exist_ok=False)
    wasm = wasm.resolve()
    archived_wasm = output / "pbkdf2_sha256.wasm"
    shutil.copyfile(wasm, archived_wasm)
    wasm = archived_wasm.resolve()
    contract = ROOT / "experiments/cryptoupgrade/algorithm/contracts/src/Pbkdf2Sha256.sol"
    manifest = {
        "experiment": "chapter-five-5.2.1-pbkdf2", "startedAt": datetime.now(timezone.utc).isoformat(),
        "rpc": rpc, "nodeCount": 1, "backend": "geth-single-node-clique-period-0",
        "levels": LEVELS, "rounds": rounds, "warmupPerPath": warmup,
        "ethCallSamplesPerPathPerRound": samples, "receiptsPerPathPerRound": receipts,
        "passwordHex": "0x70617373776f7264", "saltHex": "0x73616c74", "keyLength": 32,
        "timingBoundary": "client-side eth_call RPC request/response, excludes setup",
        "gasBoundary": "successful mined transaction receipt.gasUsed including intrinsic and calldata",
        "pricingWarning": "EvoCrypt metadata gas is configured, not a measured compute cost",
        "solidityPrimitive": "Solidity PBKDF2/HMAC loop uses the SHA-256 precompile",
        "preflightPolicy": "all seven levels tested before timing; failures retained without retuning",
        "roundOrderSeed": 20261022,
        "wasmPath": str(wasm),
        "wasmSha256": hashlib.sha256(wasm.read_bytes()).hexdigest(),
        "soliditySha256": hashlib.sha256(contract.read_bytes()).hexdigest(),
        "python": platform.python_version(), "machine": platform.platform(),
        "commit": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        "gethBinarySha256": hashlib.sha256((ROOT / "build/bin/geth").read_bytes()).hexdigest(),
        "gethClientVersion": rpc_query(rpc, "web3_clientVersion", []),
        "chainId": rpc_query(rpc, "eth_chainId", []),
        "genesisHash": rpc_query(rpc, "eth_getBlockByNumber", ["0x0", False])["hash"],
        "gasLimit": int(rpc_query(rpc, "eth_getBlockByNumber", ["latest", False])["gasLimit"], 16),
        "go": version("go", "version"), "tinygo": version("tinygo", "version"),
        "solcPath": solc, "solc": version(solc, "--version"),
    }
    (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    observations = []
    preflight = []
    registered = False

    def invoke(iterations, round_number, is_preflight):
        nonlocal registered
        location = output / (f"preflight-{iterations}.json" if is_preflight else f"round-{round_number}-{iterations}.json")
        command = ["go", "run", "./experiments/cryptoupgrade/scripts/benchexecutionefficiency",
                   "-rpc", rpc, "-algorithms", "Pbkdf2Sha256", "-pbkdf-iterations", str(iterations),
                   "-pbkdf-wasm", str(wasm), "-solc", solc,
                   "-warmup", str(0 if is_preflight else warmup), "-n", str(1 if is_preflight else samples),
                   "-receipt-samples", str(0 if is_preflight else receipts),
                   "-call-gas", "25000000", f"-upload={str(not registered).lower()}",
                   "-output-json", str(location)]
        outcome = run_command(command, timeout)
        record = {"iterations": iterations, "round": round_number, "preflight": is_preflight,
                  "command": command, "returncode": outcome.get("returncode"),
                  "error": outcome.get("error"), "stderr": outcome.get("stderr", "")[-5000:],
                  "stdout": outcome.get("stdout", "")[-3000:]}
        if outcome.get("returncode") == 0 and location.is_file():
            try:
                record["result"] = str(location.relative_to(output))
                record["validated"] = True
                inspected = inspect_result(location, iterations, 0 if is_preflight else receipts)
                if not is_preflight:
                    for implementation in inspected["implementations"]:
                        common = {"iterations": iterations, "round": round_number,
                                  "path": implementation["scheme"],
                                  "outputHash": implementation["outputSha256"],
                                  "calldataBytes": implementation["calldataBytes"]}
                        for index, millis in enumerate(implementation["latency"]["sampleMillis"]):
                            observations.append({**common, "measurementKind": "eth_call_rpc",
                                                 "sample": index, "durationNs": int(round(millis * 1_000_000))})
                        for index, receipt in enumerate(implementation["gasReceipts"]):
                            observations.append({**common, "measurementKind": "receipt_gas",
                                                 "sample": index, **receipt})
            except (ValueError, KeyError) as error:
                record["validated"] = False
                record["error"] = str(error)
        else:
            record["validated"] = False
            record["error"] = outcome.get("error") or outcome.get("stderr", "")[-3000:]
        registered = registered or (record["validated"] and is_preflight)
        (output / "preflight.jsonl" if is_preflight else output / "runs.jsonl").open(
            "a", encoding="utf-8").write(json.dumps(record) + "\n")
        return record

    for iterations in LEVELS:
        preflight.append(invoke(iterations, 0, True))
    for round_number in range(1, rounds + 1):
        order = list(LEVELS)
        random.Random(20261022 + round_number).shuffle(order)
        for iterations in order:
            if any(item["iterations"] == iterations and not item["validated"] for item in preflight):
                continue
            invoke(iterations, round_number, False)
    with (output / "samples.jsonl").open("w", encoding="utf-8") as handle:
        for sample in observations:
            handle.write(json.dumps(sample) + "\n")
    aggregate(output, observations, rounds, samples, receipts)


def aggregate(output, observations, rounds, samples, receipts):
    rows = []
    for iterations in LEVELS:
        for scheme in SCHEMES:
            group = [item for item in observations if item["iterations"] == iterations and item["path"] == scheme]
            timing_rounds = [[item["durationNs"] / 1e6 for item in group
                              if item["round"] == round_number and item["measurementKind"] == "eth_call_rpc"]
                             for round_number in range(1, rounds + 1)]
            gas_values = [item["gasUsed"] for item in group if item["measurementKind"] == "receipt_gas"]
            complete = all(len(values) == samples for values in timing_rounds) and len(gas_values) == rounds * receipts
            medians = [statistics.median(values) for values in timing_rounds if values]
            rows.append({"iterations": iterations, "path": scheme, "complete": complete,
                         "timingSamples": sum(map(len, timing_rounds)), "receiptSamples": len(gas_values),
                         "medianRpcMs": statistics.median(medians) if complete else "",
                         "minRoundMedianMs": min(medians) if complete else "",
                         "maxRoundMedianMs": max(medians) if complete else "",
                         "medianReceiptGas": statistics.median(gas_values) if complete else ""})
    with (output / "summary.csv").open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--rpc", default="http://127.0.0.1:8545")
    parser.add_argument("--wasm", type=Path, required=True)
    parser.add_argument("--solc", default="/home/liuqi/project/.tools/solc/solc-0.8.26")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--rounds", type=int, default=3)
    parser.add_argument("--samples", type=int, default=30)
    parser.add_argument("--warmup", type=int, default=10)
    parser.add_argument("--receipts", type=int, default=3)
    parser.add_argument("--timeout", type=int, default=180)
    args = parser.parse_args()
    if args.rounds < 3 or args.samples < 30 or args.warmup < 10 or args.receipts < 3:
        parser.error("requires >=3 rounds, >=30 RPC samples, >=10 warmups and >=3 receipts")
    collect(args.rpc, args.output, args.wasm, args.solc, args.rounds, args.samples, args.warmup, args.receipts, args.timeout)
