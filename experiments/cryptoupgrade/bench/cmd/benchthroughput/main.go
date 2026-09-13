package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
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

const (
	experimentName = "lab-throughput"
	schemeUpgrade  = "upgrade"
	schemeContract = "contract"
)

type config struct {
	rpc               string
	from              string
	algorithm         string
	polyLeftLen       int
	polyRightLen      int
	polyModulus       string
	txCounts          string
	schemes           string
	upload            bool
	solcPath          string
	evmVersion        string
	txGas             uint64
	uploadGas         uint64
	contractDeployGas uint64
	outputDir         string
	outputJSON        string
}

type algorithmFixture struct {
	Algorithm          string
	UpgradeName        string
	SourcePath         string
	ContractSource     string
	ContractName       string
	ContractFunction   string
	UpgradeInputTypes  []string
	UpgradeOutputTypes []string
	ContractInputTypes []string
	ContractOutputTypes []string
	UpgradeValues      []interface{}
	ContractValues     []interface{}
	Input              map[string]string
	AlgoGas            uint64
	ResultFileSuffix   string
}

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type throughputRun struct {
	TxCount         int     `json:"txCount"`
	SuccessCount    int     `json:"successCount"`
	FailedCount     int     `json:"failedCount"`
	ElapsedSeconds  float64 `json:"elapsedSeconds"`
	ThroughputTPS   float64 `json:"throughputTps"`
	TotalGasUsed    uint64  `json:"totalGasUsed"`
	FirstBlock      uint64  `json:"firstBlock"`
	LastBlock       uint64  `json:"lastBlock"`
	BlocksSpanned   uint64  `json:"blocksSpanned"`
}

type schemeResult struct {
	Scheme      string          `json:"scheme"`
	Setup       setupReference  `json:"setup"`
	Runs        []throughputRun `json:"runs"`
}

type setupReference struct {
	Action        string `json:"action"`
	Address       string `json:"address"`
	TxHash        string `json:"txHash,omitempty"`
	ReceiptStatus uint64 `json:"receiptStatus,omitempty"`
}

type suiteResult struct {
	Experiment    string         `json:"experiment"`
	Timestamp     string         `json:"timestamp"`
	RPC           string         `json:"rpc"`
	Sender        string         `json:"sender"`
	Algorithm     string         `json:"algorithm"`
	Input         map[string]string `json:"input"`
	ConsensusNote string         `json:"consensusNote"`
	Command       []string       `json:"command"`
	Results       []schemeResult `json:"results"`
	OutputJSON    string         `json:"outputJson"`
	Notes         []string       `json:"notes"`
}

type callTarget struct {
	scheme   string
	address  common.Address
	calldata []byte
}

