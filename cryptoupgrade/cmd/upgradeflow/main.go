package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type txInfo struct {
	BlockNumber *hexutil.Uint64 `json:"blockNumber"`
	Gas         hexutil.Uint64  `json:"gas"`
}

func main() {
	var (
		rpcURL  = flag.String("rpc", "http://127.0.0.1:8666", "execution RPC endpoint")
		source  = flag.String("source", "cryptoupgrade/algorithm/go/add.go", "algorithm source file")
		name    = flag.String("name", "Add", "algorithm name")
		aText   = flag.String("a", "100", "first int256 argument")
		bText   = flag.String("b", "100", "second int256 argument")
		algoGas = flag.Uint64("algo-gas", 1, "algorithm gas recorded in CodeStorage")
		txGas   = flag.Uint64("tx-gas", 0, "upload transaction gas limit; 0 means estimate via eth_estimateGas")
		from    = flag.String("from", "", "sender address; defaults to eth_accounts[0]")
	)
	flag.Parse()

	if err := run(*rpcURL, *source, *name, *aText, *bText, *algoGas, *txGas, *from); err != nil {
		fmt.Fprintf(os.Stderr, "upgrade flow failed: %v\n", err)
		os.Exit(1)
	}
}

func run(rpcURL, source, name, aText, bText string, algoGas, txGas uint64, fromText string) error {
	ctx := context.Background()
	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer client.Close()

	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return err
	}
	int256Type, err := abi.NewType("int256", "", nil)
	if err != nil {
		return err
	}
	int256Args := abi.Arguments{{Type: int256Type}, {Type: int256Type}}
	int256Return := abi.Arguments{{Type: int256Type}}

	fromAddr, err := sender(ctx, client, fromText)
	if err != nil {
		return err
	}

	compressed, err := compressFile(source)
	if err != nil {
		return err
	}

	fmt.Printf("sender: %s\n", fromAddr.Hex())
	fmt.Printf("algorithm: %s\n", name)
	fmt.Printf("source: %s\n", mustAbs(source))
	fmt.Printf("compressed-bytes(base64): %d\n", len(compressed))

	uploadData, err := codeStorageABI.Pack("uploadCode", name, compressed, algoGas, "int256,int256", "int256")
	if err != nil {
		return err
	}
	uploadArgs := txArgs{
		From: fromAddr,
		To:   &common.CodeStorageAddress,
		Data: uploadData,
	}

	gasLimit, err := resolveGasLimit(ctx, client, uploadArgs, txGas)
	if err != nil {
		return fmt.Errorf("upload preflight failed: %w", wrapPluginCompileHint(err))
	}
	gas := hexutil.Uint64(gasLimit)
	uploadArgs.Gas = &gas
	fmt.Printf("upload-gas-limit: %d\n", gasLimit)

	txHash, err := sendTransaction(ctx, client, uploadArgs)
	if err != nil {
		return err
	}
	fmt.Printf("upload-tx: %s\n", txHash.Hex())

	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return err
	}
	fmt.Printf("upload-status: %d\n", receipt.Status)
	fmt.Printf("upload-block: %d\n", receipt.BlockNumber.Uint64())
	fmt.Printf("upload-gas-used: %d\n", receipt.GasUsed)
	fmt.Printf("upload-logs: %d\n", len(receipt.Logs))
	if receipt.Status != types.ReceiptStatusSuccessful {
		return describeFailedReceipt(ctx, client, txHash, receipt, uploadArgs)
	}

	infoData, err := codeStorageABI.Pack("getInfo", name)
	if err != nil {
		return err
	}
	infoRaw, err := ethCall(ctx, client, fromAddr, common.CodeStorageAddress, infoData)
	if err != nil {
		return err
	}
	info, err := codeStorageABI.Unpack("getInfo", infoRaw)
	if err != nil {
		return err
	}
	fmt.Printf("chain-info: gas=%d itype=%q otype=%q code-bytes(base64)=%d\n", info[1].(uint64), info[2].(string), info[3].(string), len(info[0].(string)))

	a, ok := new(big.Int).SetString(aText, 10)
	if !ok {
		return fmt.Errorf("invalid a argument %q", aText)
	}
	b, ok := new(big.Int).SetString(bText, 10)
	if !ok {
		return fmt.Errorf("invalid b argument %q", bText)
	}
	encodedArgs, err := int256Args.Pack(a, b)
	if err != nil {
		return err
	}
	callData, err := codeStorageABI.Pack("callFunc", name, encodedArgs)
	if err != nil {
		return err
	}
	rawResult, err := ethCall(ctx, client, fromAddr, common.CodeStorageAddress, callData)
	if err != nil {
		return err
	}
	decoded, err := int256Return.Unpack(rawResult)
	if err != nil {
		return err
	}
	fmt.Printf("call-args: %s + %s\n", a.String(), b.String())
	fmt.Printf("call-raw: 0x%x\n", []byte(rawResult))
	fmt.Printf("call-result: %s\n", decoded[0].(*big.Int).String())
	return nil
}

