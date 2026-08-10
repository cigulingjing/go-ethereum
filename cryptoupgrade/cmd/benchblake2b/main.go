package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	"github.com/ethereum/go-ethereum/crypto/blake2b"
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

type compiledSolidity struct {
	Contract string
	ABI      abi.ABI
	Bin      []byte
}

func main() {
	var (
		rpcURL            = flag.String("rpc", "http://127.0.0.1:8666", "execution RPC endpoint")
		source            = flag.String("source", "", "algorithm source file; empty generates a wrapper around crypto/blake2b.Sum256")
		solcPath          = flag.String("solc", defaultSolcPath(), "solc compiler path")
		evmVersion        = flag.String("evm-version", "paris", "solc EVM target; use paris or earlier when the chain does not support PUSH0 (Shanghai)")
		soliditySource    = flag.String("solidity-source", "cryptoupgrade/algorithm/contracts/Blake2b.sol", "Solidity Blake2b contract source")
		solidityContract  = flag.String("solidity-contract", "Blake2b", "Solidity contract name to select from solc combined-json output")
		solidityFunction  = flag.String("solidity-function", "Sum256", "Solidity function to call with the benchmark bytes input")
		name              = flag.String("name", "Sum256", "upgrade algorithm name")
		mode              = flag.String("mode", "all", "benchmark mode: all, deploy, call")
		inputText         = flag.String("input", "Hello world!", "bytes input as UTF-8 text")
		inputHex          = flag.String("input-hex", "", "bytes input as hex; overrides -input")
		iterations        = flag.Int("n", 300, "measured eth_call iterations")
		warmup            = flag.Int("warmup", 30, "warmup eth_call iterations before measurement")
		deployIterations  = flag.Int("deploy-n", 5, "deployment iterations for -mode deploy or all")
		upload            = flag.Bool("upload", true, "upload and activate the upgrade algorithm before call benchmarking")
		algoGas           = flag.Uint64("algo-gas", 72, "algorithm gas recorded in CodeStorage")
		upgradeDeployGas  = flag.Uint64("upgrade-deploy-gas", 8000000, "gas limit for upgrade upload transactions")
		solidityDeployGas = flag.Uint64("solidity-deploy-gas", 8000000, "gas limit for Solidity Blake2b contract deployment transactions")
		solidityAddress   = flag.String("solidity-address", "", "predeployed Solidity Blake2b contract address for call benchmarking")
		precompileAddress = flag.String("precompile-address", common.Blake2bSum256Address.Hex(), "built-in Blake2b Sum256 precompile address")
		from              = flag.String("from", "", "sender address; defaults to eth_accounts[0]")
	)
	flag.Parse()

	if err := run(*rpcURL, *source, *solcPath, *evmVersion, *soliditySource, *solidityContract, *solidityFunction, *name, *mode, *inputText, *inputHex, *iterations, *warmup, *deployIterations, *upload, *algoGas, *upgradeDeployGas, *solidityDeployGas, *solidityAddress, *precompileAddress, *from); err != nil {
		fmt.Fprintf(os.Stderr, "benchmark failed: %v\n", err)
		os.Exit(1)
	}
}