type compiledContract struct {
	Contract string
	ABI      abi.ABI
	Bin      []byte
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchthroughput: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		return err
	}
	counts, err := parseTxCounts(cfg.txCounts)
	if err != nil {
		return err
	}
	selectedSchemes, err := parseSchemes(cfg.schemes)
	if err != nil {
		return err
	}

	fixture, err := buildFixture(cfg)
	if err != nil {
		return err
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
	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	outputJSON := cfg.outputJSON
	if outputJSON == "" {
		outputJSON = filepath.Join(cfg.outputDir, fixture.ResultFileSuffix+"-"+now.Format("20060102-150405")+".json")
	}

	input := make(map[string]string, len(fixture.Input)+1)
	for key, value := range fixture.Input {
		input[key] = value
	}
	input["txCounts"] = cfg.txCounts

	suite := suiteResult{
		Experiment: experimentName,
		Timestamp:  now.Format(time.RFC3339Nano),
		RPC:        cfg.rpc,
		Sender:     from.Hex(),
		Algorithm:  fixture.Algorithm,
		Input:      input,
		ConsensusNote: "single-node Clique; throughput = successCount / elapsedSeconds",
		Command:       os.Args,
		OutputJSON:    outputJSON,
		Notes: []string{
			"throughput is measured from first eth_sendTransaction to last receipt confirmation",
			"setup transactions (upload/deploy) are excluded from throughput measurement",
			"transactions are submitted sequentially without waiting for receipts between sends",
		},
	}

	for scheme := range selectedSchemes {
		fmt.Printf("setting up %s...\n", scheme)
		target, setup, err := prepareScheme(ctx, client, codeStorageABI, from, cfg, fixture, scheme)
		if err != nil {
			return fmt.Errorf("%s setup: %w", scheme, err)
		}
		if err := validateTarget(ctx, client, from, target, cfg.txGas); err != nil {
			return fmt.Errorf("%s validation: %w", scheme, err)
		}

		runs := make([]throughputRun, 0, len(counts))
		for _, count := range counts {
			fmt.Printf("  %s throughput n=%d...\n", scheme, count)
			runResult, err := measureThroughput(ctx, client, from, target, count, cfg.txGas)
			if err != nil {
				return fmt.Errorf("%s n=%d: %w", scheme, count, err)
			}
			runs = append(runs, runResult)
			fmt.Printf("    success=%d elapsed=%.2fs throughput=%.3f tx/s\n",
				runResult.SuccessCount, runResult.ElapsedSeconds, runResult.ThroughputTPS)
		}
		suite.Results = append(suite.Results, schemeResult{
			Scheme: scheme,
			Setup:  setup,
			Runs:   runs,
		})
	}

	sort.Slice(suite.Results, func(i, j int) bool {
		return suite.Results[i].Scheme < suite.Results[j].Scheme
	})

	if err := writeJSONResult(outputJSON, suite); err != nil {
		return err
	}
	printSummary(suite)
	return nil
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.rpc, "rpc", "http://127.0.0.1:8761", "JSON-RPC endpoint")
	flag.StringVar(&cfg.from, "from", "", "sender address; defaults to eth_accounts[0]")
	flag.StringVar(&cfg.algorithm, "algorithm", "PolynomialMul", "algorithm to benchmark: PolynomialMul or Blake2bSum256")
	flag.IntVar(&cfg.polyLeftLen, "poly-left-len", 10, "PolynomialMul left polynomial length")
	flag.IntVar(&cfg.polyRightLen, "poly-right-len", 10, "PolynomialMul right polynomial length")
	flag.StringVar(&cfg.polyModulus, "poly-modulus", "12289", "PolynomialMul modulus")
	flag.StringVar(&cfg.txCounts, "tx-counts", "10,20,30,40,50", "comma-separated transaction counts")
	flag.StringVar(&cfg.schemes, "schemes", "upgrade,contract", "comma-separated schemes: upgrade,contract")
	flag.BoolVar(&cfg.upload, "upload", true, "upload WASM through CodeStorage before measuring upgrade scheme")
	flag.StringVar(&cfg.solcPath, "solc", defaultSolcPath(), "solc compiler path")
	flag.StringVar(&cfg.evmVersion, "evm-version", "paris", "solc EVM target")
	flag.Uint64Var(&cfg.txGas, "tx-gas", 0, "gas limit per algorithm transaction; 0 uses node estimate + headroom")
	flag.Uint64Var(&cfg.uploadGas, "upload-gas", 5000000, "gas limit for CodeStorage.uploadCode")
	flag.Uint64Var(&cfg.contractDeployGas, "contract-deploy-gas", 0, "gas limit for contract deployment; 0 lets node estimate")
	flag.StringVar(&cfg.outputDir, "output-dir", "experiments/cryptoupgrade/results/throughput", "directory used when -output-json is empty")
	flag.StringVar(&cfg.outputJSON, "output-json", "", "write JSON result to this file")
	flag.Parse()
	return cfg
}

func (cfg *config) validate() error {
	if cfg.rpc == "" {
		return errors.New("-rpc is required")
	}
	switch algorithmKey(cfg.algorithm) {
	case "polynomialmul":
		if cfg.polyLeftLen <= 0 || cfg.polyRightLen <= 0 {
			return errors.New("poly-left-len and poly-right-len must be positive")
		}
	case "blake2bsum256":
	default:
		return fmt.Errorf("unsupported -algorithm %q (supported: PolynomialMul, Blake2bSum256)", cfg.algorithm)
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return fmt.Errorf("invalid -from %q", cfg.from)
	}
	return nil
}

