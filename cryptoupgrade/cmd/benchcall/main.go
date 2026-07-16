package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"math/big"
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

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type benchStats struct {
	Label string
	First time.Duration
	Min   time.Duration
	Max   time.Duration
	Mean  time.Duration
	P50   time.Duration
	P95   time.Duration
}

type deployResult struct {
	Label         string
	TxHash        common.Hash
	Address       common.Address
	Elapsed       time.Duration
	GasUsed       uint64
	Source        string
	CompressedLen int
}

func main() {
	var (
		rpcURL               = flag.String("rpc", "http://127.0.0.1:8666", "execution RPC endpoint")
		source               = flag.String("source", "cryptoupgrade/algorithm/add.go", "algorithm source file")
		name                 = flag.String("name", "Add", "upgrade algorithm name")
		aText                = flag.String("a", "100", "first int256 argument")
		bText                = flag.String("b", "100", "second int256 argument")
		mode                 = flag.String("mode", "all", "benchmark mode: all, deploy, call")
		iterations           = flag.Int("n", 300, "measured eth_call iterations")
		warmup               = flag.Int("warmup", 30, "warmup eth_call iterations before measurement")
		deployIterations     = flag.Int("deploy-n", 5, "deployment iterations for -mode deploy or all")
		upload               = flag.Bool("upload", true, "upload and activate the upgrade algorithm before call benchmarking")
		algoGas              = flag.Uint64("algo-gas", 1, "algorithm gas recorded in CodeStorage")
		upgradeDeployGas     = flag.Uint64("upgrade-deploy-gas", 5000000, "gas limit for upgrade upload transactions")
		traditionalDeployGas = flag.Uint64("traditional-deploy-gas", 500000, "gas limit for traditional contract deployment transactions")
		traditionalAddress   = flag.String("traditional-address", "", "predeployed traditional add contract address for call benchmarking")
		from                 = flag.String("from", "", "sender address; defaults to eth_accounts[0]")
	)
	flag.Parse()

	if err := run(*rpcURL, *source, *name, *aText, *bText, *mode, *iterations, *warmup, *deployIterations, *upload, *algoGas, *upgradeDeployGas, *traditionalDeployGas, *traditionalAddress, *from); err != nil {
		fmt.Fprintf(os.Stderr, "benchmark failed: %v\n", err)
		os.Exit(1)
	}
}

