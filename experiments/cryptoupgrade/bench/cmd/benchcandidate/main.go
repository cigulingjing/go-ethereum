package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade/wasmtool"
	"github.com/ethereum/go-ethereum/rpc"
)

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type compiledSolidity struct {
	Contract string
	ABI      abi.ABI
	Bin      []byte
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
		rpcURL            = flag.String("rpc", "http://127.0.0.1:8666", "execution RPC endpoint")
		source            = flag.String("source", "cryptoupgrade/algorithm/wasm/sha256.wasm", "algorithm wasm file")
		name              = flag.String("name", "Sha256", "upgrade algorithm name")
		itype             = flag.String("itype", "bytes", "comma-separated upgrade input ABI types")
		otype             = flag.String("otype", "bytes", "comma-separated upgrade output ABI types")
		inputHex          = flag.String("input-hex", defaultSha256InputHex, "ABI-encoded upgrade input arguments as hex")
		expectedHex       = flag.String("expected-hex", "", "optional expected ABI-encoded return value as hex")
		mode              = flag.String("mode", "all", "benchmark mode: all, deploy, call")
		iterations        = flag.Int("n", 300, "measured eth_call iterations")
		warmup            = flag.Int("warmup", 30, "warmup eth_call iterations before measurement")
		deployIterations  = flag.Int("deploy-n", 5, "deployment iterations for -mode deploy or all")
		upload            = flag.Bool("upload", true, "upload and activate the upgrade algorithm before call benchmarking")
		algoGas           = flag.Uint64("algo-gas", 1, "algorithm gas recorded in CodeStorage")
		upgradeDeployGas  = flag.Uint64("upgrade-deploy-gas", 8000000, "gas limit for upgrade upload transactions")
		solidityDeployGas = flag.Uint64("solidity-deploy-gas", 8000000, "gas limit for Solidity contract deployment transactions")
		solcPath          = flag.String("solc", defaultSolcPath(), "solc compiler path")
		evmVersion        = flag.String("evm-version", "paris", "solc EVM target")
		soliditySource    = flag.String("solidity-source", "cryptoupgrade/algorithm/contracts/Sha256.sol", "Solidity contract source")
		solidityContract  = flag.String("solidity-contract", "Sha256Contract", "Solidity contract name to select")
		solidityFunction  = flag.String("solidity-function", "Sha256", "Solidity function name to call")
		solidityInputHex  = flag.String("solidity-input-hex", "", "optional ABI-encoded Solidity function arguments; defaults to -input-hex")
		solidityAddress   = flag.String("solidity-address", "", "predeployed Solidity contract address for call benchmarking")
		from              = flag.String("from", "", "sender address; defaults to eth_accounts[0]")
	)
	flag.Parse()

	if err := run(*rpcURL, *source, *name, *itype, *otype, *inputHex, *expectedHex, *mode, *iterations, *warmup, *deployIterations, *upload, *algoGas, *upgradeDeployGas, *solidityDeployGas, *solcPath, *evmVersion, *soliditySource, *solidityContract, *solidityFunction, *solidityInputHex, *solidityAddress, *from); err != nil {
		fmt.Fprintf(os.Stderr, "benchmark failed: %v\n", err)
		os.Exit(1)
	}
}

