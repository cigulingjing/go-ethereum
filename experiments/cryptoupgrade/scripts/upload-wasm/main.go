// upload-wasm 向本地 CodeStorage 发送 WASM 升级交易。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/ethclient"
)

type result struct {
	TxHash      string `json:"txHash"`
	BlockNumber uint64 `json:"blockNumber,omitempty"`
	GasUsed     uint64 `json:"gasUsed,omitempty"`
	Status      uint64 `json:"status,omitempty"`
	Algorithm   string `json:"algorithm"`
	Mode        string `json:"mode"`
	RPC         string `json:"rpc"`
}

func main() {
	var (
		wasmPath        = flag.String("wasm", "", "WASM 文件路径")
		algorithm       = flag.String("name", "", "算法名称（默认从文件名推断）")
		mode            = flag.String("mode", "upload", "upload | immediate | version")
		version         = flag.Uint64("version", 1, "版本号（immediate/version 模式）")
		activationBlock = flag.Uint64("activation-block", 0, "生效高度（version 模式）")
		algoGas         = flag.Uint64("algo-gas", 7, "算法 Gas 配置")
		inputTypes      = flag.String("itype", "int256,int256", "输入类型")
		outputTypes     = flag.String("otype", "int256", "输出类型")
		rpcURL          = flag.String("rpc", "http://127.0.0.1:8761", "HTTP RPC URL")
		privateKey      = flag.String("key", "", "签名私钥（32 字节 hex，可带 0x 前缀）")
		chainIDFlag     = flag.Uint64("chain-id", 11223344, "链 ID")
		txGas           = flag.Uint64("tx-gas", 0, "交易 Gas 上限（0 表示 estimate）")
		waitTimeout     = flag.Duration("wait", 2*time.Minute, "等待 receipt 超时")
		waitActive      = flag.Bool("wait-active", true, "upload 模式完成后轮询 getActiveVersion")
		jsonOut         = flag.String("output-json", "", "可选结果 JSON 路径")
	)
	flag.Parse()
	if *wasmPath == "" {
		fmt.Fprintln(os.Stderr, "usage: upload-wasm -wasm <file.wasm> -key <hex> [-name Add] [-rpc URL]")
		os.Exit(2)
	}
	name := strings.TrimSpace(*algorithm)
	if name == "" {
		name = inferAlgorithmName(*wasmPath)
	}
	keyHex, err := parsePrivateKeyHex(*privateKey)
	if err != nil {
		fatal(err)
	}
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		fatal(fmt.Errorf("parse private key: %w", err))
	}
	from := crypto.PubkeyToAddress(key.PublicKey)

	encoded, err := cryptoupgrade.EncodeWasmFile(*wasmPath)
	if err != nil {
		fatal(fmt.Errorf("encode wasm: %w", err))
	}
	data, err := packUpload(*mode, name, encoded, *version, *activationBlock, *algoGas, *inputTypes, *outputTypes)
	if err != nil {
		fatal(err)
	}

	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, *rpcURL)
	if err != nil {
		fatal(fmt.Errorf("dial rpc: %w", err))
	}
	defer client.Close()

	chainID := big.NewInt(int64(*chainIDFlag))
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		fatal(fmt.Errorf("pending nonce: %w", err))
	}
	to := common.CodeStorageAddress
	callMsg := ethereum.CallMsg{From: from, To: &to, Data: data}
	gasLimit := *txGas
	if gasLimit == 0 {
		gasLimit, err = client.EstimateGas(ctx, callMsg)
		if err != nil {
			gasLimit = cryptoupgradeRequiredGas(data)
		} else {
			gasLimit = gasLimit * 12 / 10
		}
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		fatal(fmt.Errorf("suggest gas price: %w", err))
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		fatal(fmt.Errorf("sign tx: %w", err))
	}
	sentAt := time.Now()
	if err := client.SendTransaction(ctx, signed); err != nil {
		fatal(fmt.Errorf("send tx: %w", err))
	}

	out := result{
		TxHash:    signed.Hash().Hex(),
		Algorithm: name,
		Mode:      *mode,
		RPC:       *rpcURL,
	}
	fmt.Printf("submitted %s tx %s\n", *mode, out.TxHash)

	waitCtx, cancel := context.WithTimeout(ctx, *waitTimeout)
	defer cancel()
	receipt, err := waitReceipt(waitCtx, client, signed.Hash())
	if err != nil {
		fatal(fmt.Errorf("wait receipt: %w", err))
	}
	receiptAt := time.Now()
	out.BlockNumber = receipt.BlockNumber.Uint64()
	out.GasUsed = receipt.GasUsed
	out.Status = receipt.Status
	fmt.Printf("mined in block %d status=%d gasUsed=%d\n", out.BlockNumber, out.Status, out.GasUsed)
	fmt.Printf("submitToReceiptMs=%d\n", receiptAt.Sub(sentAt).Milliseconds())
	if receipt.Status != types.ReceiptStatusSuccessful {
		fatal(fmt.Errorf("transaction reverted"))
	}

	if *waitActive && *mode == "upload" {
		if err := waitAlgorithmActive(ctx, client, name, *version); err != nil {
			fatal(err)
		}
		activeAt := time.Now()
		fmt.Printf("algorithm %s is active\n", name)
		fmt.Printf("receiptToActiveMs=%d\n", activeAt.Sub(receiptAt).Milliseconds())
		fmt.Printf("submitToActiveMs=%d\n", activeAt.Sub(sentAt).Milliseconds())
	}

	if *jsonOut != "" {
		raw, _ := json.MarshalIndent(out, "", "  ")
		if err := os.WriteFile(*jsonOut, append(raw, '\n'), 0644); err != nil {
			fatal(err)
		}
	}
}