func buildFixture(cfg config) (algorithmFixture, error) {
	switch algorithmKey(cfg.algorithm) {
	case "polynomialmul":
		modulus, err := parsePositiveBigInt("poly-modulus", cfg.polyModulus)
		if err != nil {
			return algorithmFixture{}, err
		}
		left := buildPolynomialCoeffs(cfg.polyLeftLen, modulus, 0)
		right := buildPolynomialCoeffs(cfg.polyRightLen, modulus, 0)
		return algorithmFixture{
			Algorithm:           "PolynomialMul",
			UpgradeName:         "PolynomialMul",
			SourcePath:          "experiments/cryptoupgrade/algorithm/go/polynomial_mul.wasm",
			ContractSource:      "experiments/cryptoupgrade/algorithm/contracts/src/PolynomialMul.sol",
			ContractName:        "PolynomialMulContract",
			ContractFunction:    "PolynomialMul",
			UpgradeInputTypes:   []string{"uint256[]", "uint256[]", "uint256"},
			UpgradeOutputTypes:  []string{"uint256[]"},
			ContractInputTypes:  []string{"uint256[]", "uint256[]", "uint256"},
			ContractOutputTypes: []string{"uint256[]"},
			UpgradeValues:       []interface{}{left, right, modulus},
			ContractValues:      []interface{}{left, right, modulus},
			Input: map[string]string{
				"leftLen":       fmt.Sprintf("%d", cfg.polyLeftLen),
				"rightLen":      fmt.Sprintf("%d", cfg.polyRightLen),
				"mulIterations": fmt.Sprintf("%d", cfg.polyLeftLen*cfg.polyRightLen),
				"modulus":       modulus.String(),
			},
			AlgoGas:          polynomialMulAlgoGas(cfg.polyLeftLen, cfg.polyRightLen),
			ResultFileSuffix: "polynomial-mul-" + fmt.Sprintf("%dx%d", cfg.polyLeftLen, cfg.polyRightLen),
		}, nil
	case "blake2bsum256":
		sampleData := []byte("hello cryptoupgrade")
		return algorithmFixture{
			Algorithm:           "Blake2bSum256",
			UpgradeName:         "Sum256",
			SourcePath:          "experiments/cryptoupgrade/algorithm/go/archive/blake2b.wasm",
			ContractSource:      "experiments/cryptoupgrade/algorithm/contracts/src/archive/Blake2b.sol",
			ContractName:        "Blake2b",
			ContractFunction:    "Sum256",
			UpgradeInputTypes:   []string{"bytes"},
			UpgradeOutputTypes:  []string{"bytes32"},
			ContractInputTypes:  []string{"bytes"},
			ContractOutputTypes: []string{"bytes32"},
			UpgradeValues:       []interface{}{sampleData},
			ContractValues:      []interface{}{sampleData},
			Input: map[string]string{
				"dataHex": hexutil.Encode(sampleData),
			},
			AlgoGas:          3000,
			ResultFileSuffix: "blake2b-sum256",
		}, nil
	default:
		return algorithmFixture{}, fmt.Errorf("unsupported algorithm %q", cfg.algorithm)
	}
}

func prepareScheme(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, fixture algorithmFixture, scheme string) (callTarget, setupReference, error) {
	switch scheme {
	case schemeUpgrade:
		setup := setupReference{Action: "reuse-existing-upload", Address: common.CodeStorageAddress.Hex()}
		if cfg.upload {
			ref, err := uploadAlgorithm(ctx, client, codeStorageABI, from, cfg, fixture)
			if err != nil {
				return callTarget{}, setupReference{}, err
			}
			setup = ref
		}
		target, err := buildUpgradeTarget(codeStorageABI, fixture)
		if err != nil {
			return callTarget{}, setupReference{}, err
		}
		if err := waitCallFuncReady(ctx, client, from, target, cfg.txGas, 60*time.Second); err != nil {
			return callTarget{}, setupReference{}, err
		}
		return target, setup, nil
	case schemeContract:
		contract, err := compileSolidity(cfg.solcPath, cfg.evmVersion, fixture.ContractSource, fixture.ContractName, fixture.ContractFunction)
		if err != nil {
			return callTarget{}, setupReference{}, err
		}
		addr, setup, err := deployFixtureContract(ctx, client, from, contract, cfg.contractDeployGas)
		if err != nil {
			return callTarget{}, setupReference{}, err
		}
		target, err := buildContractTarget(contract, addr, fixture)
		if err != nil {
			return callTarget{}, setupReference{}, err
		}
		return target, setup, nil
	default:
		return callTarget{}, setupReference{}, fmt.Errorf("unsupported scheme %q", scheme)
	}
}