func run(rpcURL, source, name, itype, otype, inputHex, expectedHex, mode string, iterations, warmup, deployIterations int, upload bool, algoGas, upgradeDeployGas, solidityDeployGas uint64, solcPath, evmVersion, soliditySource, solidityContract, solidityFunction, solidityInputHex, solidityAddress, fromText string) error {
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

	encodedInput, err := parseHex(inputHex)
	if err != nil {
		return fmt.Errorf("invalid -input-hex: %w", err)
	}
	solidityInput := encodedInput
	if solidityInputHex != "" {
		solidityInput, err = parseHex(solidityInputHex)
		if err != nil {
			return fmt.Errorf("invalid -solidity-input-hex: %w", err)
		}
	}
	var expected []byte
	if expectedHex != "" {
		expected, err = parseHex(expectedHex)
		if err != nil {
			return fmt.Errorf("invalid -expected-hex: %w", err)
		}
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
	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return err
	}
	upgradeCallData, err := codeStorageABI.Pack("callFunc", name, encodedInput)
	if err != nil {
		return err
	}

	solidity, err := compileSolidity(solcPath, evmVersion, soliditySource, solidityContract)
	if err != nil {
		return err
	}
	method, ok := solidity.ABI.Methods[solidityFunction]
	if !ok {
		return fmt.Errorf("function %q not found in Solidity contract %s", solidityFunction, solidity.Contract)
	}
	solidityCallData := append(append([]byte{}, method.ID...), solidityInput...)

	fmt.Printf("rpc: %s\n", rpcURL)
	fmt.Printf("sender: %s\n", fromAddr.Hex())
	fmt.Printf("upgrade: source=%s name=%s itype=%q otype=%q input=%s\n", mustAbs(source), name, itype, otype, hex.EncodeToString(encodedInput))
	fmt.Printf("solidity: source=%s contract=%s function=%s bytecode=%d bytes input=%s\n", mustAbs(soliditySource), solidity.Contract, solidityFunction, len(solidity.Bin), hex.EncodeToString(solidityInput))

	if mode == "all" || mode == "deploy" {
		if err := runDeploymentBenchmark(ctx, client, codeStorageABI, solidity.Bin, fromAddr, source, name, itype, otype, solidity.Contract, deployIterations, algoGas, upgradeDeployGas, solidityDeployGas); err != nil {
			return err
		}
	}
	if mode == "deploy" {
		return nil
	}

	if upload {
		result, err := uploadAlgorithm(ctx, client, codeStorageABI, fromAddr, source, name, algoGas, upgradeDeployGas, itype, otype)
		if err != nil {
			return err
		}
		printDeployResult("call-setup-upgrade-upload", result)
		if err := waitCallFuncReady(ctx, client, fromAddr, common.CodeStorageAddress, upgradeCallData, 30*time.Second); err != nil {
			return err
		}
	}

	var solidityAddr common.Address
	if solidityAddress != "" {
		if !common.IsHexAddress(solidityAddress) {
			return fmt.Errorf("invalid solidity address %q", solidityAddress)
		}
		solidityAddr = common.HexToAddress(solidityAddress)
		fmt.Printf("call-setup-solidity-address: address=%s\n", solidityAddr.Hex())
	} else {
		var deployResult deployResult
		solidityAddr, deployResult, err = deployContract(ctx, client, fromAddr, solidity.Bin, solidityDeployGas, "solidity-contract-"+solidity.Contract+"-deploy")
		if err != nil {
			return err
		}
		printDeployResult("call-setup-solidity-deploy", deployResult)
	}

	upgradeGas, err := estimateGas(ctx, client, txArgs{From: fromAddr, To: &common.CodeStorageAddress, Data: upgradeCallData})
	if err != nil {
		return err
	}
	solidityGas, err := estimateGas(ctx, client, txArgs{From: fromAddr, To: &solidityAddr, Data: solidityCallData})
	if err != nil {
		return err
	}
	fmt.Printf("call-gas-estimate: upgrade=%d solidity=%d\n", upgradeGas, solidityGas)

	upgrade := func() ([]byte, error) {
		return ethCall(ctx, client, fromAddr, common.CodeStorageAddress, upgradeCallData)
	}
	solidityCall := func() ([]byte, error) {
		return ethCall(ctx, client, fromAddr, solidityAddr, solidityCallData)
	}
	upgradeStats, upgradeResult, err := benchmark("upgrade-callFunc-"+name, iterations, warmup, upgrade)
	if err != nil {
		return err
	}
	solidityStats, solidityResult, err := benchmark("solidity-contract-"+solidity.Contract, iterations, warmup, solidityCall)
	if err != nil {
		return err
	}
	if expected != nil {
		if !bytes.Equal(upgradeResult, expected) {
			return fmt.Errorf("upgrade result mismatch: got %x want %x", upgradeResult, expected)
		}
		if !bytes.Equal(solidityResult, expected) {
			return fmt.Errorf("solidity result mismatch: got %x want %x", solidityResult, expected)
		}
	} else if !bytes.Equal(upgradeResult, solidityResult) {
		return fmt.Errorf("upgrade/solidity result mismatch: upgrade=%x solidity=%x", upgradeResult, solidityResult)
	}

	printStats(upgradeStats, upgradeResult, upgradeGas)
	printStats(solidityStats, solidityResult, solidityGas)
	fmt.Printf("ratio-upgrade/solidity: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(upgradeStats.Mean, solidityStats.Mean),
		ratio(upgradeStats.P50, solidityStats.P50),
		ratio(upgradeStats.P95, solidityStats.P95),
	)
	return nil
}