func run(rpcURL, source, name, aText, bText, mode string, iterations, warmup, deployIterations int, upload bool, algoGas, upgradeDeployGas, traditionalDeployGas uint64, traditionalAddress, fromText string) error {
	if iterations <= 0 {
		return fmt.Errorf("-n must be positive")
	}
	if warmup < 0 {
		return fmt.Errorf("-warmup must be non-negative")
	}
	if deployIterations <= 0 {
		return fmt.Errorf("-deploy-n must be positive")
	}
	if mode != "all" && mode != "deploy" && mode != "call" {
		return fmt.Errorf("-mode must be one of all, deploy, call")
	}

	ctx := context.Background()
	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return err
	}
	defer client.Close()

	fromAddr, err := sender(ctx, client, fromText)
	if err != nil {
		return err
	}
	a, ok := new(big.Int).SetString(aText, 10)
	if !ok {
		return fmt.Errorf("invalid a argument %q", aText)
	}
	b, ok := new(big.Int).SetString(bText, 10)
	if !ok {
		return fmt.Errorf("invalid b argument %q", bText)
	}
	expected := new(big.Int).Add(a, b)

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

	encodedArgs, err := int256Args.Pack(a, b)
	if err != nil {
		return err
	}
	upgradeCallData, err := codeStorageABI.Pack("callFunc", name, encodedArgs)
	if err != nil {
		return err
	}

	fmt.Printf("rpc: %s\n", rpcURL)
	fmt.Printf("sender: %s\n", fromAddr.Hex())
	fmt.Printf("args: %s + %s, expected=%s\n", a.String(), b.String(), expected.String())

	traditionalABI, err := abi.JSON(strings.NewReader(traditionalAddABIJSON))
	if err != nil {
		return err
	}
	if mode == "all" || mode == "deploy" {
		if err := runDeploymentBenchmark(ctx, client, codeStorageABI, traditionalABI, fromAddr, name, deployIterations, algoGas, upgradeDeployGas, traditionalDeployGas); err != nil {
			return err
		}
	}
	if mode == "deploy" {
		return nil
	}

	if upload {
		result, err := uploadAlgorithm(ctx, client, codeStorageABI, fromAddr, source, name, algoGas, upgradeDeployGas)
		if err != nil {
			return err
		}
		printDeployResult("call-setup-upgrade-upload", result)
	}

	var traditionalAddr common.Address
	if traditionalAddress != "" {
		if !common.IsHexAddress(traditionalAddress) {
			return fmt.Errorf("invalid traditional address %q", traditionalAddress)
		}
		traditionalAddr = common.HexToAddress(traditionalAddress)
		fmt.Printf("call-setup-traditional-address: address=%s\n", traditionalAddr.Hex())
	} else {
		var deployResult deployResult
		traditionalAddr, deployResult, err = deployTraditionalAdd(ctx, client, traditionalABI, fromAddr, traditionalDeployGas)
		if err != nil {
			return err
		}
		printDeployResult("call-setup-traditional-deploy", deployResult)
	}
	traditionalCallData, err := traditionalABI.Pack("add", a, b)
	if err != nil {
		return err
	}

	upgrade := func() (*big.Int, error) {
		out, err := ethCall(ctx, client, fromAddr, common.CodeStorageAddress, upgradeCallData)
		if err != nil {
			return nil, err
		}
		return unpackSingleInt256(int256Return, out)
	}
	traditional := func() (*big.Int, error) {
		out, err := ethCall(ctx, client, fromAddr, traditionalAddr, traditionalCallData)
		if err != nil {
			return nil, err
		}
		return unpackSingleInt256(int256Return, out)
	}

	upgradeStats, upgradeResult, err := benchmark("upgrade-callFunc", iterations, warmup, upgrade)
	if err != nil {
		return err
	}
	traditionalStats, traditionalResult, err := benchmark("traditional-contract", iterations, warmup, traditional)
	if err != nil {
		return err
	}
	if upgradeResult.Cmp(expected) != 0 {
		return fmt.Errorf("upgrade result mismatch: got %s want %s", upgradeResult, expected)
	}
	if traditionalResult.Cmp(expected) != 0 {
		return fmt.Errorf("traditional result mismatch: got %s want %s", traditionalResult, expected)
	}

	printStats(upgradeStats, upgradeResult)
	printStats(traditionalStats, traditionalResult)
	fmt.Printf("ratio-upgrade/traditional: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(upgradeStats.Mean, traditionalStats.Mean),
		ratio(upgradeStats.P50, traditionalStats.P50),
		ratio(upgradeStats.P95, traditionalStats.P95),
	)
	return nil
}