func buildUpgradeTarget(codeStorageABI abi.ABI, fixture algorithmFixture) (callTarget, error) {
	args, err := abiArguments(fixture.UpgradeInputTypes...)
	if err != nil {
		return callTarget{}, err
	}
	encoded, err := args.Pack(fixture.UpgradeValues...)
	if err != nil {
		return callTarget{}, err
	}
	data, err := codeStorageABI.Pack("callFunc", fixture.UpgradeName, encoded)
	if err != nil {
		return callTarget{}, err
	}
	return callTarget{scheme: schemeUpgrade, address: common.CodeStorageAddress, calldata: data}, nil
}

func buildContractTarget(contract compiledContract, addr common.Address, fixture algorithmFixture) (callTarget, error) {
	data, err := contract.ABI.Pack(fixture.ContractFunction, fixture.ContractValues...)
	if err != nil {
		return callTarget{}, err
	}
	return callTarget{scheme: schemeContract, address: addr, calldata: data}, nil
}

func measureThroughput(ctx context.Context, client *rpc.Client, from common.Address, target callTarget, count int, txGasLimit uint64) (throughputRun, error) {
	gasLimit, err := resolveTxGas(ctx, client, from, target, txGasLimit)
	if err != nil {
		return throughputRun{}, err
	}

	started := time.Now()
	hashes := make([]common.Hash, 0, count)
	for i := 0; i < count; i++ {
		gas := hexutil.Uint64(gasLimit)
		txHash, err := sendTransaction(ctx, client, txArgs{
			From: from,
			To:   &target.address,
			Gas:  &gas,
			Data: target.calldata,
		})
		if err != nil {
			return throughputRun{}, fmt.Errorf("send tx %d: %w", i+1, err)
		}
		hashes = append(hashes, txHash)
	}

	var successCount int
	var failedCount int
	var totalGas uint64
	var firstBlock uint64
	var lastBlock uint64
	for i, txHash := range hashes {
		receipt, err := waitReceipt(ctx, client, txHash, 10*time.Minute)
		if err != nil {
			return throughputRun{}, fmt.Errorf("wait receipt %d: %w", i+1, err)
		}
		if receipt.Status == types.ReceiptStatusSuccessful {
			successCount++
			totalGas += receipt.GasUsed
			if firstBlock == 0 || receipt.BlockNumber.Uint64() < firstBlock {
				firstBlock = receipt.BlockNumber.Uint64()
			}
			if receipt.BlockNumber.Uint64() > lastBlock {
				lastBlock = receipt.BlockNumber.Uint64()
			}
		} else {
			failedCount++
		}
	}
	elapsed := time.Since(started).Seconds()
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(successCount) / elapsed
	}
	blocksSpanned := uint64(0)
	if lastBlock >= firstBlock && firstBlock > 0 {
		blocksSpanned = lastBlock - firstBlock + 1
	}
	return throughputRun{
		TxCount:        count,
		SuccessCount:   successCount,
		FailedCount:    failedCount,
		ElapsedSeconds: elapsed,
		ThroughputTPS:  throughput,
		TotalGasUsed:   totalGas,
		FirstBlock:     firstBlock,
		LastBlock:      lastBlock,
		BlocksSpanned:  blocksSpanned,
	}, nil
}

func resolveTxGas(ctx context.Context, client *rpc.Client, from common.Address, target callTarget, override uint64) (uint64, error) {
	if override != 0 {
		return override, nil
	}
	estimated, err := estimateGas(ctx, client, txArgs{From: from, To: &target.address, Data: target.calldata})
	if err != nil {
		return 0, err
	}
	// 留余量，避免 estimate 边界失败。
	return estimated + estimated/5 + 100000, nil
}

func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, fixture algorithmFixture) (setupReference, error) {
	sourceBytes, _, compressed, err := compressSource(ctx, fixture.SourcePath, fixture.UpgradeName, fixture.UpgradeInputTypes, fixture.UpgradeOutputTypes)
	if err != nil {
		return setupReference{}, err
	}
	data, err := codeStorageABI.Pack(
		"uploadCode",
		fixture.UpgradeName,
		compressed,
		fixture.AlgoGas,
		strings.Join(fixture.UpgradeInputTypes, ","),
		strings.Join(fixture.UpgradeOutputTypes, ","),
	)
	if err != nil {
		return setupReference{}, err
	}
	gas := hexutil.Uint64(cfg.uploadGas)
	to := common.CodeStorageAddress
	txHash, err := sendTransaction(ctx, client, txArgs{From: from, To: &to, Gas: &gas, Data: data})
	if err != nil {
		return setupReference{}, err
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return setupReference{}, err
	}
	ref := setupReference{
		Action:        "CodeStorage.uploadCode",
		Address:       common.CodeStorageAddress.Hex(),
		TxHash:        txHash.Hex(),
		ReceiptStatus: receipt.Status,
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return ref, fmt.Errorf("upload transaction failed with status %d", receipt.Status)
	}
	_ = sourceBytes
	return ref, nil
}