func runDeploymentBenchmark(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, solidityBin []byte, from common.Address, source, name, itype, otype, solidityContract string, iterations int, algoGas, upgradeDeployGas, solidityDeployGas uint64) error {
	fmt.Printf("deployment-test: iterations=%d upgrade-gas-limit=%d solidity-gas-limit=%d\n", iterations, upgradeDeployGas, solidityDeployGas)
	upgradeResults := make([]deployResult, 0, iterations)
	solidityResults := make([]deployResult, 0, iterations)
	for i := 0; i < iterations; i++ {
		upgradeResult, err := uploadAlgorithm(ctx, client, codeStorageABI, from, source, name, algoGas, upgradeDeployGas, itype, otype)
		if err != nil {
			return err
		}
		upgradeResults = append(upgradeResults, upgradeResult)

		_, solidityResult, err := deployContract(ctx, client, from, solidityBin, solidityDeployGas, "solidity-contract-"+solidityContract+"-deploy")
		if err != nil {
			return err
		}
		solidityResults = append(solidityResults, solidityResult)
	}
	printDeploySummary("deployment-upgrade-upload-"+name, upgradeResults)
	printDeploySummary("deployment-solidity-contract-"+solidityContract, solidityResults)
	fmt.Printf("deployment-ratio-upgrade/solidity: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(deployDurationStats(upgradeResults).Mean, deployDurationStats(solidityResults).Mean),
		ratio(deployDurationStats(upgradeResults).P50, deployDurationStats(solidityResults).P50),
		ratio(deployDurationStats(upgradeResults).P95, deployDurationStats(solidityResults).P95),
	)
	return nil
}

func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, source, name string, algoGas, gasLimit uint64, itype, otype string) (deployResult, error) {
	compressed, err := wasmtool.EncodePath(ctx, source, wasmtool.Spec{
		Function:    name,
		InputTypes:  splitABITypeList(itype),
		OutputTypes: splitABITypeList(otype),
	})
	if err != nil {
		return deployResult{}, err
	}
	data, err := codeStorageABI.Pack("uploadCode", name, compressed, algoGas, itype, otype)
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
		Label:         "upgrade-upload-" + name,
		TxHash:        txHash,
		Address:       common.CodeStorageAddress,
		Elapsed:       time.Since(start),
		GasUsed:       receipt.GasUsed,
		Source:        mustAbs(source),
		CompressedLen: len(compressed),
	}, nil
}