func sender(ctx context.Context, client *rpc.Client, fromText string) (common.Address, error) {
	if fromText != "" {
		if !common.IsHexAddress(fromText) {
			return common.Address{}, fmt.Errorf("invalid sender address %q", fromText)
		}
		return common.HexToAddress(fromText), nil
	}
	var accounts []common.Address
	if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
		return common.Address{}, err
	}
	if len(accounts) == 0 {
		return common.Address{}, fmt.Errorf("eth_accounts returned no unlocked accounts")
	}
	return accounts[0], nil
}

func wrapPluginCompileHint(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "executable file not found") {
		return fmt.Errorf("%w (restart geth with go in PATH, e.g. export PATH=/usr/local/go/bin:$PATH)", err)
	}
	return err
}

func resolveGasLimit(ctx context.Context, client *rpc.Client, args txArgs, txGas uint64) (uint64, error) {
	estimated, err := estimateGas(ctx, client, args)
	if err != nil {
		if txGas == 0 {
			return 0, err
		}
		fmt.Fprintf(os.Stderr, "warning: eth_estimateGas failed (%v), using -tx-gas=%d\n", err, txGas)
		return txGas, nil
	}
	limit := estimated + estimated/5 // 20% headroom
	if txGas > limit {
		limit = txGas
	}
	return limit, nil
}

func estimateGas(ctx context.Context, client *rpc.Client, args txArgs) (uint64, error) {
	var gas hexutil.Uint64
	if err := client.CallContext(ctx, &gas, "eth_estimateGas", args); err != nil {
		return 0, err
	}
	return uint64(gas), nil
}

func sendTransaction(ctx context.Context, client *rpc.Client, args txArgs) (common.Hash, error) {
	var txHash common.Hash
	if err := client.CallContext(ctx, &txHash, "eth_sendTransaction", args); err != nil {
		return common.Hash{}, err
	}
	return txHash, nil
}

func waitReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		receipt, err := getReceipt(ctx, client, txHash)
		if err != nil {
			if isTxIndexingInProgress(err) {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return nil, err
		}
		if receipt != nil {
			return receipt, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("timed out waiting for receipt %s", txHash.Hex())
}

func getReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
		return nil, err
	}
	return receipt, nil
}

func isTxIndexingInProgress(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "transaction indexing is in progress") {
		return true
	}
	var rpcErr rpc.Error
	if errors.As(err, &rpcErr) && strings.Contains(rpcErr.Error(), "transaction indexing is in progress") {
		return true
	}
	return false
}

func describeFailedReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash, receipt *types.Receipt, args txArgs) error {
	var tx txInfo
	if err := client.CallContext(ctx, &tx, "eth_getTransactionByHash", txHash); err != nil {
		return fmt.Errorf("upload transaction failed in block %d (status=0, gasUsed=%d): %w",
			receipt.BlockNumber.Uint64(), receipt.GasUsed, err)
	}

	msg := fmt.Sprintf("upload transaction failed in block %d (status=0, gasUsed=%d, gasLimit=%d",
		receipt.BlockNumber.Uint64(), receipt.GasUsed, uint64(tx.Gas))
	if tx.Gas > 0 && receipt.GasUsed >= uint64(tx.Gas) {
		msg += ", likely out of gas"
	}
	msg += ")"

	if _, err := ethCall(ctx, client, args.From, *args.To, args.Data); err != nil {
		msg += fmt.Sprintf("; replay via eth_call: %v", wrapPluginCompileHint(err))
	}
	return errors.New(msg)
}

func ethCall(ctx context.Context, client *rpc.Client, from common.Address, to common.Address, data []byte) (hexutil.Bytes, error) {
	gas := hexutil.Uint64(5000000)
	var out hexutil.Bytes
	if err := client.CallContext(ctx, &out, "eth_call", txArgs{
		From: from,
		To:   &to,
		Gas:  &gas,
		Data: data,
	}, "latest"); err != nil {
		return nil, err
	}
	return out, nil
}

func compressFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := io.Copy(gw, file); err != nil {
		return "", err
	}
	if err := gw.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
