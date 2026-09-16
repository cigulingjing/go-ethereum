// benchtxgas 在单节点网络中测量 WASM 动态升级方案与 Solidity 合约方案的真实交易 Gas。
// 每个算法采集四项 receipt gasUsed：WASM 升级交易（CodeStorage.uploadCode）、
// Solidity 部署交易、WASM 执行交易（CodeStorage.callFunc）、Solidity 执行交易（合约调用）。
// 执行交易前先用 eth_call 校验两条路径输出一致。
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	experimentName = "tx-gas-comparison"

	rfc3526PrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
		"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
		"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
		"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
		"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"
)

type config struct {
	rpc          string
	algorithms   string
	from         string
	a            string
	b            string
	algoGas      uint64
	uploadGas    uint64
	callGas      uint64
	deployGas    uint64
	solcPath     string
	evmVersion   string
	outputDir    string
	outputJSON   string
	polyLeftLen  int
	polyRightLen int
	polyProfile  string
	polyModulus  string
	polyCoefMax  int
}

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

// fixture 描述一个可对比算法：WASM 模块与 Solidity 合约使用同一组 ABI 类型和逻辑输入。
type fixture struct {
	Algorithm        string
	UpgradeName      string
	SourcePath       string
	ContractSource   string
	ContractName     string
	ContractFunction string
	InputTypes       []string
	OutputTypes      []string
	Values           []interface{}
	Input            map[string]string
	AlgoGas          uint64
}

type compiledContract struct {
	Contract string
	ABI      abi.ABI
	Bin      []byte
}

// txMeasurement 记录一笔真实交易 receipt 中的 Gas 数据。
type txMeasurement struct {
	TxHash            string `json:"txHash"`
	BlockNumber       uint64 `json:"blockNumber"`
	Status            uint64 `json:"status"`
	GasUsed           uint64 `json:"gasUsed"`
	EffectiveGasPrice string `json:"effectiveGasPrice"`
	CalldataBytes     int    `json:"calldataBytes,omitempty"`
	Note              string `json:"note,omitempty"`
}

type upgradeMeasurement struct {
	txMeasurement
	Function              string `json:"function"`
	SourcePath            string `json:"sourcePath"`
	WasmHash              string `json:"wasmHash,omitempty"`
	SourceBytes           int    `json:"sourceBytes,omitempty"`
	CompressedBase64Bytes int    `json:"compressedBase64Bytes,omitempty"`
	Contract              string `json:"contract,omitempty"`
	ContractAddress       string `json:"contractAddress,omitempty"`
	ContractBinBytes      int    `json:"contractBinBytes,omitempty"`
}

type algorithmResult struct {
	Algorithm         string             `json:"algorithm"`
	UpgradeName       string             `json:"upgradeName"`
	Input             map[string]string  `json:"input"`
	ABIInputTypes     []string           `json:"abiInputTypes"`
	ABIOutputTypes    []string           `json:"abiOutputTypes"`
	WasmUpgrade       upgradeMeasurement `json:"wasmUpgrade"`
	ContractUpgrade   upgradeMeasurement `json:"contractUpgrade"`
	WasmExecution     txMeasurement      `json:"wasmExecution"`
	ContractExecution txMeasurement      `json:"contractExecution"`
	OutputMatched     bool               `json:"outputMatched"`
	OutputValue       string             `json:"outputValue"`
}

type suiteResult struct {
	Experiment          string            `json:"experiment"`
	Timestamp           string            `json:"timestamp"`
	RPC                 string            `json:"rpc"`
	ChainID             uint64            `json:"chainId"`
	Sender              string            `json:"sender"`
	Command             []string          `json:"command"`
	Solc                string            `json:"solc"`
	EvmVersion          string            `json:"evmVersion"`
	RequestedAlgorithms []string          `json:"requestedAlgorithms"`
	Results             []algorithmResult `json:"results"`
	OutputJSON          string            `json:"outputJson"`
	Notes               []string          `json:"notes"`
}