func deployFixtureContract(ctx context.Context, client *rpc.Client, from common.Address, contract compiledContract, gasLimit uint64) (common.Address, setupReference, error) {
	args := txArgs{From: from, Data: contract.Bin}
	if gasLimit != 0 {
		gas := hexutil.Uint64(gasLimit)
		args.Gas = &gas
	}
	txHash, err := sendTransaction(ctx, client, args)
	if err != nil {
		return common.Address{}, setupReference{}, err
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return common.Address{}, setupReference{}, err
	}
	ref := setupReference{
		Action:        "deploy-contract",
		Address:       receipt.ContractAddress.Hex(),
		TxHash:        txHash.Hex(),
		ReceiptStatus: receipt.Status,
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return receipt.ContractAddress, ref, fmt.Errorf("contract deployment failed with status %d", receipt.Status)
	}
	return receipt.ContractAddress, ref, nil
}

func validateTarget(ctx context.Context, client *rpc.Client, from common.Address, target callTarget, txGas uint64) error {
	gasLimit, err := resolveTxGas(ctx, client, from, target, txGas)
	if err != nil {
		return err
	}
	gas := hexutil.Uint64(gasLimit)
	var output hexutil.Bytes
	if err := client.CallContext(ctx, &output, "eth_call", txArgs{From: from, To: &target.address, Gas: &gas, Data: target.calldata}, "latest"); err != nil {
		return err
	}
	if len(output) == 0 {
		return errors.New("validation eth_call returned empty output")
	}
	return nil
}

