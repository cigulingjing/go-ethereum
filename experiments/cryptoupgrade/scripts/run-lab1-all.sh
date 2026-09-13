#!/usr/bin/env bash
# Lab1：逐算法对比 WASM 升级与 Solidity 合约部署的耗时与 Gas（单节点）。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
RPC="${RPC:-http://127.0.0.1:8761}"
KEY="${KEY:-0x2aedaed0ff6818dbc349b870e384504aa50dfad34116c089ba58fc28638eb7a2}"
SOLC="${SOLC:-/home/lq/.local/share/svm/solc-0.8.26}"
WASM_DIR="$ROOT/experiments/cryptoupgrade/algorithm/go/wasm"
CONTRACT_DIR="$ROOT/experiments/cryptoupgrade/algorithm/contracts/src"
OUT_DIR="$ROOT/experiments/cryptoupgrade/results/upgrade-latency/lab1-compare-all-$(date -u +%Y%m%d-%H%M%S)"
DEPLOY_HELPER="/tmp/lab1_deploy_solidity.go"

mkdir -p "$OUT_DIR"

cat >"$DEPLOY_HELPER" <<'GOEOF'
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	binPath := flag.String("bin", "", "contract init bytecode hex file")
	rpcURL := flag.String("rpc", "http://127.0.0.1:8761", "rpc url")
	keyHex := flag.String("key", "", "private key hex")
	chainID := flag.Int64("chain-id", 11223344, "chain id")
	flag.Parse()
	if *binPath == "" || *keyHex == "" {
		fmt.Fprintln(os.Stderr, "usage: deploy -bin <file> -key <hex>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*binPath)
	if err != nil {
		fatal(err)
	}
	initCode, err := hex.DecodeString(string(raw))
	if err != nil {
		fatal(err)
	}
	key, err := crypto.HexToECDSA(trim0x(*keyHex))
	if err != nil {
		fatal(err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, *rpcURL)
	if err != nil {
		fatal(err)
	}
	defer client.Close()

	start := time.Now()
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		fatal(err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		fatal(err)
	}
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: from, Data: initCode})
	if err != nil {
		gasLimit = 8_000_000
	} else {
		gasLimit = gasLimit * 12 / 10
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce: nonce, Gas: gasLimit, GasPrice: gasPrice, Data: initCode,
	})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(*chainID)), key)
	if err != nil {
		fatal(err)
	}
	submitMs := time.Since(start).Milliseconds()
	if err := client.SendTransaction(ctx, signed); err != nil {
		fatal(err)
	}
	receiptStart := time.Now()
	receipt, err := waitReceipt(ctx, client, signed.Hash())
	if err != nil {
		fatal(err)
	}
	receiptMs := time.Since(receiptStart).Milliseconds()
	totalMs := time.Since(start).Milliseconds()
	if receipt.Status != types.ReceiptStatusSuccessful {
		fatal(fmt.Errorf("deploy reverted"))
	}
	fmt.Printf("txHash=%s\n", signed.Hash().Hex())
	fmt.Printf("contractAddress=%s\n", receipt.ContractAddress.Hex())
	fmt.Printf("gasUsed=%d\n", receipt.GasUsed)
	fmt.Printf("submitMs=%d\n", submitMs)
	fmt.Printf("receiptWaitMs=%d\n", receiptMs)
	fmt.Printf("totalMs=%d\n", totalMs)
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil && receipt != nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return nil, err
			}
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func trim0x(s string) string {
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		return s[2:]
	}
	return s
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
GOEOF

echo "round,algorithm,method,gasUsed,totalMs,submitToReceiptMs,receiptToActiveMs,txHash,notes" >"$OUT_DIR/summary.csv"