type decodedOutput struct {
	canonical string
	display   string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchtxgas: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		return err
	}
	fixtures, err := buildFixtures(cfg)
	if err != nil {
		return err
	}
	selected, requested, err := selectFixtures(fixtures, cfg)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return errors.New("no algorithms selected")
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
	chainID, err := queryChainID(ctx, client)
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
		outputJSON = filepath.Join(cfg.outputDir, "single-node-"+now.Format("20060102-150405"), "result.json")
	}

	suite := suiteResult{
		Experiment:          experimentName,
		Timestamp:           now.Format(time.RFC3339Nano),
		RPC:                 cfg.rpc,
		ChainID:             chainID,
		Sender:              from.Hex(),
		Command:             os.Args,
		Solc:                cfg.solcPath,
		EvmVersion:          cfg.evmVersion,
		RequestedAlgorithms: requested,
		OutputJSON:          outputJSON,
		Notes: []string{
			"all gas values are receipt gasUsed of real transactions on a single-node Clique chain, not eth_estimateGas",
			"wasmUpgrade = CodeStorage.uploadCode transaction; contractUpgrade = Solidity deployment transaction",
			"wasmExecution = CodeStorage.callFunc transaction; contractExecution = Solidity contract call transaction",
			"execution outputs are validated identical via eth_call before sending execution transactions",
		},
	}

	for _, fx := range selected {
		fmt.Printf("measuring %s...\n", fx.Algorithm)
		result, err := runFixture(ctx, client, codeStorageABI, from, cfg, fx)
		if err != nil {
			return fmt.Errorf("%s: %w", fx.Algorithm, err)
		}
		suite.Results = append(suite.Results, result)
	}

	if err := writeJSONResult(outputJSON, suite); err != nil {
		return err
	}
	printSummary(suite)
	return nil
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.rpc, "rpc", "http://127.0.0.1:8761", "JSON-RPC endpoint")
	flag.StringVar(&cfg.algorithms, "algorithms", "all", "comma-separated algorithm list or all")
	flag.StringVar(&cfg.from, "from", "", "sender address; defaults to eth_accounts[0]")
	flag.StringVar(&cfg.a, "a", "100", "first Add operand, non-negative decimal integer")
	flag.StringVar(&cfg.b, "b", "100", "second Add operand, non-negative decimal integer")
	flag.Uint64Var(&cfg.algoGas, "algo-gas", 0, "override algorithm gas metadata recorded by CodeStorage.uploadCode; 0 uses fixture defaults")
	flag.Uint64Var(&cfg.uploadGas, "upload-gas", 0, "gas limit for CodeStorage.uploadCode; 0 lets the node estimate")
	flag.Uint64Var(&cfg.deployGas, "deploy-gas", 0, "gas limit for contract deployment; 0 lets the node estimate")
	flag.Uint64Var(&cfg.callGas, "call-gas", 0, "gas limit for execution transactions; 0 lets the node estimate")
	flag.StringVar(&cfg.solcPath, "solc", defaultSolcPath(), "solc compiler path")
	flag.StringVar(&cfg.evmVersion, "evm-version", "paris", "solc EVM target")
	flag.StringVar(&cfg.outputDir, "output-dir", "experiments/cryptoupgrade/results/tx-gas", "directory used when -output-json is empty")
	flag.StringVar(&cfg.outputJSON, "output-json", "", "write JSON result to this file; defaults under -output-dir")
	flag.IntVar(&cfg.polyLeftLen, "poly-left-len", 4, "PolynomialMul left polynomial length")
	flag.IntVar(&cfg.polyRightLen, "poly-right-len", 4, "PolynomialMul right polynomial length")
	flag.StringVar(&cfg.polyProfile, "poly-profile", "", "PolynomialMul profile preset (P1..P6); overrides -poly-left-len/-poly-right-len")
	flag.StringVar(&cfg.polyModulus, "poly-modulus", "12289", "PolynomialMul modulus")
	flag.IntVar(&cfg.polyCoefMax, "poly-coef-max", 0, "PolynomialMul coefficient upper bound; when >0 use values in [1,max] cyclically")
	flag.Parse()
	return cfg
}

func (cfg *config) validate() error {
	if cfg.rpc == "" {
		return errors.New("-rpc is required")
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return fmt.Errorf("invalid -from %q", cfg.from)
	}
	if cfg.solcPath == "" {
		return errors.New("-solc is required")
	}
	if cfg.polyProfile != "" {
		size, ok := polyProfileSize(cfg.polyProfile)
		if !ok {
			return fmt.Errorf("unknown -poly-profile %q (supported: P1, P2, P3, P4, P5, P6)", cfg.polyProfile)
		}
		cfg.polyLeftLen = size
		cfg.polyRightLen = size
	}
	return nil
}