func run(rpcURL, source, solcPath, evmVersion, soliditySource, solidityContract, solidityFunction, name, mode, inputText, inputHex string, iterations, warmup, deployIterations int, upload bool, algoGas, upgradeDeployGas, solidityDeployGas uint64, solidityAddress, precompileAddress, fromText string) error {
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
	input, err := parseInput(inputText, inputHex)
	if err != nil {
		return err
	}
	if len(input) > 128 {
		return fmt.Errorf("blake2b benchmark supports inputs up to 128 bytes, got %d", len(input))
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
	expected := blake2b.Sum256(input)

	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return err
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return err
	}
	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return err
	}
	bytesArgs := abi.Arguments{{Type: bytesType}}
	bytes32Return := abi.Arguments{{Type: bytes32Type}}

	encodedArgs, err := bytesArgs.Pack(input)
	if err != nil {
		return err
	}
	upgradeCallData, err := codeStorageABI.Pack("callFunc", name, encodedArgs)
	if err != nil {
		return err
	}

	fmt.Printf("rpc: %s\n", rpcURL)
	fmt.Printf("sender: %s\n", fromAddr.Hex())
	fmt.Printf("input-len: %d input-hex=%s expected=%x\n", len(input), hex.EncodeToString(input), expected)

	tmpDir, err := os.MkdirTemp("", "cryptoupgrade-blake2b-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var solidity compiledSolidity
	solidity, err = compileSolidity(solcPath, evmVersion, soliditySource, solidityContract)
	if err != nil {
		return err
	}
	fmt.Printf("solidity: source=%s contract=%s function=%s bytecode=%d bytes solc=%s evm-version=%s\n", mustAbs(soliditySource), solidity.Contract, solidityFunction, len(solidity.Bin), solcPath, evmVersion)

	solidityCallData, err := solidity.ABI.Pack(solidityFunction, input)
	if err != nil {
		return err
	}

	if mode == "all" || mode == "deploy" {
		if err := runDeploymentBenchmark(ctx, client, codeStorageABI, solidity.Bin, fromAddr, tmpDir, name, solidity.Contract, deployIterations, algoGas, upgradeDeployGas, solidityDeployGas); err != nil {
			return err
		}
	}
	if mode == "deploy" {
		return nil
	}

	if upload {
		sourcePath := source
		if sourcePath == "" {
			sourcePath, err = writeBlake2bWrapperSource(tmpDir, name)
			if err != nil {
				return err
			}
		}
		result, err := uploadAlgorithm(ctx, client, codeStorageABI, fromAddr, sourcePath, name, algoGas, upgradeDeployGas)
		if err != nil {
			return err
		}
		printDeployResult("call-setup-upgrade-upload", result)
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
		solidityAddr, deployResult, err = deploySolidityContract(ctx, client, fromAddr, solidity.Bin, solidityDeployGas, solidity.Contract)
		if err != nil {
			return err
		}
		printDeployResult("call-setup-solidity-deploy", deployResult)
	}

	if !common.IsHexAddress(precompileAddress) {
		return fmt.Errorf("invalid precompile address %q", precompileAddress)
	}
	precompileAddr := common.HexToAddress(precompileAddress)
	fmt.Printf("call-setup-precompile-address: address=%s\n", precompileAddr.Hex())

	upgradeGas, err := estimateGas(ctx, client, txArgs{From: fromAddr, To: &common.CodeStorageAddress, Data: upgradeCallData})
	if err != nil {
		return err
	}
	solidityGas, err := estimateGas(ctx, client, txArgs{From: fromAddr, To: &solidityAddr, Data: solidityCallData})
	if err != nil {
		return err
	}
	precompileGas, err := estimateGas(ctx, client, txArgs{From: fromAddr, To: &precompileAddr, Data: input})
	if err != nil {
		return err
	}
	fmt.Printf("call-gas-estimate: upgrade=%d solidity=%d precompile=%d\n", upgradeGas, solidityGas, precompileGas)

	upgrade := func() ([32]byte, error) {
		out, err := ethCall(ctx, client, fromAddr, common.CodeStorageAddress, upgradeCallData)
		if err != nil {
			return [32]byte{}, err
		}
		return unpackBytes32(bytes32Return, out)
	}
	solidityCall := func() ([32]byte, error) {
		out, err := ethCall(ctx, client, fromAddr, solidityAddr, solidityCallData)
		if err != nil {
			return [32]byte{}, err
		}
		return unpackBytes32(bytes32Return, out)
	}
	precompile := func() ([32]byte, error) {
		out, err := ethCall(ctx, client, fromAddr, precompileAddr, input)
		if err != nil {
			return [32]byte{}, err
		}
		return unpackBytes32(bytes32Return, out)
	}

	upgradeStats, upgradeResult, err := benchmark("upgrade-callFunc-"+name, iterations, warmup, upgrade)
	if err != nil {
		return err
	}
	solidityStats, solidityResult, err := benchmark("solidity-contract-"+solidity.Contract, iterations, warmup, solidityCall)
	if err != nil {
		return err
	}
	precompileStats, precompileResult, err := benchmark("precompile-contract-blake2b", iterations, warmup, precompile)
	if err != nil {
		return err
	}
	if upgradeResult != expected {
		return fmt.Errorf("upgrade result mismatch: got %x want %x", upgradeResult, expected)
	}
	if solidityResult != expected {
		return fmt.Errorf("solidity result mismatch: got %x want %x", solidityResult, expected)
	}
	if precompileResult != expected {
		return fmt.Errorf("precompile result mismatch: got %x want %x", precompileResult, expected)
	}

	printStats(upgradeStats, upgradeResult, upgradeGas)
	printStats(solidityStats, solidityResult, solidityGas)
	printStats(precompileStats, precompileResult, precompileGas)
	fmt.Printf("ratio-upgrade/solidity: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(upgradeStats.Mean, solidityStats.Mean),
		ratio(upgradeStats.P50, solidityStats.P50),
		ratio(upgradeStats.P95, solidityStats.P95),
	)
	fmt.Printf("ratio-upgrade/precompile: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(upgradeStats.Mean, precompileStats.Mean),
		ratio(upgradeStats.P50, precompileStats.P50),
		ratio(upgradeStats.P95, precompileStats.P95),
	)
	fmt.Printf("ratio-solidity/precompile: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(solidityStats.Mean, precompileStats.Mean),
		ratio(solidityStats.P50, precompileStats.P50),
		ratio(solidityStats.P95, precompileStats.P95),
	)
	return nil
}

