#!/usr/bin/env python3
"""Run a frozen one-factor-at-a-time on-chain PBKDF2 billing sweep."""

import argparse
import csv
import hashlib
import json
import os
import shutil
import subprocess
import time
from fractions import Fraction
from math import ceil
from pathlib import Path

import yaml

from run_pbkdf2 import ROOT, rpc_query


DEFAULT_ROOT = ROOT / "experiments/cryptoupgrade/results/chapter-five/20261023-pricing-sensitivity"
BASE_CONFIG = ROOT / "experiments/cryptoupgrade/config/local-1node-pricing.yaml"
WASM = ROOT / "experiments/cryptoupgrade/results/chapter-five/20261022-single-node-pbkdf2-rebuilt/pbkdf2_sha256.wasm"


def check_matrix(matrix):
    baseline = matrix["baseline"]
    for case in matrix["cases"]:
        changed = {field for field in baseline if case[field] != baseline[field]}
        if case["factor"] == "baseline":
            if changed:
                raise ValueError(f"baseline differs: {case['name']}")
        elif case["factor"] == "alpha":
            if not changed or not changed <= {"alphaNumerator", "alphaDenominator"}:
                raise ValueError(f"multiple factors differ: {case['name']}: {changed}")
        elif changed != {case["factor"]}:
            raise ValueError(f"multiple factors differ: {case['name']}: {changed}")
    if len(matrix["cases"]) != 7 or len({case["name"] for case in matrix["cases"]}) != 7:
        raise ValueError("expected seven independent frozen cases")


def run_command(command, log_path, environment=None):
    with log_path.open("a", encoding="utf-8") as log:
        log.write("\n$ " + " ".join(map(str, command)) + "\n")
        log.flush()
        subprocess.run(command, cwd=ROOT, stdout=log, stderr=subprocess.STDOUT,
                       env=environment, check=True, timeout=3600)