func buildFixtures(cfg config) ([]fixture, error) {
	a, err := parseAddOperand("a", cfg.a)
	if err != nil {
		return nil, err
	}
	b, err := parseAddOperand("b", cfg.b)
	if err != nil {
		return nil, err
	}
	expectedAdd := new(big.Int).Add(a, b)
	if expectedAdd.Cmp(maxInt256()) > 0 {
		return nil, fmt.Errorf("a+b exceeds int256 range: %s", expectedAdd)
	}

	sampleData := []byte("hello cryptoupgrade")
	pbkdf2Password := []byte("password")
	pbkdf2Salt := []byte("salt")
	pbkdf2Iterations := big.NewInt(2)
	pbkdf2KeyLength := big.NewInt(32)
	dhPrivate := []byte{0x05}
	dhPeerPublic := dh2048PeerPublic(7)
	pedersenMessage := []byte{0x07}
	pedersenBlinding := []byte{0x0b}
	schnorrMessage := []byte("cryptoupgrade schnorr benchmark")
	schnorrProof := deterministicSchnorrProof(schnorrMessage)

	polyModulus, err := parsePositiveBigInt("poly-modulus", cfg.polyModulus)
	if err != nil {
		return nil, err
	}
	if cfg.polyLeftLen <= 0 || cfg.polyRightLen <= 0 {
		return nil, fmt.Errorf("poly-left-len and poly-right-len must be positive")
	}
	polyLeft := buildPolynomialCoeffs(cfg.polyLeftLen, polyModulus, cfg.polyCoefMax)
	polyRight := buildPolynomialCoeffs(cfg.polyRightLen, polyModulus, cfg.polyCoefMax)
	polyExpected := polynomialMulReference(polyLeft, polyRight, polyModulus)

	return []fixture{
		{
			Algorithm:        "Add",
			UpgradeName:      "Add",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/add.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/Add.sol",
			ContractName:     "AddContract",
			ContractFunction: "Add",
			InputTypes:       []string{"int256", "int256"},
			OutputTypes:      []string{"int256"},
			Values:           []interface{}{a, b},
			Input: map[string]string{
				"a":        a.String(),
				"b":        b.String(),
				"expected": expectedAdd.String(),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:        "Sha256",
			UpgradeName:      "Sha256",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/sha256.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/Sha256.sol",
			ContractName:     "Sha256Contract",
			ContractFunction: "Sha256",
			InputTypes:       []string{"bytes"},
			OutputTypes:      []string{"bytes"},
			Values:           []interface{}{sampleData},
			Input: map[string]string{
				"dataHex": hexutil.Encode(sampleData),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:        "Blake2bSum256",
			UpgradeName:      "Sum256",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/blake2b.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/Blake2b.sol",
			ContractName:     "Blake2b",
			ContractFunction: "Sum256",
			InputTypes:       []string{"bytes"},
			OutputTypes:      []string{"bytes32"},
			Values:           []interface{}{sampleData},
			Input: map[string]string{
				"dataHex": hexutil.Encode(sampleData),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:        "Pbkdf2Sha256",
			UpgradeName:      "Pbkdf2Sha256",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/pbkdf2_sha256.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/Pbkdf2Sha256.sol",
			ContractName:     "Pbkdf2Sha256Contract",
			ContractFunction: "Pbkdf2Sha256",
			InputTypes:       []string{"bytes", "bytes", "uint256", "uint256"},
			OutputTypes:      []string{"bytes"},
			Values:           []interface{}{pbkdf2Password, pbkdf2Salt, pbkdf2Iterations, pbkdf2KeyLength},
			Input: map[string]string{
				"passwordHex": hexutil.Encode(pbkdf2Password),
				"saltHex":     hexutil.Encode(pbkdf2Salt),
				"iterations":  pbkdf2Iterations.String(),
				"keyLength":   pbkdf2KeyLength.String(),
			},
			AlgoGas: 25000,
		},
		{
			Algorithm:        "Dh2048Secret",
			UpgradeName:      "Dh2048Secret",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/dh2048.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/Dh2048.sol",
			ContractName:     "Dh2048",
			ContractFunction: "Dh2048Secret",
			InputTypes:       []string{"bytes", "bytes"},
			OutputTypes:      []string{"bytes"},
			Values:           []interface{}{dhPrivate, dhPeerPublic},
			Input: map[string]string{
				"privateKeyHex":        hexutil.Encode(dhPrivate),
				"peerPublicKeyHex":     hexutil.Encode(dhPeerPublic),
				"peerPublicKeySha256":  sha256Hex(dhPeerPublic),
				"peerPublicKeyByteLen": fmt.Sprintf("%d", len(dhPeerPublic)),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:        "PedersenCommit",
			UpgradeName:      "PedersenCommit",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/pedersen_commit.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/PedersenCommit.sol",
			ContractName:     "PedersenCommitContract",
			ContractFunction: "PedersenCommit",
			InputTypes:       []string{"bytes", "bytes"},
			OutputTypes:      []string{"bytes"},
			Values:           []interface{}{pedersenMessage, pedersenBlinding},
			Input: map[string]string{
				"messageHex":  hexutil.Encode(pedersenMessage),
				"blindingHex": hexutil.Encode(pedersenBlinding),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:        "SchnorrVerify",
			UpgradeName:      "SchnorrVerify",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/schnorr_proof.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/SchnorrProof.sol",
			ContractName:     "SchnorrProof",
			ContractFunction: "SchnorrVerify",
			InputTypes:       []string{"bytes", "bytes"},
			OutputTypes:      []string{"bool"},
			Values:           []interface{}{schnorrMessage, schnorrProof},
			Input: map[string]string{
				"messageHex":      hexutil.Encode(schnorrMessage),
				"proofHex":        hexutil.Encode(schnorrProof),
				"proofSha256":     sha256Hex(schnorrProof),
				"proofByteLength": fmt.Sprintf("%d", len(schnorrProof)),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:        "PolynomialMul",
			UpgradeName:      "PolynomialMul",
			SourcePath:       "experiments/cryptoupgrade/algorithm/go/wasm/polynomial_mul.wasm",
			ContractSource:   "experiments/cryptoupgrade/algorithm/contracts/src/PolynomialMul.sol",
			ContractName:     "PolynomialMulContract",
			ContractFunction: "PolynomialMul",
			InputTypes:       []string{"uint256[]", "uint256[]", "uint256"},
			OutputTypes:      []string{"uint256[]"},
			Values:           []interface{}{polyLeft, polyRight, polyModulus},
			Input: map[string]string{
				"profile":        polyProfileName(cfg.polyLeftLen, cfg.polyRightLen),
				"leftLen":        fmt.Sprintf("%d", cfg.polyLeftLen),
				"rightLen":       fmt.Sprintf("%d", cfg.polyRightLen),
				"mulIterations":  fmt.Sprintf("%d", cfg.polyLeftLen*cfg.polyRightLen),
				"modulus":        polyModulus.String(),
				"coefMax":        fmt.Sprintf("%d", cfg.polyCoefMax),
				"expectedPrefix": bigIntSliceDisplay(polyExpected[:min(3, len(polyExpected))]),
			},
			AlgoGas: polynomialMulAlgoGas(cfg.polyLeftLen, cfg.polyRightLen),
		},
	}, nil
}

func selectFixtures(fixtures []fixture, cfg config) ([]fixture, []string, error) {
	requested := splitAlgorithmList(cfg.algorithms)
	if len(requested) == 0 || (len(requested) == 1 && algorithmKey(requested[0]) == "all") {
		names := make([]string, 0, len(fixtures))
		for _, fx := range fixtures {
			names = append(names, fx.Algorithm)
		}
		return fixtures, names, nil
	}
	requestedSet := make(map[string]string, len(requested))
	for _, name := range requested {
		requestedSet[algorithmKey(name)] = name
	}
	var selected []fixture
	for _, fx := range fixtures {
		if _, ok := requestedSet[algorithmKey(fx.Algorithm)]; ok {
			selected = append(selected, fx)
			delete(requestedSet, algorithmKey(fx.Algorithm))
		}
	}
	if len(requestedSet) > 0 {
		var unknown []string
		for _, name := range requestedSet {
			unknown = append(unknown, name)
		}
		sort.Strings(unknown)
		return nil, nil, fmt.Errorf("unknown algorithms: %s", strings.Join(unknown, ", "))
	}
	return selected, requested, nil
}

func runFixture(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, fx fixture) (algorithmResult, error) {
	result := algorithmResult{
		Algorithm:      fx.Algorithm,
		UpgradeName:    fx.UpgradeName,
		Input:          fx.Input,
		ABIInputTypes:  append([]string(nil), fx.InputTypes...),
		ABIOutputTypes: append([]string(nil), fx.OutputTypes...),
	}

	// 1. WASM 升级交易：CodeStorage.uploadCode
	wasmUpgrade, err := uploadAlgorithm(ctx, client, codeStorageABI, from, cfg, fx)
	if err != nil {
		return result, err
	}
	result.WasmUpgrade = wasmUpgrade

	// 2. 构造两条路径的 calldata，等待 WASM 异步激活完成
	callArgs, err := abiArguments(fx.InputTypes...)
	if err != nil {
		return result, err
	}
	returnArgs, err := abiArguments(fx.OutputTypes...)
	if err != nil {
		return result, err
	}
	encodedInput, err := callArgs.Pack(fx.Values...)
	if err != nil {
		return result, fmt.Errorf("pack %s input: %w", fx.Algorithm, err)
	}
	wasmCallData, err := codeStorageABI.Pack("callFunc", fx.UpgradeName, encodedInput)
	if err != nil {
		return result, fmt.Errorf("pack CodeStorage.callFunc: %w", err)
	}
	if err := waitCallFuncReady(ctx, client, from, wasmCallData, 120*time.Second); err != nil {
		return result, err
	}

	// 3. Solidity 升级交易：编译并部署合约
	contract, err := compileSolidity(cfg.solcPath, cfg.evmVersion, fx.ContractSource, fx.ContractName, fx.ContractFunction)
	if err != nil {
		return result, err
	}
	contractAddress, contractUpgrade, err := deployFixtureContract(ctx, client, from, contract, fx, cfg.deployGas)
	if err != nil {
		return result, err
	}
	result.ContractUpgrade = contractUpgrade

	contractCallData, err := contract.ABI.Pack(fx.ContractFunction, fx.Values...)
	if err != nil {
		return result, fmt.Errorf("pack %s contract call: %w", fx.Algorithm, err)
	}

	// 4. eth_call 校验两条路径输出一致
	wasmOutput, err := ethCall(ctx, client, txArgs{From: from, To: ptrAddress(common.CodeStorageAddress), Data: wasmCallData})
	if err != nil {
		return result, fmt.Errorf("eth_call wasm path: %w", err)
	}
	wasmDecoded, err := decodeSingleOutput(returnArgs, wasmOutput)
	if err != nil {
		return result, fmt.Errorf("decode wasm output: %w", err)
	}
	contractOutput, err := ethCall(ctx, client, txArgs{From: from, To: &contractAddress, Data: contractCallData})
	if err != nil {
		return result, fmt.Errorf("eth_call contract path: %w", err)
	}
	contractDecoded, err := decodeSingleOutput(returnArgs, contractOutput)
	if err != nil {
		return result, fmt.Errorf("decode contract output: %w", err)
	}
	if wasmDecoded.canonical != contractDecoded.canonical {
		return result, fmt.Errorf("output mismatch: wasm=%s contract=%s", wasmDecoded.display, contractDecoded.display)
	}
	result.OutputMatched = true
	result.OutputValue = wasmDecoded.display

	// 5. 真实执行交易：CodeStorage.callFunc
	wasmExec, _, err := sendMeasuredTx(ctx, client, from, ptrAddress(common.CodeStorageAddress), wasmCallData, cfg.callGas)
	if err != nil {
		return result, fmt.Errorf("wasm execution transaction: %w", err)
	}
	result.WasmExecution = wasmExec

	// 6. 真实执行交易：Solidity 合约调用
	contractExec, _, err := sendMeasuredTx(ctx, client, from, &contractAddress, contractCallData, cfg.callGas)
	if err != nil {
		return result, fmt.Errorf("contract execution transaction: %w", err)
	}
	result.ContractExecution = contractExec

	return result, nil
}

// uploadAlgorithm 发送 CodeStorage.uploadCode 真实交易并记录 receipt gasUsed。
func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, fx fixture) (upgradeMeasurement, error) {
	sourceBytes, wasmHash, compressed, err := compressSource(ctx, fx.SourcePath, fx.UpgradeName, fx.InputTypes, fx.OutputTypes)
	if err != nil {
		return upgradeMeasurement{}, err
	}
	algoGas := fx.AlgoGas
	if cfg.algoGas != 0 {
		algoGas = cfg.algoGas
	}
	data, err := codeStorageABI.Pack(
		"uploadCode",
		fx.UpgradeName,
		compressed,
		algoGas,
		strings.Join(fx.InputTypes, ","),
		strings.Join(fx.OutputTypes, ","),
	)
	if err != nil {
		return upgradeMeasurement{}, fmt.Errorf("pack CodeStorage.uploadCode: %w", err)
	}
	m, _, err := sendMeasuredTx(ctx, client, from, ptrAddress(common.CodeStorageAddress), data, cfg.uploadGas)
	if err != nil {
		return upgradeMeasurement{}, fmt.Errorf("upload transaction: %w", err)
	}
	return upgradeMeasurement{
		txMeasurement:         m,
		Function:              fx.UpgradeName,
		SourcePath:            mustAbs(fx.SourcePath),
		WasmHash:              wasmHash,
		SourceBytes:           sourceBytes,
		CompressedBase64Bytes: len(compressed),
	}, nil
}

// deployFixtureContract 编译并部署 Solidity 合约，记录部署交易 receipt gasUsed。
func deployFixtureContract(ctx context.Context, client *rpc.Client, from common.Address, contract compiledContract, fx fixture, gasLimit uint64) (common.Address, upgradeMeasurement, error) {
	m, receipt, err := sendMeasuredTx(ctx, client, from, nil, contract.Bin, gasLimit)
	if err != nil {
		return common.Address{}, upgradeMeasurement{}, fmt.Errorf("deploy transaction: %w", err)
	}
	addr := receipt.ContractAddress
	if addr == (common.Address{}) {
		return common.Address{}, upgradeMeasurement{}, fmt.Errorf("deployment %s returned empty contract address", m.TxHash)
	}
	return addr, upgradeMeasurement{
		txMeasurement:    m,
		Function:         fx.ContractFunction,
		SourcePath:       mustAbs(fx.ContractSource),
		Contract:         contract.Contract,
		ContractAddress:  addr.Hex(),
		ContractBinBytes: len(contract.Bin),
	}, nil
}

// sendMeasuredTx 发送真实交易并等待 receipt，返回 Gas 测量值；receipt 失败则报错。
func sendMeasuredTx(ctx context.Context, client *rpc.Client, from common.Address, to *common.Address, data []byte, gasLimit uint64) (txMeasurement, *types.Receipt, error) {
	args := txArgs{From: from, To: to, Data: data}
	if gasLimit != 0 {
		gas := hexutil.Uint64(gasLimit)
		args.Gas = &gas
	}
	var txHash common.Hash
	if err := client.CallContext(ctx, &txHash, "eth_sendTransaction", args); err != nil {
		return txMeasurement{}, nil, err
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return txMeasurement{}, nil, err
	}
	m := txMeasurement{
		TxHash:        txHash.Hex(),
		Status:        receipt.Status,
		GasUsed:       receipt.GasUsed,
		CalldataBytes: len(data),
	}
	if receipt.BlockNumber != nil {
		m.BlockNumber = receipt.BlockNumber.Uint64()
	}
	if receipt.EffectiveGasPrice != nil {
		m.EffectiveGasPrice = receipt.EffectiveGasPrice.String()
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return m, receipt, fmt.Errorf("transaction %s failed with receipt status %d", txHash.Hex(), receipt.Status)
	}
	return m, receipt, nil
}

// waitCallFuncReady 轮询 eth_call，等待 uploadCode 的异步激活完成。
func waitCallFuncReady(ctx context.Context, client *rpc.Client, from common.Address, callData []byte, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, err := ethCall(ctx, client, txArgs{From: from, To: ptrAddress(common.CodeStorageAddress), Data: callData}); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for event-driven activation: %w", lastErr)
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
		return compiledContract{}, fmt.Errorf("decode solc output: %w", err)
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
			return compiledContract{}, fmt.Errorf("decode ABI for %s: %w", contractName, err)
		}
		bin, err := hex.DecodeString(contract.Bin)
		if err != nil {
			return compiledContract{}, fmt.Errorf("decode bytecode for %s: %w", contractName, err)
		}
		if len(bin) == 0 {
			return compiledContract{}, fmt.Errorf("contract bytecode for %s is empty", contractName)
		}
		if _, ok := contractABI.Methods[functionName]; !ok {
			return compiledContract{}, fmt.Errorf("function %q not found in contract %s", functionName, contractName)
		}
		return compiledContract{Contract: shortName, ABI: contractABI, Bin: bin}, nil
	}
	sort.Strings(available)
	return compiledContract{}, fmt.Errorf("contract %q not found in %s; available contracts: %s", contractName, sourcePath, strings.Join(available, ", "))
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

func decodeSingleOutput(args abi.Arguments, output []byte) (decodedOutput, error) {
	values, err := args.Unpack(output)
	if err != nil {
		return decodedOutput{}, err
	}
	if len(values) != 1 {
		return decodedOutput{}, fmt.Errorf("expected one output value, got %d", len(values))
	}
	value := values[0]
	switch v := value.(type) {
	case *big.Int:
		return decodedOutput{canonical: "number:" + v.String(), display: v.String()}, nil
	case bool:
		return decodedOutput{canonical: fmt.Sprintf("bool:%t", v), display: fmt.Sprintf("%t", v)}, nil
	case []byte:
		encoded := hexutil.Encode(v)
		return decodedOutput{canonical: "bytes:" + encoded, display: encoded}, nil
	case [32]byte:
		encoded := hexutil.Encode(v[:])
		return decodedOutput{canonical: "bytes:" + encoded, display: encoded}, nil
	case []*big.Int:
		return decodedBigIntSlice(v), nil
	}
	if b, ok := fixedByteArray(value); ok {
		encoded := hexutil.Encode(b)
		return decodedOutput{canonical: "bytes:" + encoded, display: encoded}, nil
	}
	return decodedOutput{}, fmt.Errorf("unsupported output type %T", value)
}

func fixedByteArray(value interface{}) ([]byte, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Array || rv.Type().Elem().Kind() != reflect.Uint8 {
		return nil, false
	}
	out := make([]byte, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = byte(rv.Index(i).Uint())
	}
	return out, true
}

func decodedBigIntSlice(values []*big.Int) decodedOutput {
	return decodedOutput{
		canonical: bigIntSliceCanonical(values),
		display:   bigIntSliceDisplay(values),
	}
}

func bigIntSliceCanonical(values []*big.Int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		if value == nil {
			parts[i] = "<nil>"
			continue
		}
		parts[i] = value.String()
	}
	return "uint256[]:" + strings.Join(parts, ",")
}

func bigIntSliceDisplay(values []*big.Int) string {
	if len(values) == 0 {
		return "[]"
	}
	if len(values) <= 4 {
		return "[" + strings.Join(strings.Fields(strings.TrimPrefix(bigIntSliceCanonical(values), "uint256[]:")), ", ") + "]"
	}
	prefix := bigIntSliceDisplay(values[:3])
	return strings.TrimSuffix(prefix, "]") + ", ...(" + fmt.Sprintf("%d", len(values)) + ")]"
}

func abiArguments(types ...string) (abi.Arguments, error) {
	args := make(abi.Arguments, 0, len(types))
	for _, raw := range types {
		typ, err := abi.NewType(raw, "", nil)
		if err != nil {
			return nil, fmt.Errorf("ABI type %q: %w", raw, err)
		}
		args = append(args, abi.Argument{Type: typ})
	}
	return args, nil
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
		return common.Address{}, errors.New("eth_accounts returned no unlocked accounts; pass -from")
	}
	return accounts[0], nil
}

func queryChainID(ctx context.Context, client *rpc.Client) (uint64, error) {
	var id hexutil.Big
	if err := client.CallContext(ctx, &id, "eth_chainId"); err != nil {
		return 0, fmt.Errorf("eth_chainId: %w", err)
	}
	return (*big.Int)(&id).Uint64(), nil
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

func ethCall(ctx context.Context, client *rpc.Client, args txArgs) ([]byte, error) {
	var output hexutil.Bytes
	if err := client.CallContext(ctx, &output, "eth_call", args, "latest"); err != nil {
		return nil, err
	}
	return output, nil
}

func writeJSONResult(path string, res suiteResult) error {
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
		return fmt.Errorf("write JSON result: %w", err)
	}
	return nil
}

func printSummary(res suiteResult) {
	fmt.Println("Transaction gas comparison (single-node, receipt gasUsed)")
	fmt.Printf("  rpc:     %s (chainId=%d)\n", res.RPC, res.ChainID)
	fmt.Printf("  sender:  %s\n", res.Sender)
	fmt.Println()
	fmt.Printf("%-16s %14s %14s %14s %14s %8s\n", "Algorithm", "WasmUpgrade", "SolUpgrade", "WasmExec", "SolExec", "Matched")
	for _, r := range res.Results {
		fmt.Printf("%-16s %14d %14d %14d %14d %8t\n",
			r.Algorithm,
			r.WasmUpgrade.GasUsed,
			r.ContractUpgrade.GasUsed,
			r.WasmExecution.GasUsed,
			r.ContractExecution.GasUsed,
			r.OutputMatched,
		)
	}
	fmt.Printf("\nJSON result: %s\n", res.OutputJSON)
}

func parseAddOperand(label, raw string) (*big.Int, error) {
	value, ok := new(big.Int).SetString(strings.TrimSpace(raw), 10)
	if !ok {
		return nil, fmt.Errorf("invalid %s operand %q", label, raw)
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("%s must be non-negative for Add cross-implementation comparison", label)
	}
	if value.Cmp(maxInt256()) > 0 {
		return nil, fmt.Errorf("%s exceeds int256 range: %s", label, value)
	}
	return value, nil
}

func maxInt256() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 255)
	return limit.Sub(limit, big.NewInt(1))
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

func splitAlgorithmList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func algorithmKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
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

func ptrAddress(addr common.Address) *common.Address {
	return &addr
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hexutil.Encode(sum[:])
}

func dh2048PeerPublic(exponent int64) []byte {
	p := rfc3526Prime()
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(exponent), p)
	return fixedBytes(y, rfc3526FieldBytes())
}