func runDeploymentBenchmark(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, solidityBin []byte, from common.Address, tmpDir, baseName, solidityContract string, iterations int, algoGas, upgradeDeployGas, solidityDeployGas uint64) error {
	fmt.Printf("deployment-test: iterations=%d upgrade-gas-limit=%d solidity-gas-limit=%d\n", iterations, upgradeDeployGas, solidityDeployGas)

	upgradeResults := make([]deployResult, 0, iterations)
	solidityResults := make([]deployResult, 0, iterations)
	stamp := time.Now().UnixNano()
	for i := 0; i < iterations; i++ {
		algoName := fmt.Sprintf("%sDeploy%d_%d", baseName, stamp, i)
		sourcePath, err := writeBlake2bWrapperSource(tmpDir, algoName)
		if err != nil {
			return err
		}
		result, err := uploadAlgorithm(ctx, client, codeStorageABI, from, sourcePath, algoName, algoGas, upgradeDeployGas)
		if err != nil {
			return err
		}
		upgradeResults = append(upgradeResults, result)

		_, solidityResult, err := deploySolidityContract(ctx, client, from, solidityBin, solidityDeployGas, solidityContract)
		if err != nil {
			return err
		}
		solidityResults = append(solidityResults, solidityResult)
	}

	printDeploySummary("deployment-upgrade-upload-"+baseName, upgradeResults)
	printDeploySummary("deployment-solidity-contract-"+solidityContract, solidityResults)
	fmt.Printf("deployment-ratio-upgrade/solidity: mean=%.2fx p50=%.2fx p95=%.2fx\n",
		ratio(deployDurationStats(upgradeResults).Mean, deployDurationStats(solidityResults).Mean),
		ratio(deployDurationStats(upgradeResults).P50, deployDurationStats(solidityResults).P50),
		ratio(deployDurationStats(upgradeResults).P95, deployDurationStats(solidityResults).P95),
	)
	return nil
}

func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, source, name string, algoGas, gasLimit uint64) (deployResult, error) {
	compressed, err := compressFile(source)
	if err != nil {
		return deployResult{}, err
	}
	data, err := codeStorageABI.Pack("uploadCode", name, compressed, algoGas, "bytes", "bytes32")
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
		Label:         "upgrade-upload-blake2b",
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

func shortContractName(name string) string {
	if i := strings.LastIndex(name, ":"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func deploySolidityContract(ctx context.Context, client *rpc.Client, from common.Address, initCode []byte, gasLimit uint64, contractName string) (common.Address, deployResult, error) {
	return deployContract(ctx, client, from, initCode, gasLimit, "solidity-contract-"+contractName+"-deploy")
}

func deployContract(ctx context.Context, client *rpc.Client, from common.Address, initCode []byte, gasLimit uint64, label string) (common.Address, deployResult, error) {
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
		return common.Address{}, deployResult{}, fmt.Errorf("%s failed: tx=%s gasUsed=%d", label, txHash.Hex(), receipt.GasUsed)
	}
	result := deployResult{
		Label:   label,
		TxHash:  txHash,
		Address: receipt.ContractAddress,
		Elapsed: time.Since(start),
		GasUsed: receipt.GasUsed,
	}
	return receipt.ContractAddress, result, nil
}

func benchmark(label string, iterations, warmup int, call func() ([32]byte, error)) (benchStats, [32]byte, error) {
	firstStart := time.Now()
	result, err := call()
	if err != nil {
		return benchStats{}, [32]byte{}, fmt.Errorf("%s first call: %w", label, err)
	}
	first := time.Since(firstStart)

	for i := 0; i < warmup; i++ {
		result, err = call()
		if err != nil {
			return benchStats{}, [32]byte{}, fmt.Errorf("%s warmup %d: %w", label, i, err)
		}
	}

	samples := make([]time.Duration, iterations)
	var sum time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		result, err = call()
		elapsed := time.Since(start)
		if err != nil {
			return benchStats{}, [32]byte{}, fmt.Errorf("%s iteration %d: %w", label, i, err)
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

func writeBlake2bWrapperSource(dir, name string) (string, error) {
	if !isASCIIIdentifier(name) {
		return "", fmt.Errorf("algorithm name %q is not a valid Go identifier", name)
	}
	source := fmt.Sprintf(`package main

import "github.com/ethereum/go-ethereum/crypto/blake2b"

func %s(data []byte) [32]byte {
	return blake2b.Sum256(data)
}
`, name)
	path := filepath.Join(dir, name+".go")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func unpackBytes32(args abi.Arguments, out []byte) ([32]byte, error) {
	values, err := args.Unpack(out)
	if err != nil {
		return [32]byte{}, err
	}
	if len(values) != 1 {
		return [32]byte{}, fmt.Errorf("expected one bytes32 output, got %d", len(values))
	}
	switch value := values[0].(type) {
	case [32]byte:
		return value, nil
	case common.Hash:
		return [32]byte(value), nil
	default:
		return [32]byte{}, fmt.Errorf("expected [32]byte output, got %T", values[0])
	}
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

func parseInput(inputText, inputHex string) ([]byte, error) {
	if inputHex == "" {
		return []byte(inputText), nil
	}
	trimmed := strings.TrimPrefix(inputHex, "0x")
	input, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid -input-hex: %w", err)
	}
	return input, nil
}

func printStats(stats benchStats, result [32]byte, gas uint64) {
	fmt.Printf("%s: result=%x gas-estimate=%d first=%s mean=%s p50=%s p95=%s min=%s max=%s\n",
		stats.Label, result, gas, stats.First, stats.Mean, stats.P50, stats.P95, stats.Min, stats.Max,
	)
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

// percentile 计算sorted中第p百分位的值，p为0-1之间的小数
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