func packUpload(mode, name, encoded string, version, activationBlock, algoGas uint64, itype, otype string) ([]byte, error) {
	switch strings.ToLower(mode) {
	case "upload":
		return cryptoupgrade.CodeStorageABI.Pack("uploadCode", name, encoded, algoGas, itype, otype)
	case "immediate":
		return cryptoupgrade.CodeStorageABI.Pack("uploadCodeImmediate", name, version, encoded, algoGas, itype, otype)
	case "version":
		if activationBlock == 0 {
			return nil, fmt.Errorf("version mode requires -activation-block")
		}
		return cryptoupgrade.CodeStorageABI.Pack("uploadCodeVersion", name, version, encoded, algoGas, itype, otype, activationBlock)
	default:
		return nil, fmt.Errorf("unsupported mode %q", mode)
	}
}

func inferAlgorithmName(path string) string {
	base := path
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		base = path[i+1:]
	}
	base = strings.TrimSuffix(base, ".wasm")
	parts := strings.Split(base, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func parsePrivateKeyHex(raw string) (string, error) {
	hexKey := strings.TrimSpace(strings.TrimPrefix(raw, "0x"))
	if hexKey == "" {
		return "", fmt.Errorf("missing -key: expected 32-byte hex private key")
	}
	if len(hexKey) != 64 {
		return "", fmt.Errorf("invalid -key: expected 32-byte hex private key, got %d hex chars", len(hexKey))
	}
	for _, r := range hexKey {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return "", fmt.Errorf("invalid -key: private key must be hex")
		}
	}
	return hexKey, nil
}

func cryptoupgradeRequiredGas(input []byte) uint64 {
	// CodeStorage 升级路径不再单独定价；estimate 失败时使用保守 fallback。
	_ = input
	return 8_000_000
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	// bind.WaitMined 固定 1s 轮询，period=0 时会把真实出块时间淹没掉。
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

func waitAlgorithmActive(ctx context.Context, client *ethclient.Client, name string, version uint64) error {
	data, err := cryptoupgrade.CodeStorageABI.Pack("getActiveVersion", name)
	if err != nil {
		return err
	}
	to := common.CodeStorageAddress
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		out, err := client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
		if err == nil && len(out) > 0 {
			values, err := cryptoupgrade.CodeStorageABI.Unpack("getActiveVersion", out)
			if err == nil && len(values) > 0 {
				if active, ok := values[0].(uint64); ok && active >= version {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for %s activation", name)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
