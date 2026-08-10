// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const codeStorageABIJSON = `[
	{"type":"function","name":"uploadCode","inputs":[{"name":"name","type":"string"},{"name":"base64Code","type":"string"},{"name":"gasLimit","type":"uint64"},{"name":"inputType","type":"string"},{"name":"outputType","type":"string"}],"outputs":[]},
	{"type":"function","name":"callFunc","inputs":[{"name":"name","type":"string"},{"name":"inputs","type":"bytes"}],"outputs":[{"name":"","type":"bytes"}],"stateMutability":"nonpayable"}
]`

type config struct {
	rpc              string
	mode             string
	algorithm        string
	sourcePath       string
	inputTypes       string
	outputTypes      string
	inputHex         string
	expectedHex      string
	precompileAddr   string
	from             string
	warmup           int
	samples          int
	outputJSON       string
	upload           bool
	algoGas          uint64
	upgradeDeployGas uint64
}

type callArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type setupMetrics struct {
	Scheme          string  `json:"scheme"`
	TxCount         int     `json:"txCount"`
	TxHash          string  `json:"txHash,omitempty"`
	ReceiptStatus   uint64  `json:"receiptStatus"`
	ElapsedMillis   float64 `json:"elapsedMillis"`
	GasUsed         uint64  `json:"gasUsed"`
	SourceBytes     int     `json:"sourceBytes,omitempty"`
	CompressedBytes int     `json:"compressedBytes,omitempty"`
	Address         string  `json:"address,omitempty"`
}

type callMetrics struct {
	Scheme           string  `json:"scheme"`
	Warmup           int     `json:"warmup"`
	Samples          int     `json:"samples"`
	GasEstimate      uint64  `json:"gasEstimate"`
	FirstMillis      float64 `json:"firstMillis"`
	MeanMillis       float64 `json:"meanMillis"`
	P50Millis        float64 `json:"p50Millis"`
	P95Millis        float64 `json:"p95Millis"`
	MinMillis        float64 `json:"minMillis"`
	MaxMillis        float64 `json:"maxMillis"`
	OutputHex        string  `json:"outputHex,omitempty"`
	OutputSHA256     string  `json:"outputSha256,omitempty"`
	OutputByteLength int     `json:"outputByteLength,omitempty"`
}

type ratioMetrics struct {
	MeanTime           float64 `json:"meanTimeUpgradeOverPrecompile"`
	P50Time            float64 `json:"p50TimeUpgradeOverPrecompile"`
	P95Time            float64 `json:"p95TimeUpgradeOverPrecompile"`
	Gas                float64 `json:"gasUpgradeOverPrecompile"`
	UpgradeSetupGas    uint64  `json:"upgradeSetupGas"`
	PrecompileSetupGas uint64  `json:"precompileSetupGas"`
}

type result struct {
	Timestamp         string       `json:"timestamp"`
	RPC               string       `json:"rpc"`
	Mode              string       `json:"mode"`
	Sender            string       `json:"sender"`
	Algorithm         string       `json:"algorithm"`
	SourcePath        string       `json:"sourcePath"`
	InputTypes        []string     `json:"inputTypes"`
	OutputTypes       []string     `json:"outputTypes"`
	InputHex          string       `json:"inputHex"`
	ExpectedHex       string       `json:"expectedHex,omitempty"`
	PrecompileAddress string       `json:"precompileAddress"`
	Command           []string     `json:"command"`
	UpgradeSetup      setupMetrics `json:"upgradeSetup"`
	PrecompileSetup   setupMetrics `json:"precompileSetup"`
	UpgradeCall       callMetrics  `json:"upgradeCall,omitempty"`
	PrecompileCall    callMetrics  `json:"precompileCall,omitempty"`
	Ratios            ratioMetrics `json:"ratios,omitempty"`
	OutputMatched     bool         `json:"outputMatched"`
}

type benchmarkStats struct {
	first  time.Duration
	mean   time.Duration
	p50    time.Duration
	p95    time.Duration
	min    time.Duration
	max    time.Duration
	output []byte
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchrealchain: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		return err
	}
	inputData, err := parseHex(cfg.inputHex)
	if err != nil {
		return fmt.Errorf("invalid -input-hex: %w", err)
	}
	expectedOutput, err := optionalHex(cfg.expectedHex)
	if err != nil {
		return fmt.Errorf("invalid -expected-hex: %w", err)
	}
	inputTypes, err := parseABITypeList(cfg.inputTypes)
	if err != nil {
		return fmt.Errorf("invalid -itype: %w", err)
	}
	outputTypes, err := parseABITypeList(cfg.outputTypes)
	if err != nil {
		return fmt.Errorf("invalid -otype: %w", err)
	}
	precompileAddress := common.HexToAddress(cfg.precompileAddr)
	codeStorageABI, err := abi.JSON(strings.NewReader(codeStorageABIJSON))
	if err != nil {
		return err
	}
	callData, err := codeStorageABI.Pack("callFunc", cfg.algorithm, inputData)
	if err != nil {
		return fmt.Errorf("pack CodeStorage.callFunc: %w", err)
	}
	ctx := context.Background()
	client, err := rpc.DialContext(ctx, cfg.rpc)
	if err != nil {
		return fmt.Errorf("dial %s: %w", cfg.rpc, err)
	}
	defer client.Close()

	from, err := resolveSender(ctx, client, cfg.from)
	if err != nil {
		return err
	}
	res := result{
		Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
		RPC:               cfg.rpc,
		Mode:              cfg.mode,
		Sender:            from.Hex(),
		Algorithm:         cfg.algorithm,
		SourcePath:        cfg.sourcePath,
		InputTypes:        typeNames(inputTypes),
		OutputTypes:       typeNames(outputTypes),
		InputHex:          hexutil.Encode(inputData),
		ExpectedHex:       cfg.expectedHex,
		PrecompileAddress: precompileAddress.Hex(),
		Command:           os.Args,
		PrecompileSetup: setupMetrics{
			Scheme:  "precompile",
			TxCount: 0,
			GasUsed: 0,
			Address: precompileAddress.Hex(),
		},
	}

	if cfg.upload {
		setup, err := uploadAlgorithm(ctx, client, codeStorageABI, from, cfg)
		if err != nil {
			return err
		}
		res.UpgradeSetup = setup
	} else {
		res.UpgradeSetup = setupMetrics{Scheme: "upgrade", TxCount: 0}
	}

	if cfg.mode == "setup" {
		res.OutputMatched = true
		printSummary(res)
		return writeJSONResult(cfg.outputJSON, res)
	}

	upgradeTo := common.CodeStorageAddress
	precompileTo := precompileAddress
	upgradeGas, err := estimateGas(ctx, client, callArgs{From: from, To: &upgradeTo, Data: callData})
	if err != nil {
		return fmt.Errorf("estimate upgrade call gas: %w", wrapPluginCompileHint(err))
	}
	precompileGas, err := estimateGas(ctx, client, callArgs{From: from, To: &precompileTo, Data: inputData})
	if err != nil {
		return fmt.Errorf("estimate precompile call gas: %w", err)
	}
	upgradeStats, err := benchmark(cfg.warmup, cfg.samples, func() ([]byte, error) {
		return ethCall(ctx, client, callArgs{From: from, To: &upgradeTo, Data: callData})
	})
	if err != nil {
		return fmt.Errorf("benchmark upgrade path: %w", wrapPluginCompileHint(err))
	}
	precompileStats, err := benchmark(cfg.warmup, cfg.samples, func() ([]byte, error) {
		return ethCall(ctx, client, callArgs{From: from, To: &precompileTo, Data: inputData})
	})
	if err != nil {
		return fmt.Errorf("benchmark precompile path: %w", err)
	}
	// CodeStorage.callFunc 在 EVM special-case 中直接返回算法的 ABI 输出，而不是再次按 callFunc 的 bytes 返回值包装。
	// 因此这里与 precompile 比较同一层的算法 ABI 输出，避免把 bytes 输出误解包成逻辑值后再比较。
	upgradeOutput := upgradeStats.output
	precompileOutput := precompileStats.output
	if expectedOutput != nil {
		if !bytes.Equal(upgradeOutput, expectedOutput) {
			return fmt.Errorf("upgrade output mismatch: got %s want %s", hexutil.Encode(upgradeOutput), hexutil.Encode(expectedOutput))
		}
		if !bytes.Equal(precompileOutput, expectedOutput) {
			return fmt.Errorf("precompile output mismatch: got %s want %s", hexutil.Encode(precompileOutput), hexutil.Encode(expectedOutput))
		}
	} else if !bytes.Equal(upgradeOutput, precompileOutput) {
		return fmt.Errorf("upgrade/precompile output mismatch: upgrade=%s precompile=%s", hexutil.Encode(upgradeOutput), hexutil.Encode(precompileOutput))
	}
	res.OutputMatched = true
	res.UpgradeCall = metricsFromStats("upgrade", cfg, upgradeGas, upgradeOutput, upgradeStats)
	res.PrecompileCall = metricsFromStats("precompile", cfg, precompileGas, precompileOutput, precompileStats)
	res.Ratios = ratioMetrics{
		MeanTime:           safeRatioDuration(upgradeStats.mean, precompileStats.mean),
		P50Time:            safeRatioDuration(upgradeStats.p50, precompileStats.p50),
		P95Time:            safeRatioDuration(upgradeStats.p95, precompileStats.p95),
		Gas:                safeRatioUint64(upgradeGas, precompileGas),
		UpgradeSetupGas:    res.UpgradeSetup.GasUsed,
		PrecompileSetupGas: res.PrecompileSetup.GasUsed,
	}
	printSummary(res)
	return writeJSONResult(cfg.outputJSON, res)
}

func parseFlags() config {
	defaultInput := hexutil.Encode(mustPackBytes([]byte("hello cryptoupgrade")))
	cfg := config{}
	flag.StringVar(&cfg.rpc, "rpc", "http://127.0.0.1:8666", "JSON-RPC endpoint")
	flag.StringVar(&cfg.mode, "mode", "all", "benchmark mode: all, setup, or call")
	flag.StringVar(&cfg.algorithm, "name", "Sha256", "algorithm name registered in CodeStorage")
	flag.StringVar(&cfg.sourcePath, "source", "cryptoupgrade/algorithm/go/sha256.go", "algorithm source file to upload")
	flag.StringVar(&cfg.inputTypes, "itype", "bytes", "comma-separated ABI input types")
	flag.StringVar(&cfg.outputTypes, "otype", "bytes", "comma-separated ABI output types")
	flag.StringVar(&cfg.inputHex, "input-hex", defaultInput, "ABI-encoded input bytes")
	flag.StringVar(&cfg.expectedHex, "expected-hex", "", "expected ABI-encoded output bytes")
	flag.StringVar(&cfg.precompileAddr, "precompile-address", common.CryptoUpgradeSha256Address.Hex(), "precompile address")
	flag.StringVar(&cfg.from, "from", "", "sender address; defaults to eth_accounts[0]")
	flag.IntVar(&cfg.warmup, "warmup", 10, "warmup eth_call count per path")
	flag.IntVar(&cfg.samples, "n", 100, "measured eth_call count per path")
	flag.StringVar(&cfg.outputJSON, "output-json", "", "write full benchmark result JSON to this file")
	flag.BoolVar(&cfg.upload, "upload", true, "upload the algorithm before measuring")
	flag.Uint64Var(&cfg.algoGas, "algo-gas", 3000, "algorithm gas limit metadata for CodeStorage.uploadCode")
	flag.Uint64Var(&cfg.upgradeDeployGas, "upgrade-deploy-gas", 8000000, "gas limit for CodeStorage.uploadCode transaction; use 0 to estimate")
	flag.Parse()
	cfg.mode = strings.ToLower(strings.TrimSpace(cfg.mode))
	return cfg
}

func (cfg config) validate() error {
	if cfg.rpc == "" {
		return errors.New("-rpc is required")
	}
	if cfg.mode != "all" && cfg.mode != "setup" && cfg.mode != "call" {
		return errors.New("-mode must be one of: all, setup, call")
	}
	if cfg.algorithm == "" {
		return errors.New("-name is required")
	}
	if cfg.precompileAddr == "" || !common.IsHexAddress(cfg.precompileAddr) {
		return errors.New("-precompile-address must be a hex address")
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return errors.New("-from must be a hex address")
	}
	if cfg.samples <= 0 {
		return errors.New("-n must be greater than zero")
	}
	if cfg.warmup < 0 {
		return errors.New("-warmup cannot be negative")
	}
	if cfg.upload {
		if cfg.sourcePath == "" {
			return errors.New("-source is required when -upload=true")
		}
		if _, err := os.Stat(cfg.sourcePath); err != nil {
			return fmt.Errorf("source file %q: %w", cfg.sourcePath, err)
		}
	}
	return nil
}

func parseABITypeList(raw string) ([]abi.ArgumentMarshaling, error) {
	parts := splitComma(raw)
	args := make([]abi.ArgumentMarshaling, len(parts))
	for i, part := range parts {
		if _, err := abi.NewType(part, "", nil); err != nil {
			return nil, fmt.Errorf("%q: %w", part, err)
		}
		args[i] = abi.ArgumentMarshaling{Type: part}
	}
	return args, nil
}

func typeNames(args []abi.ArgumentMarshaling) []string {
	names := make([]string, len(args))
	for i, arg := range args {
		names[i] = arg.Type
	}
	return names
}

func splitComma(raw string) []string {
	fields := strings.Split(raw, ",")
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func parseHex(raw string) ([]byte, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "0x"))
	if raw == "" {
		return nil, nil
	}
	if len(raw)%2 != 0 {
		raw = "0" + raw
	}
	return hex.DecodeString(raw)
}

func optionalHex(raw string) ([]byte, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	return parseHex(raw)
}

func mustPackBytes(value []byte) []byte {
	typ, err := abi.NewType("bytes", "", nil)
	if err != nil {
		panic(err)
	}
	args := abi.Arguments{{Type: typ}}
	out, err := args.Pack(value)
	if err != nil {
		panic(err)
	}
	return out
}

func resolveSender(ctx context.Context, client *rpc.Client, raw string) (common.Address, error) {
	if raw != "" {
		return common.HexToAddress(raw), nil
	}
	var accounts []common.Address
	if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
		return common.Address{}, fmt.Errorf("eth_accounts: %w", err)
	}
	if len(accounts) == 0 {
		return common.Address{}, errors.New("eth_accounts returned no senders; pass -from with an unlocked account")
	}
	return accounts[0], nil
}

func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config) (setupMetrics, error) {
	sourceBytes, payload, err := compressSource(cfg.sourcePath)
	if err != nil {
		return setupMetrics{}, err
	}
	data, err := codeStorageABI.Pack("uploadCode", cfg.algorithm, payload, cfg.algoGas, cfg.inputTypes, cfg.outputTypes)
	if err != nil {
		return setupMetrics{}, fmt.Errorf("pack CodeStorage.uploadCode: %w", err)
	}
	to := common.CodeStorageAddress
	gasLimit, err := resolveGasLimit(ctx, client, callArgs{From: from, To: &to, Data: data}, cfg.upgradeDeployGas)
	if err != nil {
		return setupMetrics{}, fmt.Errorf("resolve upload gas: %w", wrapPluginCompileHint(err))
	}
	started := time.Now()
	txHash, err := sendTransaction(ctx, client, callArgs{From: from, To: &to, Gas: &gasLimit, Data: data})
	if err != nil {
		return setupMetrics{}, fmt.Errorf("send upload transaction: %w", wrapPluginCompileHint(err))
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return setupMetrics{}, err
	}
	metrics := setupMetrics{
		Scheme:          "upgrade",
		TxCount:         1,
		TxHash:          txHash.Hex(),
		ReceiptStatus:   receipt.Status,
		ElapsedMillis:   millis(time.Since(started)),
		GasUsed:         receipt.GasUsed,
		SourceBytes:     sourceBytes,
		CompressedBytes: len(payload),
		Address:         common.CodeStorageAddress.Hex(),
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return metrics, fmt.Errorf("upload transaction %s failed with receipt status %d", txHash.Hex(), receipt.Status)
	}
	return metrics, nil
}

func compressSource(path string) (int, string, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return 0, "", fmt.Errorf("read source: %w", err)
	}
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(source); err != nil {
		return 0, "", fmt.Errorf("gzip source: %w", err)
	}
	if err := gz.Close(); err != nil {
		return 0, "", fmt.Errorf("close gzip: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(compressed.Bytes())
	return len(source), encoded, nil
}

func resolveGasLimit(ctx context.Context, client *rpc.Client, args callArgs, fixed uint64) (hexutil.Uint64, error) {
	if fixed > 0 {
		return hexutil.Uint64(fixed), nil
	}
	gas, err := estimateGas(ctx, client, args)
	if err != nil {
		return 0, err
	}
	if gas == 0 {
		return 0, nil
	}
	withHeadroom := gas + gas/5
	if withHeadroom < gas {
		return hexutil.Uint64(gas), nil
	}
	return hexutil.Uint64(withHeadroom), nil
}

func estimateGas(ctx context.Context, client *rpc.Client, args callArgs) (uint64, error) {
	var gas hexutil.Uint64
	if err := client.CallContext(ctx, &gas, "eth_estimateGas", args); err != nil {
		return 0, err
	}
	return uint64(gas), nil
}

func sendTransaction(ctx context.Context, client *rpc.Client, args callArgs) (common.Hash, error) {
	var txHash common.Hash
	if err := client.CallContext(ctx, &txHash, "eth_sendTransaction", args); err != nil {
		return common.Hash{}, err
	}
	return txHash, nil
}

func waitReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	deadline := time.Now().Add(timeout)
	for {
		var receipt *types.Receipt
		if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
			if isReceiptIndexing(err) {
				if time.Now().After(deadline) {
					return nil, fmt.Errorf("timed out waiting for receipt %s", txHash.Hex())
				}
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("eth_getTransactionReceipt %s: %w", txHash.Hex(), err)
		}
		if receipt != nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for receipt %s", txHash.Hex())
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func isReceiptIndexing(err error) bool {
	// dev 模式刚提交交易时，receipt 查询可能早于 tx indexer 完成，此时继续轮询即可。
	return strings.Contains(strings.ToLower(err.Error()), "transaction indexing is in progress")
}

func ethCall(ctx context.Context, client *rpc.Client, args callArgs) ([]byte, error) {
	var out hexutil.Bytes
	if err := client.CallContext(ctx, &out, "eth_call", args, "latest"); err != nil {
		return nil, err
	}
	return out, nil
}

func benchmark(warmup, samples int, call func() ([]byte, error)) (benchmarkStats, error) {
	for i := 0; i < warmup; i++ {
		if _, err := call(); err != nil {
			return benchmarkStats{}, fmt.Errorf("warmup %d: %w", i+1, err)
		}
	}
	durations := make([]time.Duration, 0, samples)
	var firstOutput []byte
	for i := 0; i < samples; i++ {
		started := time.Now()
		out, err := call()
		elapsed := time.Since(started)
		if err != nil {
			return benchmarkStats{}, fmt.Errorf("sample %d: %w", i+1, err)
		}
		if i == 0 {
			firstOutput = append([]byte(nil), out...)
		} else if !bytes.Equal(firstOutput, out) {
			return benchmarkStats{}, fmt.Errorf("sample %d returned non-deterministic output: first=%s current=%s", i+1, hexutil.Encode(firstOutput), hexutil.Encode(out))
		}
		durations = append(durations, elapsed)
	}
	return summarize(durations, firstOutput), nil
}

func summarize(durations []time.Duration, output []byte) benchmarkStats {
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	total := time.Duration(0)
	for _, duration := range durations {
		total += duration
	}
	return benchmarkStats{
		first:  durations[0],
		mean:   total / time.Duration(len(durations)),
		p50:    percentile(sorted, 0.50),
		p95:    percentile(sorted, 0.95),
		min:    sorted[0],
		max:    sorted[len(sorted)-1],
		output: output,
	}
}

func percentile(sorted []time.Duration, q float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(float64(len(sorted))*q)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func metricsFromStats(scheme string, cfg config, gas uint64, output []byte, stats benchmarkStats) callMetrics {
	sum := sha256.Sum256(output)
	return callMetrics{
		Scheme:           scheme,
		Warmup:           cfg.warmup,
		Samples:          cfg.samples,
		GasEstimate:      gas,
		FirstMillis:      millis(stats.first),
		MeanMillis:       millis(stats.mean),
		P50Millis:        millis(stats.p50),
		P95Millis:        millis(stats.p95),
		MinMillis:        millis(stats.min),
		MaxMillis:        millis(stats.max),
		OutputHex:        hexutil.Encode(output),
		OutputSHA256:     hex.EncodeToString(sum[:]),
		OutputByteLength: len(output),
	}
}

func printSummary(res result) {
	fmt.Printf("Real-chain cryptoupgrade benchmark\n")
	fmt.Printf("  rpc:        %s\n", res.RPC)
	fmt.Printf("  sender:     %s\n", res.Sender)
	fmt.Printf("  algorithm:  %s\n", res.Algorithm)
	fmt.Printf("  input:      %s\n", res.InputHex)
	fmt.Printf("  precompile: %s\n", res.PrecompileAddress)
	fmt.Printf("\nSetup\n")
	fmt.Printf("  upgrade:    txs=%d gas=%d elapsed=%.3fms source=%dB compressed=%dB\n",
		res.UpgradeSetup.TxCount, res.UpgradeSetup.GasUsed, res.UpgradeSetup.ElapsedMillis,
		res.UpgradeSetup.SourceBytes, res.UpgradeSetup.CompressedBytes)
	fmt.Printf("  precompile: txs=%d gas=%d address=%s\n", res.PrecompileSetup.TxCount, res.PrecompileSetup.GasUsed, res.PrecompileSetup.Address)
	if res.Mode == "setup" {
		return
	}
	fmt.Printf("\nCall eth_call latency (%d warmup, %d samples)\n", res.UpgradeCall.Warmup, res.UpgradeCall.Samples)
	fmt.Printf("  upgrade:    mean=%.3fms p50=%.3fms p95=%.3fms min=%.3fms max=%.3fms gas=%d\n",
		res.UpgradeCall.MeanMillis, res.UpgradeCall.P50Millis, res.UpgradeCall.P95Millis,
		res.UpgradeCall.MinMillis, res.UpgradeCall.MaxMillis, res.UpgradeCall.GasEstimate)
	fmt.Printf("  precompile: mean=%.3fms p50=%.3fms p95=%.3fms min=%.3fms max=%.3fms gas=%d\n",
		res.PrecompileCall.MeanMillis, res.PrecompileCall.P50Millis, res.PrecompileCall.P95Millis,
		res.PrecompileCall.MinMillis, res.PrecompileCall.MaxMillis, res.PrecompileCall.GasEstimate)
	fmt.Printf("\nRatios, upgrade / precompile\n")
	fmt.Printf("  mean=%.2fx p50=%.2fx p95=%.2fx gas=%.2fx setup_gas=%d/%d output_match=%t\n",
		res.Ratios.MeanTime, res.Ratios.P50Time, res.Ratios.P95Time, res.Ratios.Gas,
		res.Ratios.UpgradeSetupGas, res.Ratios.PrecompileSetupGas, res.OutputMatched)
}

func writeJSONResult(path string, res result) error {
	if path == "" {
		return nil
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create JSON output directory: %w", err)
		}
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}

func millis(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / float64(time.Millisecond)
}

func safeRatioDuration(a, b time.Duration) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func safeRatioUint64(a, b uint64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func wrapPluginCompileHint(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "can not compile code") || strings.Contains(msg, "cannot compile code") {
		return fmt.Errorf("%w (check that the Geth node was built with CGO enabled and the Go plugin toolchain is available)", err)
	}
	return err
}
