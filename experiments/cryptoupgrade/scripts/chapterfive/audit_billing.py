#!/usr/bin/env python3
"""Re-query mined receipts and parent blocks for the frozen single-node billing run."""

import argparse
import hashlib
import json
from collections import defaultdict
from datetime import datetime, timezone
from pathlib import Path

from run_pbkdf2 import rpc_query


def audit(directory, rpc):
    manifest = json.loads((directory / "manifest.json").read_text(encoding="utf-8"))
    samples = [json.loads(line) for line in (directory / "samples.jsonl").read_text(encoding="utf-8").splitlines()]
    genesis = rpc_query(rpc, "eth_getBlockByNumber", ["0x0", False])
    if genesis["hash"] != manifest["genesisHash"]:
        raise ValueError("wrong experiment chain")
    derived = hashlib.pbkdf2_hmac("sha256", b"password", b"salt", manifest["iterations"], 32)
    if len(derived) != 32:
        raise ValueError("invalid PBKDF2 reference")
    observed = defaultdict(set)
    unique_transactions = set()
    empty_parents = 0
    for sample in samples:
        tx_hash = sample["txHash"]
        if tx_hash in unique_transactions:
            raise ValueError(f"duplicate transaction {tx_hash}")
        unique_transactions.add(tx_hash)
        receipt = rpc_query(rpc, "eth_getTransactionReceipt", [tx_hash])
        if receipt is None or int(receipt["status"], 16) != 1 or int(receipt["gasUsed"], 16) != sample["gasUsed"]:
            raise ValueError(f"receipt mismatch {tx_hash}")
        if receipt["blockHash"] != sample["blockHash"]:
            raise ValueError(f"receipt block mismatch {tx_hash}")
        target = rpc_query(rpc, "eth_getBlockByHash", [sample["blockHash"], False])
        if target["transactions"] != [tx_hash] or target["parentHash"] != sample["parentHash"]:
            raise ValueError(f"target block includes another transaction {tx_hash}")
        separator_receipt = rpc_query(rpc, "eth_getTransactionReceipt", [sample["separatorTxHash"]])
        if separator_receipt is None or int(separator_receipt["status"], 16) != 1:
            raise ValueError(f"neutral transfer missing {tx_hash}")
        next_hash = target["parentHash"]
        for depth in range(32):
            parent = rpc_query(rpc, "eth_getBlockByHash", [next_hash, False])
            if parent["hash"] == separator_receipt["blockHash"]:
                if parent["transactions"] != [sample["separatorTxHash"]]:
                    raise ValueError(f"separator block includes other transactions {tx_hash}")
                if depth > 0:
                    empty_parents += 1
                break
            if parent["transactions"] or parent["hash"] == genesis["hash"]:
                raise ValueError(f"non-neutral parent {tx_hash}")
            next_hash = parent["parentHash"]
        else:
            raise ValueError(f"separator too far from target {tx_hash}")
        if len(receipt["logs"]) != 1 or receipt["logs"][0]["address"].lower() != manifest["batchAddress"].lower():
            raise ValueError(f"missing batch event {tx_hash}")
        event_data = bytes.fromhex(receipt["logs"][0]["data"][2:])
        if len(event_data) != 64 or int.from_bytes(event_data[:32], "big") != sample["callsPerTx"]:
            raise ValueError(f"batch event count mismatch {tx_hash}")
        observed[sample["callsPerTx"]].add(event_data[32:])
    if len(unique_transactions) != len(samples) or any(len(digests) != 1 for digests in observed.values()):
        raise ValueError("different digest for equal call count")
    if set(observed) != set(manifest["levels"]):
        raise ValueError("missing call count")
    report = {"verifiedAt": datetime.now(timezone.utc).isoformat(), "rpc": rpc,
              "genesisHash": genesis["hash"], "verifiedTargetReceipts": len(samples),
              "verifiedUniqueTargetTransactions": len(unique_transactions),
              "emptyParentPaths": empty_parents, "verifiedBatchOutputCounts": sorted(observed),
              "note": "receipt Gas, per-block exclusivity, neutral parent ancestry and event digests checked against live chain"}
    (directory / "chain-audit.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    parser.add_argument("--rpc", default="http://127.0.0.1:8545")
    arguments = parser.parse_args()
    print(json.dumps(audit(arguments.directory, arguments.rpc), indent=2))