run_wasm() {
  local algo="$1" name="$2" wasm="$3" itype="$4" otype="$5" gas="$6"
  local uplog="$OUT_DIR/${algo}-wasm.log"
  local start end elapsed gasUsed txHash receiptMs activeMs
  start=$(date +%s%3N)
  if go run "$ROOT/experiments/cryptoupgrade/cmd/upload-wasm/main.go" \
    -wasm "$wasm" -name "$name" -itype "$itype" -otype "$otype" -algo-gas "$gas" \
    -key "$KEY" -rpc "$RPC" -output-json "$OUT_DIR/${algo}-wasm.json" \
    >"$uplog" 2>&1; then
    end=$(date +%s%3N)
    elapsed=$((end - start))
    gasUsed=$(grep -oP 'gasUsed=\K[0-9]+' "$uplog" | tail -1 || echo 0)
    txHash=$(grep -oP 'tx \K0x[0-9a-fA-F]+' "$uplog" | head -1 || echo "")
    receiptMs=$(grep -oP 'submitToReceiptMs=\K[0-9]+' "$uplog" | tail -1 || echo "")
    activeMs=$(grep -oP 'receiptToActiveMs=\K[0-9]+' "$uplog" | tail -1 || echo "")
    echo "${CURRENT_ROUND:-1},$algo,wasm,$gasUsed,$elapsed,${receiptMs:-0},${activeMs:-0},$txHash,ok" >>"$OUT_DIR/summary.csv"
    echo "  WASM $algo: gas=$gasUsed submitToReceipt=${receiptMs}ms receiptToActive=${activeMs}ms wall=${elapsed}ms"
  else
    end=$(date +%s%3N)
    elapsed=$((end - start))
    echo "${CURRENT_ROUND:-1},$algo,wasm,0,$elapsed,0,0,,failed" >>"$OUT_DIR/summary.csv"
    echo "  WASM $algo: FAILED (see $uplog)"
    tail -3 "$uplog" >&2 || true
  fi
}

run_solidity() {
  local algo="$1" src="$2" contract="$3"
  local binfile="$OUT_DIR/${algo}-deploy.bin"
  local sollog="$OUT_DIR/${algo}-solidity.log"
  local compilelog="$OUT_DIR/${algo}-solidity-compile.log"
  if ! "$SOLC" --optimize --via-ir --combined-json bin "$src" >"$compilelog" 2>&1; then
    echo "${CURRENT_ROUND:-1},$algo,solidity,0,0,0,0,,compile_failed" >>"$OUT_DIR/summary.csv"
    echo "  Solidity $algo: COMPILE FAILED"
    tail -5 "$compilelog" >&2 || true
    return
  fi
  python3 - "$compilelog" "$contract" "$binfile" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
name = sys.argv[2]
out = sys.argv[3]
for k, v in data.get("contracts", {}).items():
    if k.endswith(":" + name):
        open(out, "w").write(v["bin"])
        sys.exit(0)
sys.exit(1)
PY
  if [ ! -s "$binfile" ]; then
    echo "${CURRENT_ROUND:-1},$algo,solidity,0,0,0,0,,bin_missing" >>"$OUT_DIR/summary.csv"
    echo "  Solidity $algo: bytecode not found"
    return
  fi
  local start end
  start=$(date +%s%3N)
  if go run "$DEPLOY_HELPER" -bin "$binfile" -key "$KEY" -rpc "$RPC" >"$sollog" 2>&1; then
    end=$(date +%s%3N)
    local gasUsed totalMs txHash receiptMs
    gasUsed=$(grep -oP 'gasUsed=\K[0-9]+' "$sollog" | tail -1)
    totalMs=$(grep -oP 'totalMs=\K[0-9]+' "$sollog" | tail -1)
    receiptMs=$(grep -oP 'receiptWaitMs=\K[0-9]+' "$sollog" | tail -1)
    txHash=$(grep -oP 'txHash=\K0x[0-9a-fA-F]+' "$sollog" | tail -1)
    echo "${CURRENT_ROUND:-1},$algo,solidity,$gasUsed,$totalMs,${receiptMs:-0},0,$txHash,ok" >>"$OUT_DIR/summary.csv"
    echo "  Solidity $algo: gas=$gasUsed submitToReceipt=${receiptMs}ms total=${totalMs}ms"
  else
    end=$(date +%s%3N)
    echo "${CURRENT_ROUND:-1},$algo,solidity,0,$((end - start)),0,0,,deploy_failed" >>"$OUT_DIR/summary.csv"
    echo "  Solidity $algo: DEPLOY FAILED"
    tail -5 "$sollog" >&2 || true
  fi
}

