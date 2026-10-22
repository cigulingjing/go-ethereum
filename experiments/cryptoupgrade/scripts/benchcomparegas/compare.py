#!/usr/bin/env python3
"""Compare bounded pricing quotes to matched single-node Solidity receipts."""

import argparse
import hashlib
import json
from pathlib import Path


ALGORITHMS = (
    "Add", "Sha256", "Blake2bSum256", "Pbkdf2Sha256",
    "Dh2048Secret", "PedersenCommit", "SchnorrVerify", "PolynomialMul",
)


def validate_input(name, values):
    repeated = "0x" + "73" * 256
    expected = {
        "Add": values.get("a") == str(2**63 - 1) and values.get("b") == str(2**63 - 1),
        "Sha256": values.get("dataHex") == repeated,
        "Blake2bSum256": values.get("dataHex") == "0x" + "73" * 32,
        "Pbkdf2Sha256": values.get("passwordHex") == "0x70617373776f7264"
        and values.get("saltHex") == "0x73616c74" and values.get("iterations") == "500"
        and values.get("keyLength") == "32",
        "Dh2048Secret": values.get("privateKeyHex") == "0x" + (2**127 + 1).to_bytes(16, "big").hex()
        and values.get("peerPublicKeyHex") == "0x" + "00" * 255 + "04",
        "SchnorrVerify": values.get("messageHex") == repeated
        and values.get("proofByteLength") == "768",
        "PolynomialMul": all(values.get(key) == value for key, value in {
            "leftLen": "4", "rightLen": "4", "modulus": "12289", "coefMax": "0",
        }.items()),
    }
    if not expected.get(name, False):
        raise ValueError(f"{name}: Solidity receipt input does not match calibrated level")


def compare(calibration_bytes, model, baselines):
    if model["calibrationSHA256"] != hashlib.sha256(calibration_bytes).hexdigest():
        raise ValueError("model calibration digest mismatch")
    calibration = json.loads(calibration_bytes)
    if [item["parentCalls"] for item in model["scenarios"]] != list(range(5)):
        raise ValueError("expected parent call range 0..4")
    baseline_by_name = {}
    chain_ids = set()
    for baseline in baselines:
        if baseline.get("experiment") != "tx-gas-comparison" or not baseline.get("referenceInputs"):
            raise ValueError("baseline was not collected with -reference-inputs")
        chain_ids.add(baseline["chainId"])
        for item in baseline["results"]:
            name = item["algorithm"]
            if name in baseline_by_name or name not in ALGORITHMS:
                raise ValueError(f"duplicate or unknown algorithm {name}")
            validate_input(name, item["input"])
            if not item["outputMatched"] or any(item[path]["status"] != 1 for path in ("wasmExecution", "contractExecution")):
                raise ValueError(f"failed or unmatched receipt: {name}")
            if not 0 < item["legacyAlgorithmGas"] <= item["wasmExecution"]["gasUsed"]:
                raise ValueError(f"invalid legacy algorithm charge: {name}")
            baseline_by_name[name] = item
    if len(chain_ids) != 1:
        raise ValueError("baselines belong to different chains")
    if set(baseline_by_name) != set(ALGORITHMS) - {"PedersenCommit"}:
        raise ValueError("expected seven matched receipts; Pedersen Solidity reference input reverts")

    rows = []
    for index, name in enumerate(ALGORITHMS):
        base = baseline_by_name.get(name)
        cells = []
        for scenario in model["scenarios"]:
            sample = scenario["samples"][index]
            if sample["algorithm"] != name or len(sample["callGas"]) != 5:
                raise ValueError(f"incomplete quote grid: {name}")
            entry = calibration["algorithms"][name]
            level = 32 if name == "Blake2bSum256" else entry["referenceValue"]
            fee = (next(point["rawProportionalGas"] for point in entry["measurements"]
                        if point["value"] == 32) if name == "Blake2bSum256" else entry["referenceBaseGas"])
            if sample["referenceValue"] != level or sample["baseGas"] != fee:
                raise ValueError(f"quote uses wrong calibrated fee: {name}")
            for call_index, quote in enumerate(sample["callGas"], start=1):
                cell = {"parentCalls": scenario["parentCalls"], "callIndex": call_index, "algorithmQuote": quote}
                if base is not None:
                    projected = base["wasmExecution"]["gasUsed"] - base["legacyAlgorithmGas"] + quote
                    cell["projectedStandaloneGas"] = projected
                    cell["differenceVsSolidity"] = projected - base["contractExecution"]["gasUsed"]
                    cell["fitsStandaloneBlockGasLimit"] = projected <= model["blockGasLimit"]
                cells.append(cell)
        row = {"algorithm": name, "metric": calibration["algorithms"][name]["metric"],
               "inputLevel": 32 if name == "Blake2bSum256" else calibration["algorithms"][name]["referenceValue"],
               "baseFee": cells[0]["algorithmQuote"], "grid": cells}
        if base is not None:
            row.update({"solidityReceiptGas": base["contractExecution"]["gasUsed"],
                        "legacyWasmReceiptGas": base["wasmExecution"]["gasUsed"],
                        "legacyAlgorithmGas": base["legacyAlgorithmGas"],
                        "solidityTxHash": base["contractExecution"]["txHash"],
                        "wasmTxHash": base["wasmExecution"]["txHash"],
                        "input": base["input"],
                        "cheaperGridPoints": sum(cell["differenceVsSolidity"] < 0 for cell in cells),
                        "inputEncodingComparable": name != "Dh2048Secret"})
            if name == "Dh2048Secret":
                row["inputCaveat"] = "Solidity requires 256-byte left-padded peer public key; calibration used single byte 0x04 (same integer)."
        else:
            row["unavailable"] = "Solidity PedersenCommit reverts for both 128-bit and 16-bit calibrated scalars; no matched receipt."
        rows.append(row)
    return {"kind": "offline counterfactual pricing comparison; Solidity and legacy WASM are real receipts",
            "chainId": chain_ids.pop(), "calibrationSHA256": model["calibrationSHA256"],
            "model": {key: model[key] for key in ("alpha", "threshold", "loadMapping", "blockGasLimit")},
            "range": {"parentCalls": [0, 4], "sameTransactionAlgorithmCallIndex": [1, 5]},
            "method": "projected standalone = legacy WASM receipt - legacy algorithm charge + dynamic quote; i>1 is a counterfactual marginal quote, not a batched transaction receipt",
            "algorithms": rows}