func waitCallFuncReady(ctx context.Context, client *rpc.Client, from common.Address, target callTarget, txGas uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	gasLimit, err := resolveTxGas(ctx, client, from, target, txGas)
	if err != nil {
		return err
	}
	gas := hexutil.Uint64(gasLimit)
	for time.Now().Before(deadline) {
		var output hexutil.Bytes
		if err := client.CallContext(ctx, &output, "eth_call", txArgs{From: from, To: &target.address, Gas: &gas, Data: target.calldata}, "latest"); err == nil && len(output) > 0 {
			return nil
		} else if err != nil {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for upgrade activation: %w", lastErr)
}

func parseTxCounts(raw string) ([]int, error) {
	var counts []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var value int
		if _, err := fmt.Sscanf(part, "%d", &value); err != nil || value <= 0 {
			return nil, fmt.Errorf("invalid tx count %q", part)
		}
		counts = append(counts, value)
	}
	if len(counts) == 0 {
		return nil, errors.New("at least one tx count is required")
	}
	return counts, nil
}

func parseSchemes(raw string) (map[string]bool, error) {
	selected := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		switch part {
		case schemeUpgrade, schemeContract:
			selected[part] = true
		default:
			return nil, fmt.Errorf("unknown scheme %q", part)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("at least one scheme must be selected")
	}
	return selected, nil
}

func compileSolidity(solcPath, evmVersion, sourcePath, contractName, functionName string) (compiledContract, error) {
	args := []string{"--optimize", "--via-ir", "--combined-json", "abi,bin"}
	if evmVersion != "" {
		args = append(args, "--evm-version", evmVersion)
	}
	args = append(args, sourcePath)
	cmd := exec.Command(solcPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return compiledContract{}, fmt.Errorf("compile contract: %w\n%s", err, string(output))
	}
	var combined struct {
		Contracts map[string]struct {
			ABI json.RawMessage `json:"abi"`
			Bin string          `json:"bin"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(output, &combined); err != nil {
		return compiledContract{}, err
	}
	for name, contract := range combined.Contracts {
		shortName := shortContractName(name)
		if shortName != contractName {
			continue
		}
		contractABI, err := abi.JSON(bytes.NewReader(contract.ABI))
		if err != nil {
			return compiledContract{}, err
		}
		bin, err := hexutil.Decode("0x" + contract.Bin)
		if err != nil {
			return compiledContract{}, err
		}
		if _, ok := contractABI.Methods[functionName]; !ok {
			return compiledContract{}, fmt.Errorf("function %q not found", functionName)
		}
		return compiledContract{Contract: shortName, ABI: contractABI, Bin: bin}, nil
	}
	return compiledContract{}, fmt.Errorf("contract %q not found in %s", contractName, sourcePath)
}

func compressSource(ctx context.Context, path, function string, inputTypes, outputTypes []string) (int, string, string, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return 0, "", "", err
	}
	wasm, encoded, err := wasmtool.BuildEncodedPath(ctx, path, wasmtool.Spec{
		Function:    function,
		InputTypes:  inputTypes,
		OutputTypes: outputTypes,
	})
	if err != nil {
		return 0, "", "", err
	}
	return len(source), wasmtool.WasmHash(wasm), encoded, nil
}

func abiArguments(types ...string) (abi.Arguments, error) {
	args := make(abi.Arguments, 0, len(types))
	for _, raw := range types {
		typ, err := abi.NewType(raw, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Type: typ})
	}
	return args, nil
}

func buildPolynomialCoeffs(length int, modulus *big.Int, coefMax int) []*big.Int {
	coeffs := make([]*big.Int, length)
	for i := range coeffs {
		var value int64
		if coefMax > 0 {
			value = int64(i%coefMax + 1)
		} else {
			value = int64(i + 1)
		}
		coeffs[i] = new(big.Int).Mod(big.NewInt(value), modulus)
		if coeffs[i].Sign() == 0 {
			coeffs[i].SetInt64(1)
		}
	}
	return coeffs
}

func polynomialMulAlgoGas(leftLen, rightLen int) uint64 {
	products := uint64(leftLen * rightLen)
	return 5000 + products*200
}

func parsePositiveBigInt(label, raw string) (*big.Int, error) {
	value, ok := new(big.Int).SetString(strings.TrimSpace(raw), 10)
	if !ok {
		return nil, fmt.Errorf("invalid %s %q", label, raw)
	}
	if value.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be positive", label)
	}
	return value, nil
}

func resolveSender(ctx context.Context, client *rpc.Client, raw string) (common.Address, error) {
	if raw != "" {
		return common.HexToAddress(raw), nil
	}
	var accounts []common.Address
	if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
		return common.Address{}, err
	}
	if len(accounts) == 0 {
		return common.Address{}, errors.New("eth_accounts returned no unlocked accounts")
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
	for {
		var receipt *types.Receipt
		if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "transaction indexing is in progress") {
				if time.Now().After(deadline) {
					return nil, fmt.Errorf("timed out waiting for receipt %s", txHash.Hex())
				}
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return nil, err
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

func estimateGas(ctx context.Context, client *rpc.Client, args txArgs) (uint64, error) {
	var gas hexutil.Uint64
	if err := client.CallContext(ctx, &gas, "eth_estimateGas", args); err != nil {
		return 0, err
	}
	return uint64(gas), nil
}

func writeJSONResult(path string, res suiteResult) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func printSummary(res suiteResult) {
	fmt.Printf("Throughput benchmark (%s)\n", res.Algorithm)
	fmt.Printf("  rpc:    %s\n", res.RPC)
	fmt.Printf("  input:  %s\n", formatInput(res.Input))
	fmt.Println()
	for _, scheme := range res.Results {
		fmt.Printf("%s (%s)\n", scheme.Scheme, scheme.Setup.Address)
		for _, run := range scheme.Runs {
			fmt.Printf("  n=%2d  success=%d  elapsed=%6.2fs  throughput=%7.3f tx/s  blocks=%d\n",
				run.TxCount, run.SuccessCount, run.ElapsedSeconds, run.ThroughputTPS, run.BlocksSpanned)
		}
	}
	fmt.Printf("\nJSON result: %s\n", res.OutputJSON)
}

func shortContractName(name string) string {
	if i := strings.LastIndex(name, ":"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func formatInput(input map[string]string) string {
	keys := make([]string, 0, len(input))
	for key := range input {
		if key == "txCounts" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+input[key])
	}
	return strings.Join(parts, ", ")
}

func algorithmKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
}

func defaultSolcPath() string {
	candidates := []string{
		"/home/lq/.local/share/svm/solc-0.8.26",
		"../.tools/solc/solc-0.8.26",
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