def run_case(root, binary, matrix, case, index):
    case_root = root / case["name"]
    if (case_root / "billing/chain-audit.json").exists():
        return
    if (case_root / "billing").exists():
        raise FileExistsError(f"partial receipts must be retained; use a fresh directory: {case_root}")
    case_root.mkdir(parents=True, exist_ok=True)
    failed_log = case_root / "node.log"
    if failed_log.exists():
        failed_log.rename(case_root / "node-failed-start.log")
    network_root = ROOT / "experiments/cryptoupgrade/output/network/pricing-sensitivity" / case["name"]
    config = yaml.safe_load(BASE_CONFIG.read_text(encoding="utf-8"))
    config["network"].update(name=f"pricing-sensitivity-{case['name']}",
                             chainId=11223400 + index, networkId=11223400 + index)
    config["local"]["httpPortBase"] = 19100 + index
    config["local"]["outputDir"] = str(network_root)
    config["accounts"]["signer"]["privateKey"] = str((BASE_CONFIG.parent / "signer.key").resolve())
    config["accounts"]["signer"]["address"] = f"0x{config['accounts']['signer']['address']:040x}"
    config["cryptoupgrade"]["addSource"] = str((ROOT / "experiments/cryptoupgrade/algorithm/go/wasm/add.wasm").resolve())
    pricing_path = case_root / "pricing.json"
    pricing_path.write_text(json.dumps({
        "threshold": case["threshold"],
        "alpha": {"numerator": case["alphaNumerator"], "denominator": case["alphaDenominator"]},
        "bands": [{"maxMetric": 2**64-1, "gas": case["baseGas"]}],
    }, indent=2) + "\n", encoding="utf-8")
    config_path = case_root / "config.yaml"
    config_path.write_text(yaml.safe_dump(config, sort_keys=False), encoding="utf-8")
    log_path = case_root / "commands.log"
    run_command(["go", "run", "./experiments/cryptoupgrade/scripts/render", "-config", str(config_path),
                 "-out", str(network_root), "-geth", str(binary)], log_path)
    genesis_path = network_root / "genesis.json"
    shutil.copyfile(genesis_path, case_root / "genesis.json")
    node_root = network_root / "node1"
    run_command([str(binary), "init", "--datadir", str(node_root / "datadir"), str(genesis_path)], log_path)
    rpc_port = 19100 + index
    rpc = f"http://127.0.0.1:{rpc_port}"
    environment = os.environ.copy()
    environment.pop("EVOCRYPT_EXPERIMENT_DISABLE_WAZERO_TERMINATION", None)
    environment["GETH_CRYPTOUPGRADE_PLUGIN_DIR"] = str(node_root / "plugin")
    environment["CRYPTOUPGRADE_MODULE"] = str(ROOT)
    command = [str(binary), "--config", str(node_root / "config.toml"),
               "--datadir", str(node_root / "datadir"), "--networkid", str(11223400 + index),
               "--port", str(31400 + index), "--nodekey", str(node_root / "nodekey"),
               "--syncmode", "full", "--http", "--http.addr", "127.0.0.1",
               "--http.port", str(rpc_port), "--http.api", "web3,eth,net,admin,debug,clique,txpool,miner",
               "--authrpc.addr", "127.0.0.1", "--authrpc.port", str(19500 + index),
               "--ipcdisable", "--allow-insecure-unlock", "--mine", "--unlock",
               "0xF5F871aA6Bd253914705898c66251f994aa426FA", "--password",
               str(node_root / "password.txt"), "--miner.pending.feeRecipient",
               "0xF5F871aA6Bd253914705898c66251f994aa426FA", "--cache", "128", "--verbosity", "1"]
    with (case_root / "node.log").open("w", encoding="utf-8") as node_log:
        node = subprocess.Popen(command, cwd=ROOT, env=environment, stdout=node_log, stderr=subprocess.STDOUT)
        try:
            for _ in range(100):
                if node.poll() is not None:
                    raise RuntimeError(f"node exited early for {case['name']}")
                try:
                    if int(rpc_query(rpc, "eth_chainId", []), 16) == 11223400 + index:
                        break
                    raise RuntimeError("unexpected experiment chain ID")
                except (OSError, TimeoutError, ConnectionError):
                    time.sleep(0.2)
            else:
                raise TimeoutError(f"node not ready: {case['name']}")
            provenance = {"case": case, "gethBinarySha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
                          "wasmSha256": hashlib.sha256(WASM.read_bytes()).hexdigest(),
                          "genesisFileSha256": hashlib.sha256(genesis_path.read_bytes()).hexdigest(),
                          "collectorSha256": hashlib.sha256((ROOT / "experiments/cryptoupgrade/scripts/chapterfive/benchbilling/main.go").read_bytes()).hexdigest(),
                          "closeOnContextDone": True, "nodeCommand": command, "rpc": rpc}
            (case_root / "provenance.json").write_text(json.dumps(provenance, indent=2) + "\n", encoding="utf-8")
            setup_path = case_root / "setup.json"
            run_command(["go", "run", "./experiments/cryptoupgrade/scripts/benchexecutionefficiency",
                         "-algorithms", "Pbkdf2Sha256", "-rpc", rpc, "-pbkdf-wasm", str(WASM),
                         "-pbkdf-iterations", "10", "-warmup", "2", "-n", "2",
                         "-receipt-samples", "0", "-output-json", str(setup_path)], log_path, environment)
            reference = json.loads(setup_path.read_text(encoding="utf-8"))["results"][0]["setup"]["contract"]["address"]
            run_command(["go", "run", "./experiments/cryptoupgrade/scripts/upload-wasm",
                         "-rpc", rpc, "-wasm", str(WASM), "-name", "Pbkdf2Sha256",
                         "-mode", "immediate", "-version", "2", "-pricing", str(pricing_path),
                         "-itype", "bytes,bytes,uint256,uint256", "-otype", "bytes",
                         "-algo-gas", str(case["baseGas"]), "-chain-id", str(11223400 + index),
                         "-key-file", config["accounts"]["signer"]["privateKey"]], log_path, environment)
            billing = case_root / "billing"
            run_command(["go", "run", "./experiments/cryptoupgrade/scripts/chapterfive/benchbilling",
                         "-rpc", rpc, "-reference", reference, "-genesis", str(genesis_path),
                         "-output", str(billing), "-rounds", str(matrix["controls"]["rounds"]),
                         "-repeats", str(matrix["controls"]["receiptsPerPathPerLevelPerRound"]),
                         "-warmup", str(matrix["controls"]["warmupPerPathPerLevel"]),
                         "-iterations", str(matrix["controls"]["internalIterations"]),
                         "-base-gas", str(case["baseGas"]), "-alpha-numerator", str(case["alphaNumerator"]),
                         "-alpha-denominator", str(case["alphaDenominator"]),
                         "-threshold", str(case["threshold"])], log_path, environment)
            run_command(["python3", "experiments/cryptoupgrade/scripts/chapterfive/plot_billing.py", str(billing)], log_path)
            run_command(["python3", "experiments/cryptoupgrade/scripts/chapterfive/audit_billing.py",
                         str(billing), "--rpc", rpc], log_path)
        finally:
            node.terminate()
            try:
                node.wait(timeout=15)
            except subprocess.TimeoutExpired:
                node.kill()
                node.wait(timeout=15)