func deterministicSchnorrProof(message []byte) []byte {
	p, q := rfc3526Subgroup()
	g := big.NewInt(4)
	x := big.NewInt(17)
	k := big.NewInt(23)
	y := new(big.Int).Exp(g, x, p)
	t := new(big.Int).Exp(g, k, p)
	c := schnorrChallenge(q, y, t, message)
	s := new(big.Int).Mul(c, x)
	s.Add(s, k)
	s.Mod(s, q)
	size := rfc3526FieldBytes()
	proof := make([]byte, 0, 3*size)
	proof = append(proof, fixedBytes(y, size)...)
	proof = append(proof, fixedBytes(t, size)...)
	proof = append(proof, fixedBytes(s, size)...)
	return proof
}

func schnorrChallenge(q, y, t *big.Int, message []byte) *big.Int {
	h := sha256.New()
	h.Write([]byte("cryptoupgrade-schnorr"))
	h.Write(fixedBytes(y, rfc3526FieldBytes()))
	h.Write(fixedBytes(t, rfc3526FieldBytes()))
	h.Write(message)
	c := new(big.Int).SetBytes(h.Sum(nil))
	c.Mod(c, q)
	return c
}

func rfc3526Prime() *big.Int {
	p, _ := new(big.Int).SetString(rfc3526PrimeHex, 16)
	return p
}