func runDeploymentBenchmark(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, traditionalABI abi.ABI, from common.Address, baseName string, iterations int, algoGas, upgradeDeployGas, traditionalDeployGas uint64) error {
	fmt.Printf("deployment-test: iterations=%d upgrade-gas-limit=%d traditional-gas-limit=%d\n", iterations, upgradeDeployGas, traditionalDeployGas)

	upgradeResults := make([]deployResult, 0, iterations)
	traditionalResults := make([]deployResult, 0, iterations)
	tmpDir, err := os.MkdirTemp("", "cryptoupgrade-deploy-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	stamp := time.Now().UnixNano()
	for i := 0; i < iterations; i++ {
		algoName := fmt.Sprintf("%sDeploy%d_%d", baseName, stamp, i)
		sourcePath, err := writeGeneratedAddSource(tmpDir, algoName)
		if err != nil {
			return err
		}
		result, err := uploadAlgorithm(ctx, client, codeStorageABI, from, sourcePath, algoName, algoGas, upgradeDeployGas)
		if err != nil {
			return err
		}
		upgradeResults = append(upgradeResults, result)

		_, traditionalResult, err := deployTraditionalAdd(ctx, client, traditionalABI, from, traditionalDeployGas)
		if err != nil {
			return err
		}
		traditionalResults = append(traditionalResults, traditionalResult)
	}

	printDeploySummary("deployment-upgrade-upload", upgradeResults)
	printDeploySummary("deployment-traditional-contract", traditionalResults)
	fmt.Printf("deployment-ratio-upgrade/traditional: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(deployDurationStats(upgradeResults).Mean, deployDurationStats(traditionalResults).Mean),
		ratio(deployDurationStats(upgradeResults).P50, deployDurationStats(traditionalResults).P50),
		ratio(deployDurationStats(upgradeResults).P95, deployDurationStats(traditionalResults).P95),
	)
	return nil
}

func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, source, name string, algoGas, gasLimit uint64) (deployResult, error) {
	compressed, err := compressFile(source)
	if err != nil {
		return deployResult{}, err
	}
	data, err := codeStorageABI.Pack("uploadCode", name, compressed, algoGas, "int256,int256", "int256")
	if err != nil {
		return deployResult{}, err
	}
	gas := hexutil.Uint64(gasLimit)
	args := txArgs{
		From: from,
		To:   &common.CodeStorageAddress,
		Data: data,
		Gas:  &gas,
	}

	start := time.Now()
	txHash, err := sendTransaction(ctx, client, args)
	if err != nil {
		return deployResult{}, err
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return deployResult{}, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return deployResult{}, fmt.Errorf("upload failed: tx=%s gasUsed=%d", txHash.Hex(), receipt.GasUsed)
	}
	return deployResult{
		Label:         "upgrade-upload",
		TxHash:        txHash,
		Address:       common.CodeStorageAddress,
		Elapsed:       time.Since(start),
		GasUsed:       receipt.GasUsed,
		Source:        mustAbs(source),
		CompressedLen: len(compressed),
	}, nil
}

func deployTraditionalAdd(ctx context.Context, client *rpc.Client, contractABI abi.ABI, from common.Address, gasLimit uint64) (common.Address, deployResult, error) {
	initCode := traditionalAddInitCode(contractABI.Methods["add"].ID)
	gas := hexutil.Uint64(gasLimit)
	args := txArgs{
		From: from,
		Data: initCode,
		Gas:  &gas,
	}

	start := time.Now()
	txHash, err := sendTransaction(ctx, client, args)
	if err != nil {
		return common.Address{}, deployResult{}, err
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return common.Address{}, deployResult{}, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return common.Address{}, deployResult{}, fmt.Errorf("traditional deploy failed: tx=%s gasUsed=%d", txHash.Hex(), receipt.GasUsed)
	}
	result := deployResult{
		Label:   "traditional-contract-deploy",
		TxHash:  txHash,
		Address: receipt.ContractAddress,
		Elapsed: time.Since(start),
		GasUsed: receipt.GasUsed,
	}
	return receipt.ContractAddress, result, nil
}

func writeGeneratedAddSource(dir, name string) (string, error) {
	if !isASCIIIdentifier(name) {
		return "", fmt.Errorf("generated algorithm name %q is not a valid Go identifier", name)
	}
	source := fmt.Sprintf(`package main

import "math/big"

func %s(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}
`, name)
	path := filepath.Join(dir, name+".go")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func isASCIIIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		if i == 0 {
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' {
				continue
			}
			return false
		}
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return false
	}
	return true
}

func printDeployResult(label string, result deployResult) {
	fmt.Printf("%s: tx=%s address=%s gas-used=%d elapsed=%s",
		label, result.TxHash.Hex(), result.Address.Hex(), result.GasUsed, result.Elapsed,
	)
	if result.Source != "" {
		fmt.Printf(" source=%s", result.Source)
	}
	if result.CompressedLen > 0 {
		fmt.Printf(" gzip-base64-len=%d", result.CompressedLen)
	}
	fmt.Println()
}