func compileSolidity(solcPath, evmVersion, sourcePath, contractName string) (compiledSolidity, error) {
	args := []string{"--optimize", "--via-ir", "--combined-json", "abi,bin"}
	if evmVersion != "" {
		args = append(args, "--evm-version", evmVersion)
	}
	args = append(args, sourcePath)
	cmd := exec.Command(solcPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return compiledSolidity{}, fmt.Errorf("compile solidity: %w\n%s", err, string(out))
	}

	var combined struct {
		Contracts map[string]struct {
			ABI json.RawMessage `json:"abi"`
			Bin string          `json:"bin"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(out, &combined); err != nil {
		return compiledSolidity{}, err
	}
	var available []string
	for name, contract := range combined.Contracts {
		shortName := shortContractName(name)
		available = append(available, shortName)
		if shortName != contractName {
			continue
		}
		contractABI, err := abi.JSON(bytes.NewReader(contract.ABI))
		if err != nil {
			return compiledSolidity{}, err
		}
		bin, err := hex.DecodeString(contract.Bin)
		if err != nil {
			return compiledSolidity{}, err
		}
		if len(bin) == 0 {
			return compiledSolidity{}, fmt.Errorf("solidity bytecode for %s is empty", contractName)
		}
		return compiledSolidity{Contract: shortName, ABI: contractABI, Bin: bin}, nil
	}
	sort.Strings(available)
	return compiledSolidity{}, fmt.Errorf("contract %q not found in %s; available contracts: %s", contractName, sourcePath, strings.Join(available, ", "))
}

func deployContract(ctx context.Context, client *rpc.Client, from common.Address, initCode []byte, gasLimit uint64, label string) (common.Address, deployResult, error) {
	gas := hexutil.Uint64(gasLimit)
	args := txArgs{From: from, Data: initCode, Gas: &gas}

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
		return common.Address{}, deployResult{}, fmt.Errorf("%s failed: tx=%s gasUsed=%d", label, txHash.Hex(), receipt.GasUsed)
	}
	result := deployResult{Label: label, TxHash: txHash, Address: receipt.ContractAddress, Elapsed: time.Since(start), GasUsed: receipt.GasUsed}
	return receipt.ContractAddress, result, nil
}

func benchmark(label string, iterations, warmup int, call func() ([]byte, error)) (benchStats, []byte, error) {
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

func ethCall(ctx context.Context, client *rpc.Client, from common.Address, to common.Address, data []byte) (hexutil.Bytes, error) {
	gas := hexutil.Uint64(5000000)
	var out hexutil.Bytes
	if err := client.CallContext(ctx, &out, "eth_call", txArgs{From: from, To: &to, Gas: &gas, Data: data}, "latest"); err != nil {
		return nil, err
	}
	return out, nil
}

func waitCallFuncReady(ctx context.Context, client *rpc.Client, from common.Address, to common.Address, data []byte, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, err := ethCall(ctx, client, from, to, data); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for event-driven activation: %w", lastErr)
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

func parseHex(input string) ([]byte, error) {
	trimmed := strings.TrimPrefix(input, "0x")
	if trimmed == "" {
		return nil, nil
	}
	return hex.DecodeString(trimmed)
}

func splitABITypeList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func printStats(stats benchStats, result []byte, gas uint64) {
	fmt.Printf("%s: result=%x gas-estimate=%d first=%s mean=%s p50=%s p95=%s min=%s max=%s\n", stats.Label, result, gas, stats.First, stats.Mean, stats.P50, stats.P95, stats.Min, stats.Max)
}

func printDeployResult(label string, result deployResult) {
	fmt.Printf("%s: tx=%s address=%s gas-used=%d elapsed=%s", label, result.TxHash.Hex(), result.Address.Hex(), result.GasUsed, result.Elapsed)
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
	fmt.Printf("%s: n=%d first=%s mean=%s p50=%s p95=%s min=%s max=%s gas-mean=%.1f gas-min=%d gas-max=%d\n", label, len(results), stats.First, stats.Mean, stats.P50, stats.P95, stats.Min, stats.Max, float64(gasSum)/float64(len(results)), minGas, maxGas)
}

func deployDurationStats(results []deployResult) benchStats {
	samples := make([]time.Duration, len(results))
	var sum time.Duration
	for i, result := range results {
		samples[i] = result.Elapsed
		sum += result.Elapsed
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return benchStats{First: results[0].Elapsed, Min: samples[0], Max: samples[len(samples)-1], Mean: sum / time.Duration(len(samples)), P50: percentile(samples, 0.50), P95: percentile(samples, 0.95)}
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

func shortContractName(name string) string {
	if i := strings.LastIndex(name, ":"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func defaultSolcPath() string {
	candidates := []string{
		"../.tools/solc/solc-0.8.26",
		"/home/liuqi/project/.tools/solc/solc-0.8.26",
		"solc",
	}
	for _, candidate := range candidates {
		if candidate == "solc" {
			return candidate
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "solc"
}

const defaultSha256InputHex = "0000000000000000000000000000000000000000000000000000000000000020" +
	"000000000000000000000000000000000000000000000000000000000000000c" +
	"48656c6c6f20776f726c64210000000000000000000000000000000000000000"
