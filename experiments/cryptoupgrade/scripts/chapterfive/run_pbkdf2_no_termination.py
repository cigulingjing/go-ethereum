#!/usr/bin/env python3
"""Repeat the EvoCrypt PBKDF2 timing grid with wazero termination checks disabled."""

import argparse
import csv
import hashlib
import json
import random
import shutil
import statistics
import subprocess
from datetime import datetime, timezone
from pathlib import Path

from run_pbkdf2 import LEVELS, ROOT, rpc_query, run_command


def validated_samples(result_path, iterations, expected_samples):
    result = json.loads(result_path.read_text(encoding="utf-8"))["results"][0]
    reference = "0x" + hashlib.pbkdf2_hmac("sha256", b"password", b"salt", iterations, 32).hex()
    implementations = result["implementations"]
    if result["outputMatched"] is not True or len(implementations) != 1:
        raise ValueError("expected one successful EvoCrypt implementation")
    implementation = implementations[0]
    if implementation["scheme"] != "upgrade" or implementation["outputValue"] != reference:
        raise ValueError(f"incorrect EvoCrypt PBKDF2 output at iteration {iterations}")
    samples = implementation["latency"]["sampleMillis"]
    if len(samples) != expected_samples or implementation["latency"]["warmup"] != 10:
        raise ValueError("missing samples or incorrect warmup")
    return samples


def collect(rpc, output, wasm, binary, close_on_context_done):
    output.mkdir(parents=True, exist_ok=False)
    archived_wasm = output / "pbkdf2_sha256.wasm"
    shutil.copyfile(wasm, archived_wasm)
    manifest = {
        "experiment": "chapter-five-5.2.1-pbkdf2-upgrade-termination-control",
        "startedAt": datetime.now(timezone.utc).isoformat(),
        "closeOnContextDone": close_on_context_done,
        "runtimeSetting": ("EVOCRYPT_EXPERIMENT_DISABLE_WAZERO_TERMINATION=1" if not close_on_context_done
                           else "EVOCRYPT_EXPERIMENT_DISABLE_WAZERO_TERMINATION unset"),
        "warning": ("WASM context cancellation and execution budget are not enforced in this isolated node"
                    if not close_on_context_done else "default wazero termination check remains enabled"),
        "levels": LEVELS, "rounds": 3, "warmupPerRound": 10, "samplesPerRound": 30,
        "timingBoundary": "client-side eth_call RPC request/response, excludes setup",
        "passwordHex": "0x70617373776f7264", "saltHex": "0x73616c74", "keyLength": 32,
        "rpc": rpc, "nodeCount": 1, "gethBinary": str(binary.resolve()),
        "gethBinarySha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
        "gethClientVersion": rpc_query(rpc, "web3_clientVersion", []),
        "chainId": rpc_query(rpc, "eth_chainId", []),
        "genesisHash": rpc_query(rpc, "eth_getBlockByNumber", ["0x0", False])["hash"],
        "wasmSha256": hashlib.sha256(archived_wasm.read_bytes()).hexdigest(),
        "commit": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        "roundOrderSeed": 20261022,
    }
    (output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    measurements = {iterations: {} for iterations in LEVELS}
    registered = False
    for round_number in range(4):
        order = list(LEVELS)
        if round_number:
            random.Random(20261022 + round_number).shuffle(order)
        for iterations in order:
            preflight = round_number == 0
            result_path = output / (f"preflight-{iterations}.json" if preflight else
                                    f"round-{round_number}-{iterations}.json")
            command = ["go", "run", "./experiments/cryptoupgrade/scripts/benchexecutionefficiency",
                       "-rpc", rpc, "-algorithms", "Pbkdf2Sha256", "-schemes", "upgrade",
                       "-pbkdf-iterations", str(iterations), "-pbkdf-wasm", str(archived_wasm.resolve()),
                       "-warmup", str(0 if preflight else 10), "-n", str(1 if preflight else 30),
                       "-receipt-samples", "0", "-call-gas", "25000000",
                       f"-upload={str(not registered).lower()}", "-output-json", str(result_path)]
            outcome = run_command(command, 120)
            record = {"iterations": iterations, "round": round_number, "command": command,
                      "returncode": outcome.get("returncode"), "error": outcome.get("error"),
                      "stderr": outcome.get("stderr", "")[-3000:]}
            if outcome.get("returncode") == 0 and result_path.is_file():
                try:
                    samples = json.loads(result_path.read_text(encoding="utf-8"))["results"][0]["implementations"][0]
                    reference = "0x" + hashlib.pbkdf2_hmac("sha256", b"password", b"salt", iterations, 32).hex()
                    if samples["scheme"] != "upgrade" or samples["outputValue"] != reference:
                        raise ValueError("PBKDF2 reference mismatch")
                    if preflight:
                        if len(samples["latency"]["sampleMillis"]) != 1:
                            raise ValueError("missing preflight sample")
                    else:
                        measurements[iterations][round_number] = validated_samples(result_path, iterations, 30)
                    record["validated"] = True
                    record["result"] = result_path.name
                except (KeyError, IndexError, ValueError) as error:
                    record["error"] = str(error)
            else:
                record["error"] = record["error"] or record["stderr"]
            if preflight and record.get("validated"):
                registered = True
            with (output / "runs.jsonl").open("a", encoding="utf-8") as handle:
                handle.write(json.dumps(record) + "\n")
            if not record.get("validated"):
                raise RuntimeError(f"failed PBKDF2 iterations={iterations} round={round_number}: {record['error']}")
    with (output / "samples.jsonl").open("w", encoding="utf-8") as handle:
        for iterations, rounds in measurements.items():
            for round_number, samples in sorted(rounds.items()):
                for index, duration_ms in enumerate(samples):
                    handle.write(json.dumps({"iterations": iterations, "round": round_number,
                                             "path": "upgrade", "sample": index,
                                             "measurementKind": "eth_call_rpc",
                                             "durationNs": round(duration_ms * 1_000_000)}) + "\n")
    with (output / "summary.csv").open("w", newline="", encoding="utf-8") as handle:
        writer = csv.writer(handle)
        writer.writerow(("iterations", "complete", "timingSamples", "medianRpcMs",
                         "minRoundMedianMs", "maxRoundMedianMs"))
        for iterations, rounds in measurements.items():
            complete = len(rounds) == 3 and all(len(samples) == 30 for samples in rounds.values())
            medians = [statistics.median(rounds[number]) for number in sorted(rounds)]
            writer.writerow((iterations, complete, sum(map(len, rounds.values())),
                             statistics.median(medians) if complete else "",
                             min(medians) if complete else "", max(medians) if complete else ""))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--rpc", required=True)
    parser.add_argument("--wasm", required=True, type=Path)
    parser.add_argument("--geth-binary", required=True, type=Path)
    parser.add_argument("--close-on-context-done", choices=("true", "false"), default="false")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    collect(args.rpc, args.output, args.wasm, args.geth_binary, args.close_on_context_done == "true")