func printDeploySummary(label string, results []deployResult) {
	if len(results) == 0 {
		return
	}
	stats := deployDurationStats(results)
	minGas, maxGas := results[0].GasUsed, results[0].GasUsed
	var gasSum uint64
	for _, result := range results {
		if result.GasUsed < minGas {
			minGas = result.GasUsed
		}
		if result.GasUsed > maxGas {
			maxGas = result.GasUsed
		}
		gasSum += result.GasUsed
	}
	fmt.Printf("%s: n=%d first=%s mean=%s p50=%s p95=%s min=%s max=%s gas-mean=%.1f gas-min=%d gas-max=%d\n",
		label, len(results), stats.First, stats.Mean, stats.P50, stats.P95, stats.Min, stats.Max,
		float64(gasSum)/float64(len(results)), minGas, maxGas,
	)
}

func deployDurationStats(results []deployResult) benchStats {
	samples := make([]time.Duration, len(results))
	var sum time.Duration
	for i, result := range results {
		samples[i] = result.Elapsed
		sum += result.Elapsed
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return benchStats{
		First: results[0].Elapsed,
		Min:   samples[0],
		Max:   samples[len(samples)-1],
		Mean:  sum / time.Duration(len(samples)),
		P50:   percentile(samples, 0.50),
		P95:   percentile(samples, 0.95),
	}
}

func benchmark(label string, iterations, warmup int, call func() (*big.Int, error)) (benchStats, *big.Int, error) {
	firstStart := time.Now()
	result, err := call()
	if err != nil {
		return benchStats{}, nil, fmt.Errorf("%s first call: %w", label, err)
	}
	first := time.Since(firstStart)

	for i := 0; i < warmup; i++ {
		result, err = call()
		if err != nil {
			return benchStats{}, nil, fmt.Errorf("%s warmup %d: %w", label, i, err)
		}
	}

	samples := make([]time.Duration, iterations)
	var sum time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		result, err = call()
		elapsed := time.Since(start)
		if err != nil {
			return benchStats{}, nil, fmt.Errorf("%s iteration %d: %w", label, i, err)
		}
		samples[i] = elapsed
		sum += elapsed
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	stats := benchStats{
		Label: label,
		First: first,
		Min:   samples[0],
		Max:   samples[len(samples)-1],
		Mean:  sum / time.Duration(len(samples)),
		P50:   percentile(samples, 0.50),
		P95:   percentile(samples, 0.95),
	}
	return stats, result, nil
}

func printStats(stats benchStats, result *big.Int) {
	fmt.Printf("%s: result=%s first=%s mean=%s p50=%s p95=%s min=%s max=%s\n",
		stats.Label, result.String(), stats.First, stats.Mean, stats.P50, stats.P95, stats.Min, stats.Max,
	)
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 1 {
		return sorted[0]
	}
	index := int(float64(len(sorted)-1)*p + 0.5)
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func ratio(a, b time.Duration) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func unpackSingleInt256(args abi.Arguments, out []byte) (*big.Int, error) {
	values, err := args.Unpack(out)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("expected one int256 output, got %d", len(values))
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("expected *big.Int output, got %T", values[0])
	}
	return value, nil
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
		var receipt *types.Receipt
		if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
			if strings.Contains(err.Error(), "transaction indexing is in progress") {
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

const traditionalAddABIJSON = `[
  {
    "inputs": [
      {
        "internalType": "int256",
        "name": "a",
        "type": "int256"
      },
      {
        "internalType": "int256",
        "name": "b",
        "type": "int256"
      }
    ],
    "name": "add",
    "outputs": [
      {
        "internalType": "int256",
        "name": "",
        "type": "int256"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  }
]`

func traditionalAddInitCode(selector []byte) []byte {
	if len(selector) != 4 {
		panic("traditional add selector must be 4 bytes")
	}
	runtime := []byte{
		0x60, 0x00, 0x35, 0x60, 0xe0, 0x1c, 0x63,
		selector[0], selector[1], selector[2], selector[3],
		0x14, 0x60, 0x14, 0x57,
		0x60, 0x00, 0x60, 0x00, 0xfd,
		0x5b, 0x60, 0x04, 0x35, 0x60, 0x24, 0x35, 0x01,
		0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3,
	}
	init := []byte{
		0x60, byte(len(runtime)),
		0x60, 0x0c,
		0x60, 0x00,
		0x39,
		0x60, byte(len(runtime)),
		0x60, 0x00,
		0xf3,
	}
	return append(init, runtime...)
}