def consolidate(root, matrix):
    rows = []
    solidity_by_count = {}
    non_tariff_by_count = {}
    for case in matrix["cases"]:
        billing = root / case["name"] / "billing"
        audit = json.loads((billing / "chain-audit.json").read_text(encoding="utf-8"))
        manifest = json.loads((billing / "manifest.json").read_text(encoding="utf-8"))
        profile = manifest["profile"]
        actual = (profile["algorithms"][0]["bands"][0]["gas"],
                  profile["alpha"]["numerator"], profile["alpha"]["denominator"], profile["threshold"])
        expected_profile = (case["baseGas"], case["alphaNumerator"], case["alphaDenominator"], case["threshold"])
        if actual != expected_profile:
            raise ValueError(f"wrong on-chain profile in manifest: {case['name']}")
        expected = (len(matrix["controls"]["callsPerTransaction"]) * 2 *
                    matrix["controls"]["rounds"] * matrix["controls"]["receiptsPerPathPerLevelPerRound"])
        if audit["verifiedTargetReceipts"] != expected:
            raise ValueError(f"incomplete receipt audit: {case['name']}")
        alpha = Fraction(case["alphaNumerator"], case["alphaDenominator"])
        quotes_by_count = {
            count: [ceil(case["baseGas"] * (1 + alpha) ** max(0, index - case["threshold"]))
                    for index in range(1, count + 1)]
            for count in matrix["controls"]["callsPerTransaction"]
        }
        for line in (billing / "samples.jsonl").read_text(encoding="utf-8").splitlines():
            sample = json.loads(line)
            if sample["path"] == "evocryptCalls" and sample["quotedCallGas"] != quotes_by_count[sample["callsPerTx"]]:
                raise ValueError(f"quote mismatch: {case['name']} n={sample['callsPerTx']}")
        with (billing / "billing-summary.csv").open(encoding="utf-8") as handle:
            records = list(csv.DictReader(handle))
        for count in matrix["controls"]["callsPerTransaction"]:
            by_path = {entry["path"]: entry for entry in records if int(entry["callsPerTx"]) == count}
            if set(by_path) != {"solidityCalls", "evocryptCalls"}:
                raise ValueError(f"missing paths: {case['name']} n={count}")
            solidity_gas = by_path["solidityCalls"]["medianGas"]
            if count in solidity_by_count and solidity_by_count[count] != solidity_gas:
                raise ValueError(f"Solidity control changed: {case['name']} n={count}")
            solidity_by_count[count] = solidity_gas
            non_tariff = int(float(by_path["evocryptCalls"]["medianGas"])) - sum(quotes_by_count[count])
            if count in non_tariff_by_count and non_tariff_by_count[count] != non_tariff:
                raise ValueError(f"non-tariff EvoCrypt gas changed: {case['name']} n={count}")
            non_tariff_by_count[count] = non_tariff
            rows.append({"case": case["name"], "factor": case["factor"], "baseGas": case["baseGas"],
                         "alpha": f"{case['alphaNumerator']}/{case['alphaDenominator']}",
                         "threshold": case["threshold"], "callsPerTx": count,
                         "solidityReceiptGas": solidity_gas,
                         "evocryptReceiptGas": by_path["evocryptCalls"]["medianGas"],
                         "algorithmQuoteSum": sum(quotes_by_count[count]),
                         "lastCallQuote": quotes_by_count[count][-1],
                         "evocryptNonTariffGas": non_tariff,
                         "minRoundMedianEvoGas": by_path["evocryptCalls"]["minRoundMedianGas"],
                         "maxRoundMedianEvoGas": by_path["evocryptCalls"]["maxRoundMedianGas"],
                         "emptyParentPaths": audit["emptyParentPaths"],
                         "samplesPerPath": by_path["evocryptCalls"]["samples"]})
    with (root / "sensitivity-summary.csv").open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=DEFAULT_ROOT)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--scenario", choices=["all"] + [case["name"] for case in json.loads((DEFAULT_ROOT / "matrix.json").read_text())["cases"]], default="all")
    arguments = parser.parse_args()
    root = arguments.root.resolve()
    binary = arguments.binary.resolve()
    matrix = json.loads((root / "matrix.json").read_text(encoding="utf-8"))
    check_matrix(matrix)
    for index, case in enumerate(matrix["cases"]):
        if arguments.scenario in ("all", case["name"]):
            print(f"running {case['name']} on chain {11223400 + index}", flush=True)
            run_case(root, binary, matrix, case, index)
    if all((root / case["name"] / "billing/chain-audit.json").exists() for case in matrix["cases"]):
        consolidate(root, matrix)