func rfc3526Subgroup() (*big.Int, *big.Int) {
	p := rfc3526Prime()
	q := new(big.Int).Sub(p, big.NewInt(1))
	q.Rsh(q, 1)
	return p, q
}

func rfc3526FieldBytes() int {
	return (len(rfc3526PrimeHex) + 1) / 2
}

func fixedBytes(x *big.Int, size int) []byte {
	if x == nil {
		return nil
	}
	b := x.Bytes()
	if len(b) > size {
		return b[len(b)-size:]
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

func polynomialMulAlgoGas(leftLen, rightLen int) uint64 {
	products := uint64(leftLen * rightLen)
	// 与 precompile 的 step gas 同量级，避免大负载下 callFunc 因 metadata gas 不足失败。
	return 5000 + products*200
}

// polyProfileSizeMap 定义 PolynomialMul 实验档位，与 benchexecutionefficiency 保持一致。
var polyProfileSizeMap = map[string]int{
	"P1": 4,  // 4×4=16
	"P2": 6,  // 6×6=36
	"P3": 8,  // 8×8=64
	"P4": 9,  // 9×9=81
	"P5": 10, // 10×10=100
	"P6": 11, // 11×11=121；12×12 在 TinyGo WASM 下返回空结果
}

func polyProfileSize(name string) (int, bool) {
	size, ok := polyProfileSizeMap[strings.ToUpper(strings.TrimSpace(name))]
	return size, ok
}

func polyProfileName(leftLen, rightLen int) string {
	if leftLen == rightLen {
		for name, size := range polyProfileSizeMap {
			if leftLen == size {
				return name
			}
		}
	}
	return fmt.Sprintf("custom-%dx%d", leftLen, rightLen)
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

func polynomialMulReference(left, right []*big.Int, modulus *big.Int) []*big.Int {
	result := make([]*big.Int, len(left)+len(right)-1)
	for i := range result {
		result[i] = new(big.Int)
	}
	for i, l := range left {
		lc := new(big.Int).Mod(l, modulus)
		for j, r := range right {
			rc := new(big.Int).Mod(r, modulus)
			term := new(big.Int).Mul(lc, rc)
			term.Mod(term, modulus)
			result[i+j].Add(result[i+j], term)
			result[i+j].Mod(result[i+j], modulus)
		}
	}
	return result
}

func defaultSolcPath() string {
	candidates := []string{
		"/home/lq/.local/share/svm/solc-0.8.26",
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