ROUNDS="${ROUNDS:-3}"
NAME_SUFFIX="${NAME_SUFFIX:-$(date +%H%M%S)}"

echo "Lab1 all-algorithms comparison -> $OUT_DIR"
echo "RPC=$RPC rounds=$ROUNDS suffix=$NAME_SUFFIX"

for round in $(seq 1 "$ROUNDS"); do
  CURRENT_ROUND="$round"
  echo ""
  echo "===== round $round/$ROUNDS ====="
  echo "[Add] round=$round"
  run_wasm Add "AddLab1${NAME_SUFFIX}r${round}" "$WASM_DIR/add.wasm" "int256,int256" int256 3000
  run_solidity Add "$CONTRACT_DIR/archive/Add.sol" AddContract

  echo "[Sha256] round=$round"
  run_wasm Sha256 "Sha256Lab1${NAME_SUFFIX}r${round}" "$WASM_DIR/sha256.wasm" bytes bytes 3000
  run_solidity Sha256 "$CONTRACT_DIR/archive/Sha256.sol" Sha256Contract

  echo "[Blake2bSum256] round=$round"
  run_wasm Blake2bSum256 "Sum256Lab1${NAME_SUFFIX}r${round}" "$WASM_DIR/blake2b.wasm" bytes bytes32 3000
  run_solidity Blake2bSum256 "$CONTRACT_DIR/archive/Blake2b.sol" Blake2b

  echo "[Pbkdf2Sha256] round=$round"
  run_wasm Pbkdf2Sha256 "Pbkdf2Sha256Lab1${NAME_SUFFIX}r${round}" "$WASM_DIR/pbkdf2_sha256.wasm" "bytes,bytes,uint256,uint256" bytes 25000
  run_solidity Pbkdf2Sha256 "$CONTRACT_DIR/archive/Pbkdf2Sha256.sol" Pbkdf2Sha256Contract

  echo "[Dh2048Secret] round=$round"
  run_wasm Dh2048Secret "Dh2048SecretLab1${NAME_SUFFIX}r${round}" "$WASM_DIR/dh2048.wasm" "bytes,bytes" bytes 200000
  run_solidity Dh2048Secret "$CONTRACT_DIR/archive/Dh2048.sol" Dh2048

  echo "[PedersenCommit] round=$round"
  run_wasm PedersenCommit "PedersenCommitLab1${NAME_SUFFIX}r${round}" "$WASM_DIR/pedersen_commit.wasm" "bytes,bytes" bytes 200000
  run_solidity PedersenCommit "$CONTRACT_DIR/archive/PedersenCommit.sol" PedersenCommitContract

  echo "[SchnorrVerify] round=$round"
  run_wasm SchnorrVerify "SchnorrVerifyLab1${NAME_SUFFIX}r${round}" "$WASM_DIR/schnorr_proof.wasm" "bytes,bytes" bool 200000
  run_solidity SchnorrVerify "$CONTRACT_DIR/archive/SchnorrProof.sol" SchnorrProof

  echo "[PolynomialMul] round=$round"
  run_wasm PolynomialMul "PolynomialMulLab1${NAME_SUFFIX}r${round}" "$WASM_DIR/polynomial_mul.wasm" "uint256[],uint256[],uint256" "uint256[]" 50000
  run_solidity PolynomialMul "$CONTRACT_DIR/PolynomialMul.sol" PolynomialMulContract
done

echo ""
echo "=== Summary ==="
column -t -s, "$OUT_DIR/summary.csv" 2>/dev/null || cat "$OUT_DIR/summary.csv"
echo ""
echo "Results saved to $OUT_DIR"