def markdown(result):
    lines = ["# 动态 Gas 与 Solidity 对照（固定输入，有限范围）", "",
             "**实验性质**：Solidity 与旧版 WASM 是隔离单节点的真实 `receipt.gasUsed`；新费率尚未接入链上，比较列是固定其他开销后的**离线反事实估算**，不是新费率交易收据。",
             "", "## 范围和方法", "",
             "- 八算法报价取标定参考输入；Blake2b 对照改取已测的 32 B 档，因为 Solidity 合约拒绝 256 B。DH 公钥对 Solidity 左填充至 256 B，标定使用 1 B，逻辑整数同为 4，但 ABI 输入不完全相同。",
             "- 固定 `α=1/4`、`N₀=2`、`C∈{0,1,2,3,4}`（`m=1.0…1.4`）、`i∈{1,2,3,4,5}`（单笔交易内同算法调用序号），共 25 个报价点/算法；`i>1` 不能当作 5 笔独立交易的序号。",
             "- `G_proj(C,i) = G_WASM,legacy(receipt) − G_alg,legacy + Quote(C,i)`，与**相同算法及输入**的 `G_Solidity(receipt)` 对比。这里假设替换算法费用不改变固有 Gas、calldata 或封装费用；对 `i>1` 只是以单次调用交易封装估算边际费用，并非批处理总 Gas。",
             "- 区块 Gas 上限 30,000,000；历史 WASM 费用只是扣费差分基准，不等于新计价已启用。DH 仅为相同计算值的示意性比较。", "",
             "## 结果", "",
             "| 算法 | 输入档 | 基础费 | Solidity 实收 | 新费率估算 C=0,i=1 | 新费率估算 C=4,i=5 | 低于 Solidity / 25 |",
             "| --- | ---: | ---: | ---: | ---: | ---: | ---: |"]
    for row in result["algorithms"]:
        if "unavailable" in row:
            lines.append(f"| {row['algorithm']} | {row['inputLevel']} | {row['baseFee']:,} | — | — | — | — |")
            continue
        first, last = row["grid"][0], row["grid"][-1]
        lines.append(f"| {row['algorithm']}{'†' if not row['inputEncodingComparable'] else ''} | {row['inputLevel']} | {row['baseFee']:,} | {row['solidityReceiptGas']:,} | {first['projectedStandaloneGas']:,} | {last['projectedStandaloneGas']:,} | {row['cheaperGridPoints']}/25 |")
    lines += ["", "† DH 输入编码不同；不可作为严格同一 ABI 输入的对照。Pedersen 的已标定 128 bit 和 16 bit 输入在 Solidity 合约执行时回退，不能填造对照收据。",
              "", "每个算法 25 个逐点报价、真实交易哈希、旧费用和对照差值见同名 JSON。重复调用报价超过单次交易上限或累计超过区块上限时，交易会失败；表中的端点只是数学估算，不保证可执行。",
              "", "## 复现", "",
              "在隔离链上使用 `benchtxgas -reference-inputs -algorithms Add,Sha256`、`-algorithms Blake2bSum256,Pbkdf2Sha256`、`-algorithms Dh2048Secret`、`-algorithms SchnorrVerify,PolynomialMul` 分批采集，对应原始 JSON 文件为 `2026-10-22-solidity-{first,middle,dh,tail}.json`。Pedersen 的 128/16 bit 两档 `eth_call` 均回退，故不纳入收据比较。用 `go run ./experiments/cryptoupgrade/scripts/benchcallgas -blake32` 生成 25 点报价，再执行 `python3 experiments/cryptoupgrade/scripts/benchcomparegas/compare.py --help` 查看对照命令参数；不得用历史默认输入的收据直接替换。需要动态计价完成链上集成后，才能测量真实的新费率 `receipt.gasUsed`。", ""]
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--calibration", required=True, type=Path)
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--baseline", required=True, action="append", type=Path)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--report", required=True, type=Path)
    args = parser.parse_args()
    result = compare(args.calibration.read_bytes(), json.loads(args.model.read_text()),
                     [json.loads(path.read_text()) for path in args.baseline])
    args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    args.report.write_text(markdown(result))


if __name__ == "__main__":
    main()
