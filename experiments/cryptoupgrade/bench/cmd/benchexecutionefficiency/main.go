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
	"math"
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
	experimentName   = "lab2-execution-efficiency"
	schemeUpgrade    = "upgrade"
	schemePrecompile = "precompile"
	schemeContract   = "contract"

	rfc3526PrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
		"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
		"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
		"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
		"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"
)

type config struct {
	rpc               string
	algorithms        string
	name              string
	from              string
	a                 string
	b                 string
	warmup            int
	samples           int
	upload            bool
	algoGas           uint64
	uploadGas         uint64
	contractDeployGas uint64
	callGas           uint64
	solcPath          string
	evmVersion        string
	outputDir         string
	outputJSON        string
}

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type benchmarkFixture struct {
	Algorithm             string
	UpgradeName           string
	SourcePath            string
	ContractSource        string
	ContractName          string
	ContractFunction      string
	PrecompileName        string
	PrecompileAddress     common.Address
	UpgradeInputTypes     []string
	UpgradeOutputTypes    []string
	ContractInputTypes    []string
	ContractOutputTypes   []string
	PrecompileInputTypes  []string
	PrecompileOutputTypes []string
	UpgradeValues         []interface{}
	ContractValues        []interface{}
	PrecompileValues      []interface{}
	Input                 map[string]string
	AlgoGas               uint64
}

type compiledContract struct {
	Contract string
	ABI      abi.ABI
	Bin      []byte
}

type setupReference struct {
	Scheme                string `json:"scheme"`
	Action                string `json:"action"`
	Version               uint64 `json:"version,omitempty"`
	ActivationBlock       uint64 `json:"activationBlock,omitempty"`
	WasmHash              string `json:"wasmHash,omitempty"`
	Address               string `json:"address,omitempty"`
	TxHash                string `json:"txHash,omitempty"`
	ReceiptStatus         uint64 `json:"receiptStatus,omitempty"`
	SourcePath            string `json:"sourcePath,omitempty"`
	Contract              string `json:"contract,omitempty"`
	Function              string `json:"function,omitempty"`
	SourceBytes           int    `json:"sourceBytes,omitempty"`
	CompressedBase64Bytes int    `json:"compressedBase64Bytes,omitempty"`
	Note                  string `json:"note,omitempty"`
}

type latencyMetrics struct {
	Warmup        int       `json:"warmup"`
	Samples       int       `json:"samples"`
	FirstMillis   float64   `json:"firstMillis"`
	MeanMillis    float64   `json:"meanMillis"`
	P50Millis     float64   `json:"p50Millis"`
	P95Millis     float64   `json:"p95Millis"`
	MinMillis     float64   `json:"minMillis"`
	MaxMillis     float64   `json:"maxMillis"`
	SampleMillis  []float64 `json:"sampleMillis"`
	ValidatedOnce bool      `json:"validatedOnceBeforeMeasurement"`
}

type implementationResult struct {
	Scheme          string         `json:"scheme"`
	Address         string         `json:"address"`
	Function        string         `json:"function,omitempty"`
	ABIInputTypes   []string       `json:"abiInputTypes"`
	ABIOutputTypes  []string       `json:"abiOutputTypes"`
	CalldataBytes   int            `json:"calldataBytes"`
	GasEstimate     uint64         `json:"gasEstimate"`
	Latency         latencyMetrics `json:"latency"`
	OutputHex       string         `json:"outputHex"`
	OutputSHA256    string         `json:"outputSha256"`
	OutputValue     string         `json:"outputValue"`
	OutputCanonical string         `json:"outputCanonical"`
}

type ratioMetrics struct {
	MeanTime    float64 `json:"meanTime"`
	P50Time     float64 `json:"p50Time"`
	P95Time     float64 `json:"p95Time"`
	GasEstimate float64 `json:"gasEstimate"`
}

type ratioSet struct {
	UpgradeOverContract    ratioMetrics `json:"upgradeOverContract"`
	UpgradeOverPrecompile  ratioMetrics `json:"upgradeOverPrecompile"`
	ContractOverPrecompile ratioMetrics `json:"contractOverPrecompile"`
}

type algorithmResult struct {
	Algorithm       string                    `json:"algorithm"`
	UpgradeName     string                    `json:"upgradeName"`
	Input           map[string]string         `json:"input"`
	Setup           map[string]setupReference `json:"setup"`
	Implementations []implementationResult    `json:"implementations"`
	Ratios          ratioSet                  `json:"ratios"`
	OutputMatched   bool                      `json:"outputMatched"`
}

type skippedAlgorithm struct {
	Algorithm string `json:"algorithm"`
	Reason    string `json:"reason"`
}

type suiteResult struct {
	Experiment          string             `json:"experiment"`
	Timestamp           string             `json:"timestamp"`
	RPC                 string             `json:"rpc"`
	Sender              string             `json:"sender"`
	Command             []string           `json:"command"`
	Warmup              int                `json:"warmup"`
	Samples             int                `json:"samples"`
	RequestedAlgorithms []string           `json:"requestedAlgorithms"`
	Results             []algorithmResult  `json:"results"`
	Skipped             []skippedAlgorithm `json:"skipped"`
	OutputJSON          string             `json:"outputJson"`
	Notes               []string           `json:"notes"`
}

type callTarget struct {
	scheme      string
	address     common.Address
	function    string
	inputTypes  []string
	outputTypes []string
	calldata    []byte
	returns     abi.Arguments
}

type decodedOutput struct {
	canonical string
	display   string
}

type benchmarkStats struct {
	durations []time.Duration
	first     time.Duration
	mean      time.Duration
	p50       time.Duration
	p95       time.Duration
	min       time.Duration
	max       time.Duration
	output    []byte
	decoded   decodedOutput
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchexecutionefficiency: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		return err
	}
	fixtures, skipped, err := buildFixtures(cfg)
	if err != nil {
		return err
	}
	selected, selectedSkipped, requested, err := selectFixtures(fixtures, skipped, cfg)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		return errors.New("no comparable algorithms selected")
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
		outputJSON = filepath.Join(cfg.outputDir, "all-"+now.Format("20060102-150405"), "result.json")
	}

	suite := suiteResult{
		Experiment:          experimentName,
		Timestamp:           now.Format(time.RFC3339Nano),
		RPC:                 cfg.rpc,
		Sender:              from.Hex(),
		Command:             os.Args,
		Warmup:              cfg.warmup,
		Samples:             cfg.samples,
		RequestedAlgorithms: requested,
		OutputJSON:          outputJSON,
		Skipped:             selectedSkipped,
		Notes: []string{
			"setup entries are reproducibility data only; execution efficiency compares eth_call latency and eth_estimateGas gasEstimate",
			"gasEstimate is not transaction receipt gasUsed",
			"contract deployment and CodeStorage.uploadCode are excluded from measured latency",
		},
	}

	for _, fixture := range selected {
		fmt.Printf("benchmarking %s...\n", fixture.Algorithm)
		result, err := runFixture(ctx, client, codeStorageABI, from, cfg, fixture)
		if err != nil {
			return fmt.Errorf("%s: %w", fixture.Algorithm, err)
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
	flag.StringVar(&cfg.rpc, "rpc", "http://127.0.0.1:8666", "JSON-RPC endpoint")
	flag.StringVar(&cfg.algorithms, "algorithms", "all", "comma-separated algorithm list or all")
	flag.StringVar(&cfg.name, "name", "", "single algorithm alias for -algorithms")
	flag.StringVar(&cfg.from, "from", "", "sender address; defaults to eth_accounts[0]")
	flag.StringVar(&cfg.a, "a", "100", "first Add operand, non-negative decimal integer")
	flag.StringVar(&cfg.b, "b", "100", "second Add operand, non-negative decimal integer")
	flag.IntVar(&cfg.warmup, "warmup", 10, "warmup eth_call count per implementation")
	flag.IntVar(&cfg.samples, "n", 100, "measured eth_call count per implementation")
	flag.BoolVar(&cfg.upload, "upload", true, "upload each selected Go algorithm through CodeStorage before measuring")
	flag.Uint64Var(&cfg.algoGas, "algo-gas", 0, "override algorithm gas metadata recorded by CodeStorage.uploadCode; 0 uses fixture defaults")
	flag.Uint64Var(&cfg.uploadGas, "upload-gas", 5000000, "gas limit for CodeStorage.uploadCode setup transaction")
	flag.Uint64Var(&cfg.contractDeployGas, "contract-deploy-gas", 0, "gas limit for contract setup deployment; 0 lets the node estimate")
	flag.Uint64Var(&cfg.callGas, "call-gas", 50000000, "gas limit for eth_call validation and latency measurement")
	flag.StringVar(&cfg.solcPath, "solc", defaultSolcPath(), "solc compiler path")
	flag.StringVar(&cfg.evmVersion, "evm-version", "paris", "solc EVM target")
	flag.StringVar(&cfg.outputDir, "output-dir", "experiments/cryptoupgrade/results/execution-efficiency", "directory used when -output-json is empty")
	flag.StringVar(&cfg.outputJSON, "output-json", "", "write JSON result to this file; defaults under -output-dir")
	flag.Parse()
	return cfg
}

func (cfg config) validate() error {
	if cfg.rpc == "" {
		return errors.New("-rpc is required")
	}
	if cfg.samples <= 0 {
		return errors.New("-n must be positive")
	}
	if cfg.warmup < 0 {
		return errors.New("-warmup cannot be negative")
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return fmt.Errorf("invalid -from %q", cfg.from)
	}
	if cfg.solcPath == "" {
		return errors.New("-solc is required")
	}
	return nil
}

func buildFixtures(cfg config) ([]benchmarkFixture, []skippedAlgorithm, error) {
	a, err := parseAddOperand("a", cfg.a)
	if err != nil {
		return nil, nil, err
	}
	b, err := parseAddOperand("b", cfg.b)
	if err != nil {
		return nil, nil, err
	}
	expectedAdd := new(big.Int).Add(a, b)
	if expectedAdd.Cmp(maxInt256()) > 0 {
		return nil, nil, fmt.Errorf("a+b exceeds int256 range: %s", expectedAdd)
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

	fixtures := []benchmarkFixture{
		{
			Algorithm:             "Add",
			UpgradeName:           "Add",
			SourcePath:            "cryptoupgrade/algorithm/wasm/add.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/Add.sol",
			ContractName:          "AddContract",
			ContractFunction:      "Add",
			PrecompileName:        "Add",
			PrecompileAddress:     common.CryptoUpgradeAddAddress,
			UpgradeInputTypes:     []string{"int256", "int256"},
			UpgradeOutputTypes:    []string{"int256"},
			ContractInputTypes:    []string{"int256", "int256"},
			ContractOutputTypes:   []string{"int256"},
			PrecompileInputTypes:  []string{"uint256", "uint256"},
			PrecompileOutputTypes: []string{"uint256"},
			UpgradeValues:         []interface{}{a, b},
			ContractValues:        []interface{}{a, b},
			PrecompileValues:      []interface{}{a, b},
			Input: map[string]string{
				"a":        a.String(),
				"b":        b.String(),
				"expected": expectedAdd.String(),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:             "Sha256",
			UpgradeName:           "Sha256",
			SourcePath:            "cryptoupgrade/algorithm/wasm/sha256.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/Sha256.sol",
			ContractName:          "Sha256Contract",
			ContractFunction:      "Sha256",
			PrecompileName:        "Sha256",
			PrecompileAddress:     common.CryptoUpgradeSha256Address,
			UpgradeInputTypes:     []string{"bytes"},
			UpgradeOutputTypes:    []string{"bytes"},
			ContractInputTypes:    []string{"bytes"},
			ContractOutputTypes:   []string{"bytes"},
			PrecompileInputTypes:  []string{"bytes"},
			PrecompileOutputTypes: []string{"bytes"},
			UpgradeValues:         []interface{}{sampleData},
			ContractValues:        []interface{}{sampleData},
			PrecompileValues:      []interface{}{sampleData},
			Input: map[string]string{
				"dataHex": hexutil.Encode(sampleData),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:             "Blake2bSum256",
			UpgradeName:           "Sum256",
			SourcePath:            "cryptoupgrade/algorithm/wasm/blake2b.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/Blake2b.sol",
			ContractName:          "Blake2b",
			ContractFunction:      "Sum256",
			PrecompileName:        "Blake2bSum256",
			PrecompileAddress:     common.CryptoUpgradeBlake2bSum256Address,
			UpgradeInputTypes:     []string{"bytes"},
			UpgradeOutputTypes:    []string{"bytes32"},
			ContractInputTypes:    []string{"bytes"},
			ContractOutputTypes:   []string{"bytes32"},
			PrecompileInputTypes:  []string{"bytes"},
			PrecompileOutputTypes: []string{"bytes"},
			UpgradeValues:         []interface{}{sampleData},
			ContractValues:        []interface{}{sampleData},
			PrecompileValues:      []interface{}{sampleData},
			Input: map[string]string{
				"dataHex": hexutil.Encode(sampleData),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:             "Pbkdf2Sha256",
			UpgradeName:           "Pbkdf2Sha256",
			SourcePath:            "cryptoupgrade/algorithm/wasm/pbkdf2_sha256.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/Pbkdf2Sha256.sol",
			ContractName:          "Pbkdf2Sha256Contract",
			ContractFunction:      "Pbkdf2Sha256",
			PrecompileName:        "Pbkdf2Sha256",
			PrecompileAddress:     common.CryptoUpgradePbkdf2Sha256Address,
			UpgradeInputTypes:     []string{"bytes", "bytes", "uint256", "uint256"},
			UpgradeOutputTypes:    []string{"bytes"},
			ContractInputTypes:    []string{"bytes", "bytes", "uint256", "uint256"},
			ContractOutputTypes:   []string{"bytes"},
			PrecompileInputTypes:  []string{"bytes", "bytes", "uint256", "uint256"},
			PrecompileOutputTypes: []string{"bytes"},
			UpgradeValues:         []interface{}{pbkdf2Password, pbkdf2Salt, pbkdf2Iterations, pbkdf2KeyLength},
			ContractValues:        []interface{}{pbkdf2Password, pbkdf2Salt, pbkdf2Iterations, pbkdf2KeyLength},
			PrecompileValues:      []interface{}{pbkdf2Password, pbkdf2Salt, pbkdf2Iterations, pbkdf2KeyLength},
			Input: map[string]string{
				"passwordHex": hexutil.Encode(pbkdf2Password),
				"saltHex":     hexutil.Encode(pbkdf2Salt),
				"iterations":  pbkdf2Iterations.String(),
				"keyLength":   pbkdf2KeyLength.String(),
			},
			AlgoGas: 25000,
		},
		{
			Algorithm:             "Dh2048Secret",
			UpgradeName:           "Dh2048Secret",
			SourcePath:            "cryptoupgrade/algorithm/wasm/dh2048.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/Dh2048.sol",
			ContractName:          "Dh2048",
			ContractFunction:      "Dh2048Secret",
			PrecompileName:        "Dh2048Secret",
			PrecompileAddress:     common.CryptoUpgradeDh2048SecretAddress,
			UpgradeInputTypes:     []string{"bytes", "bytes"},
			UpgradeOutputTypes:    []string{"bytes"},
			ContractInputTypes:    []string{"bytes", "bytes"},
			ContractOutputTypes:   []string{"bytes"},
			PrecompileInputTypes:  []string{"bytes", "bytes"},
			PrecompileOutputTypes: []string{"bytes"},
			UpgradeValues:         []interface{}{dhPrivate, dhPeerPublic},
			ContractValues:        []interface{}{dhPrivate, dhPeerPublic},
			PrecompileValues:      []interface{}{dhPrivate, dhPeerPublic},
			Input: map[string]string{
				"privateKeyHex":        hexutil.Encode(dhPrivate),
				"peerPublicKeyHex":     hexutil.Encode(dhPeerPublic),
				"peerPublicKeySha256":  sha256Hex(dhPeerPublic),
				"peerPublicKeyByteLen": fmt.Sprintf("%d", len(dhPeerPublic)),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:             "PedersenCommit",
			UpgradeName:           "PedersenCommit",
			SourcePath:            "cryptoupgrade/algorithm/wasm/pedersen_commit.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/PedersenCommit.sol",
			ContractName:          "PedersenCommitContract",
			ContractFunction:      "PedersenCommit",
			PrecompileName:        "PedersenCommit",
			PrecompileAddress:     common.CryptoUpgradePedersenCommitAddress,
			UpgradeInputTypes:     []string{"bytes", "bytes"},
			UpgradeOutputTypes:    []string{"bytes"},
			ContractInputTypes:    []string{"bytes", "bytes"},
			ContractOutputTypes:   []string{"bytes"},
			PrecompileInputTypes:  []string{"bytes", "bytes"},
			PrecompileOutputTypes: []string{"bytes"},
			UpgradeValues:         []interface{}{pedersenMessage, pedersenBlinding},
			ContractValues:        []interface{}{pedersenMessage, pedersenBlinding},
			PrecompileValues:      []interface{}{pedersenMessage, pedersenBlinding},
			Input: map[string]string{
				"messageHex":  hexutil.Encode(pedersenMessage),
				"blindingHex": hexutil.Encode(pedersenBlinding),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:             "SchnorrVerify",
			UpgradeName:           "SchnorrVerify",
			SourcePath:            "cryptoupgrade/algorithm/wasm/schnorr_proof.wasm",
			ContractSource:        "cryptoupgrade/algorithm/contracts/SchnorrProof.sol",
			ContractName:          "SchnorrProof",
			ContractFunction:      "SchnorrVerify",
			PrecompileName:        "SchnorrVerify",
			PrecompileAddress:     common.CryptoUpgradeSchnorrVerifyAddress,
			UpgradeInputTypes:     []string{"bytes", "bytes"},
			UpgradeOutputTypes:    []string{"bool"},
			ContractInputTypes:    []string{"bytes", "bytes"},
			ContractOutputTypes:   []string{"bool"},
			PrecompileInputTypes:  []string{"bytes", "bytes"},
			PrecompileOutputTypes: []string{"bool"},
			UpgradeValues:         []interface{}{schnorrMessage, schnorrProof},
			ContractValues:        []interface{}{schnorrMessage, schnorrProof},
			PrecompileValues:      []interface{}{schnorrMessage, schnorrProof},
			Input: map[string]string{
				"messageHex":      hexutil.Encode(schnorrMessage),
				"proofHex":        hexutil.Encode(schnorrProof),
				"proofSha256":     sha256Hex(schnorrProof),
				"proofByteLength": fmt.Sprintf("%d", len(schnorrProof)),
			},
			AlgoGas: 200000,
		},
	}

	skipped := []skippedAlgorithm{
		{Algorithm: "Blake2bSum384", Reason: "missing comparable Go and contract implementation"},
		{Algorithm: "Blake2bSum512", Reason: "missing comparable Go and contract implementation"},
		{Algorithm: "AesCBCEncrypt", Reason: "Go implementation is archived and no comparable contract implementation exists"},
		{Algorithm: "AesCBCDecrypt", Reason: "Go implementation is archived and no comparable contract implementation exists"},
		{Algorithm: "Dh2048Private", Reason: "key-generation helper without comparable contract algorithm"},
		{Algorithm: "Dh2048Public", Reason: "key-generation helper without comparable contract algorithm"},
		{Algorithm: "Ed25519Keygen", Reason: "Go implementation is archived and no comparable contract implementation exists"},
		{Algorithm: "Ed25519PublicKey", Reason: "Go implementation is archived and no comparable contract implementation exists"},
		{Algorithm: "Ed25519Sign", Reason: "Go implementation is archived and no comparable contract implementation exists"},
		{Algorithm: "Ed25519Verify", Reason: "Go implementation is archived and no comparable contract implementation exists"},
		{Algorithm: "PedersenVerify", Reason: "missing comparable active Go and contract implementation"},
		{Algorithm: "SchnorrPublicKey", Reason: "key-generation helper without comparable contract algorithm"},
	}
	return fixtures, skipped, nil
}

func selectFixtures(fixtures []benchmarkFixture, skipped []skippedAlgorithm, cfg config) ([]benchmarkFixture, []skippedAlgorithm, []string, error) {
	raw := cfg.algorithms
	if cfg.name != "" {
		raw = cfg.name
	}
	requested := splitAlgorithmList(raw)
	if len(requested) == 0 || (len(requested) == 1 && algorithmKey(requested[0]) == "all") {
		names := make([]string, 0, len(fixtures))
		for _, fixture := range fixtures {
			names = append(names, fixture.Algorithm)
		}
		return fixtures, skipped, names, nil
	}

	requestedSet := make(map[string]string, len(requested))
	for _, name := range requested {
		requestedSet[algorithmKey(name)] = name
	}
	var selected []benchmarkFixture
	for _, fixture := range fixtures {
		if _, ok := requestedSet[algorithmKey(fixture.Algorithm)]; ok {
			selected = append(selected, fixture)
			delete(requestedSet, algorithmKey(fixture.Algorithm))
		}
	}
	var selectedSkipped []skippedAlgorithm
	for _, entry := range skipped {
		if _, ok := requestedSet[algorithmKey(entry.Algorithm)]; ok {
			selectedSkipped = append(selectedSkipped, entry)
			delete(requestedSet, algorithmKey(entry.Algorithm))
		}
	}
	if len(requestedSet) > 0 {
		var unknown []string
		for _, name := range requestedSet {
			unknown = append(unknown, name)
		}
		sort.Strings(unknown)
		return nil, nil, nil, fmt.Errorf("unknown algorithms: %s", strings.Join(unknown, ", "))
	}
	return selected, selectedSkipped, requested, nil
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

func runFixture(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, fixture benchmarkFixture) (algorithmResult, error) {
	setup := make(map[string]setupReference)
	if cfg.upload {
		ref, err := uploadAlgorithm(ctx, client, codeStorageABI, from, cfg, fixture)
		if err != nil {
			return algorithmResult{}, err
		}
		setup[schemeUpgrade] = ref
	} else {
		setup[schemeUpgrade] = setupReference{
			Scheme:   schemeUpgrade,
			Action:   "reuse-existing-upload",
			Address:  common.CodeStorageAddress.Hex(),
			Function: fixture.UpgradeName,
			Note:     "selected algorithms must already be uploaded in the connected node",
		}
	}

	contract, err := compileSolidity(cfg.solcPath, cfg.evmVersion, fixture.ContractSource, fixture.ContractName, fixture.ContractFunction)
	if err != nil {
		return algorithmResult{}, err
	}
	contractAddress, contractSetup, err := deployFixtureContract(ctx, client, from, contract, fixture, cfg.contractDeployGas)
	if err != nil {
		return algorithmResult{}, err
	}
	setup[schemeContract] = contractSetup
	setup[schemePrecompile] = setupReference{
		Scheme:   schemePrecompile,
		Action:   "precompile-address",
		Address:  fixture.PrecompileAddress.Hex(),
		Function: fixture.PrecompileName,
		Note:     "precompile requires no setup transaction",
	}

	targets, err := buildTargets(codeStorageABI, contract, contractAddress, fixture)
	if err != nil {
		return algorithmResult{}, err
	}
	if cfg.upload {
		if err := waitCallFuncReady(ctx, client, from, targets[0], cfg.callGas, 30*time.Second); err != nil {
			return algorithmResult{}, err
		}
	}
	if err := validateTargets(ctx, client, from, targets, cfg.callGas); err != nil {
		return algorithmResult{}, err
	}

	implementations := make([]implementationResult, 0, len(targets))
	for _, target := range targets {
		gas, err := estimateGas(ctx, client, txArgs{From: from, To: &target.address, Data: target.calldata})
		if err != nil {
			return algorithmResult{}, fmt.Errorf("estimate %s gas: %w", target.scheme, err)
		}
		stats, err := benchmarkTarget(ctx, client, from, target, cfg.warmup, cfg.samples, cfg.callGas)
		if err != nil {
			return algorithmResult{}, fmt.Errorf("benchmark %s: %w", target.scheme, err)
		}
		implementations = append(implementations, makeImplementationResult(target, gas, stats, cfg.warmup))
	}

	return algorithmResult{
		Algorithm:       fixture.Algorithm,
		UpgradeName:     fixture.UpgradeName,
		Input:           fixture.Input,
		Setup:           setup,
		Implementations: implementations,
		Ratios:          makeRatios(implementations),
		OutputMatched:   true,
	}, nil
}

func uploadAlgorithm(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, fixture benchmarkFixture) (setupReference, error) {
	sourceBytes, wasmHash, compressed, err := compressSource(ctx, fixture.SourcePath, fixture.UpgradeName, fixture.UpgradeInputTypes, fixture.UpgradeOutputTypes)
	if err != nil {
		return setupReference{}, err
	}
	algoGas := fixture.AlgoGas
	if cfg.algoGas != 0 {
		algoGas = cfg.algoGas
	}
	data, err := codeStorageABI.Pack(
		"uploadCode",
		fixture.UpgradeName,
		compressed,
		algoGas,
		strings.Join(fixture.UpgradeInputTypes, ","),
		strings.Join(fixture.UpgradeOutputTypes, ","),
	)
	if err != nil {
		return setupReference{}, fmt.Errorf("pack CodeStorage.uploadCode: %w", err)
	}
	gas := hexutil.Uint64(cfg.uploadGas)
	to := common.CodeStorageAddress
	txHash, err := sendTransaction(ctx, client, txArgs{From: from, To: &to, Gas: &gas, Data: data})
	if err != nil {
		return setupReference{}, fmt.Errorf("send upload transaction: %w", err)
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return setupReference{}, err
	}
	ref := setupReference{
		Scheme:                schemeUpgrade,
		Action:                "CodeStorage.uploadCode",
		Version:               1,
		ActivationBlock:       0,
		WasmHash:              wasmHash,
		Address:               common.CodeStorageAddress.Hex(),
		TxHash:                txHash.Hex(),
		ReceiptStatus:         receipt.Status,
		SourcePath:            mustAbs(fixture.SourcePath),
		Function:              fixture.UpgradeName,
		SourceBytes:           sourceBytes,
		CompressedBase64Bytes: len(compressed),
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return ref, fmt.Errorf("upload transaction %s failed with receipt status %d", txHash.Hex(), receipt.Status)
	}
	return ref, nil
}

func deployFixtureContract(ctx context.Context, client *rpc.Client, from common.Address, contract compiledContract, fixture benchmarkFixture, gasLimit uint64) (common.Address, setupReference, error) {
	addr, txHash, receipt, err := deployContract(ctx, client, from, contract.Bin, gasLimit)
	if err != nil {
		return common.Address{}, setupReference{}, err
	}
	ref := setupReference{
		Scheme:        schemeContract,
		Action:        "deploy-contract",
		Address:       addr.Hex(),
		TxHash:        txHash.Hex(),
		ReceiptStatus: receipt.Status,
		SourcePath:    mustAbs(fixture.ContractSource),
		Contract:      contract.Contract,
		Function:      fixture.ContractFunction,
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return addr, ref, fmt.Errorf("contract deployment %s failed with receipt status %d", txHash.Hex(), receipt.Status)
	}
	return addr, ref, nil
}

func buildTargets(codeStorageABI abi.ABI, contract compiledContract, contractAddress common.Address, fixture benchmarkFixture) ([]callTarget, error) {
	upgradeArgs, err := abiArguments(fixture.UpgradeInputTypes...)
	if err != nil {
		return nil, err
	}
	upgradeReturn, err := abiArguments(fixture.UpgradeOutputTypes...)
	if err != nil {
		return nil, err
	}
	contractReturn, err := abiArguments(fixture.ContractOutputTypes...)
	if err != nil {
		return nil, err
	}
	precompileArgs, err := abiArguments(fixture.PrecompileInputTypes...)
	if err != nil {
		return nil, err
	}
	precompileReturn, err := abiArguments(fixture.PrecompileOutputTypes...)
	if err != nil {
		return nil, err
	}

	encodedUpgradeInput, err := upgradeArgs.Pack(fixture.UpgradeValues...)
	if err != nil {
		return nil, fmt.Errorf("pack %s upgrade input: %w", fixture.Algorithm, err)
	}
	upgradeCallData, err := codeStorageABI.Pack("callFunc", fixture.UpgradeName, encodedUpgradeInput)
	if err != nil {
		return nil, fmt.Errorf("pack CodeStorage.callFunc: %w", err)
	}
	contractCallData, err := contract.ABI.Pack(fixture.ContractFunction, fixture.ContractValues...)
	if err != nil {
		return nil, fmt.Errorf("pack %s contract call: %w", fixture.Algorithm, err)
	}
	precompileCallData, err := precompileArgs.Pack(fixture.PrecompileValues...)
	if err != nil {
		return nil, fmt.Errorf("pack %s precompile input: %w", fixture.Algorithm, err)
	}

	return []callTarget{
		{
			scheme:      schemeUpgrade,
			address:     common.CodeStorageAddress,
			function:    fixture.UpgradeName,
			inputTypes:  append([]string(nil), fixture.UpgradeInputTypes...),
			outputTypes: append([]string(nil), fixture.UpgradeOutputTypes...),
			calldata:    upgradeCallData,
			returns:     upgradeReturn,
		},
		{
			scheme:      schemeContract,
			address:     contractAddress,
			function:    fixture.ContractFunction,
			inputTypes:  append([]string(nil), fixture.ContractInputTypes...),
			outputTypes: append([]string(nil), fixture.ContractOutputTypes...),
			calldata:    contractCallData,
			returns:     contractReturn,
		},
		{
			scheme:      schemePrecompile,
			address:     fixture.PrecompileAddress,
			function:    fixture.PrecompileName,
			inputTypes:  append([]string(nil), fixture.PrecompileInputTypes...),
			outputTypes: append([]string(nil), fixture.PrecompileOutputTypes...),
			calldata:    precompileCallData,
			returns:     precompileReturn,
		},
	}, nil
}

func validateTargets(ctx context.Context, client *rpc.Client, from common.Address, targets []callTarget, callGas uint64) error {
	var expected decodedOutput
	var expectedScheme string
	for i, target := range targets {
		output, err := ethCall(ctx, client, txArgs{From: from, To: &target.address, Data: target.calldata}, callGas)
		if err != nil {
			return fmt.Errorf("validate %s call: %w", target.scheme, err)
		}
		decoded, err := decodeSingleOutput(target.returns, output)
		if err != nil {
			return fmt.Errorf("decode %s output: %w", target.scheme, err)
		}
		if i == 0 {
			expected = decoded
			expectedScheme = target.scheme
			continue
		}
		if decoded.canonical != expected.canonical {
			return fmt.Errorf("%s output mismatch: got %s want %s from %s", target.scheme, decoded.display, expected.display, expectedScheme)
		}
	}
	return nil
}

func waitCallFuncReady(ctx context.Context, client *rpc.Client, from common.Address, target callTarget, callGas uint64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, err := ethCall(ctx, client, txArgs{From: from, To: &target.address, Data: target.calldata}, callGas); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for event-driven activation of %s: %w", target.function, lastErr)
}

func benchmarkTarget(ctx context.Context, client *rpc.Client, from common.Address, target callTarget, warmup, samples int, callGas uint64) (benchmarkStats, error) {
	for i := 0; i < warmup; i++ {
		if _, err := ethCall(ctx, client, txArgs{From: from, To: &target.address, Data: target.calldata}, callGas); err != nil {
			return benchmarkStats{}, fmt.Errorf("warmup %d: %w", i+1, err)
		}
	}
	durations := make([]time.Duration, 0, samples)
	var firstOutput []byte
	var firstDecoded decodedOutput
	for i := 0; i < samples; i++ {
		started := time.Now()
		output, err := ethCall(ctx, client, txArgs{From: from, To: &target.address, Data: target.calldata}, callGas)
		elapsed := time.Since(started)
		if err != nil {
			return benchmarkStats{}, fmt.Errorf("sample %d: %w", i+1, err)
		}
		decoded, err := decodeSingleOutput(target.returns, output)
		if err != nil {
			return benchmarkStats{}, fmt.Errorf("sample %d decode output: %w", i+1, err)
		}
		if i == 0 {
			firstOutput = append([]byte(nil), output...)
			firstDecoded = decoded
		} else if decoded.canonical != firstDecoded.canonical {
			return benchmarkStats{}, fmt.Errorf("sample %d returned non-deterministic output: first=%s current=%s", i+1, firstDecoded.display, decoded.display)
		}
		durations = append(durations, elapsed)
	}
	stats := summarizeDurations(durations)
	stats.output = firstOutput
	stats.decoded = firstDecoded
	return stats, nil
}

func makeImplementationResult(target callTarget, gas uint64, stats benchmarkStats, warmup int) implementationResult {
	sum := sha256.Sum256(stats.output)
	sampleMillis := make([]float64, len(stats.durations))
	for i, duration := range stats.durations {
		sampleMillis[i] = millis(duration)
	}
	return implementationResult{
		Scheme:         target.scheme,
		Address:        target.address.Hex(),
		Function:       target.function,
		ABIInputTypes:  append([]string(nil), target.inputTypes...),
		ABIOutputTypes: append([]string(nil), target.outputTypes...),
		CalldataBytes:  len(target.calldata),
		GasEstimate:    gas,
		Latency: latencyMetrics{
			Warmup:        warmup,
			Samples:       len(sampleMillis),
			FirstMillis:   millis(stats.first),
			MeanMillis:    millis(stats.mean),
			P50Millis:     millis(stats.p50),
			P95Millis:     millis(stats.p95),
			MinMillis:     millis(stats.min),
			MaxMillis:     millis(stats.max),
			SampleMillis:  sampleMillis,
			ValidatedOnce: true,
		},
		OutputHex:       hexutil.Encode(stats.output),
		OutputSHA256:    hex.EncodeToString(sum[:]),
		OutputValue:     stats.decoded.display,
		OutputCanonical: stats.decoded.canonical,
	}
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

func summarizeDurations(durations []time.Duration) benchmarkStats {
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	total := time.Duration(0)
	for _, duration := range durations {
		total += duration
	}
	return benchmarkStats{
		durations: append([]time.Duration(nil), durations...),
		first:     durations[0],
		mean:      total / time.Duration(len(durations)),
		p50:       percentile(sorted, 0.50),
		p95:       percentile(sorted, 0.95),
		min:       sorted[0],
		max:       sorted[len(sorted)-1],
	}
}

func makeRatios(implementations []implementationResult) ratioSet {
	byScheme := make(map[string]implementationResult, len(implementations))
	for _, impl := range implementations {
		byScheme[impl.Scheme] = impl
	}
	return ratioSet{
		UpgradeOverContract:    ratioImplementation(byScheme[schemeUpgrade], byScheme[schemeContract]),
		UpgradeOverPrecompile:  ratioImplementation(byScheme[schemeUpgrade], byScheme[schemePrecompile]),
		ContractOverPrecompile: ratioImplementation(byScheme[schemeContract], byScheme[schemePrecompile]),
	}
}

func ratioImplementation(a, b implementationResult) ratioMetrics {
	return ratioMetrics{
		MeanTime:    safeRatio(a.Latency.MeanMillis, b.Latency.MeanMillis),
		P50Time:     safeRatio(a.Latency.P50Millis, b.Latency.P50Millis),
		P95Time:     safeRatio(a.Latency.P95Millis, b.Latency.P95Millis),
		GasEstimate: safeRatio(float64(a.GasEstimate), float64(b.GasEstimate)),
	}
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

func deployContract(ctx context.Context, client *rpc.Client, from common.Address, initCode []byte, gasLimit uint64) (common.Address, common.Hash, *types.Receipt, error) {
	args := txArgs{From: from, Data: initCode}
	if gasLimit != 0 {
		gas := hexutil.Uint64(gasLimit)
		args.Gas = &gas
	}
	txHash, err := sendTransaction(ctx, client, args)
	if err != nil {
		return common.Address{}, common.Hash{}, nil, fmt.Errorf("send contract deployment transaction: %w", err)
	}
	receipt, err := waitReceipt(ctx, client, txHash, 2*time.Minute)
	if err != nil {
		return common.Address{}, common.Hash{}, nil, err
	}
	return receipt.ContractAddress, txHash, receipt, nil
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
			return nil, fmt.Errorf("ABI type %q: %w", raw, err)
		}
		args = append(args, abi.Argument{Type: typ})
	}
	return args, nil
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

func ethCall(ctx context.Context, client *rpc.Client, args txArgs, callGas uint64) ([]byte, error) {
	if callGas != 0 {
		gas := hexutil.Uint64(callGas)
		args.Gas = &gas
	}
	var output hexutil.Bytes
	if err := client.CallContext(ctx, &output, "eth_call", args, "latest"); err != nil {
		return nil, err
	}
	return output, nil
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
	fmt.Println("Lab2 execution efficiency benchmark")
	fmt.Printf("  rpc:       %s\n", res.RPC)
	fmt.Printf("  sender:    %s\n", res.Sender)
	fmt.Printf("  samples:   warmup=%d n=%d\n", res.Warmup, res.Samples)
	fmt.Println()
	fmt.Println("Call metrics (eth_call latency, eth_estimateGas gasEstimate)")
	for _, algorithm := range res.Results {
		fmt.Printf("\n%s input=%s matched=%t\n", algorithm.Algorithm, formatInput(algorithm.Input), algorithm.OutputMatched)
		for _, impl := range algorithm.Implementations {
			fmt.Printf("  %-10s gas=%d mean=%.3fms p50=%.3fms p95=%.3fms min=%.3fms max=%.3fms\n",
				impl.Scheme,
				impl.GasEstimate,
				impl.Latency.MeanMillis,
				impl.Latency.P50Millis,
				impl.Latency.P95Millis,
				impl.Latency.MinMillis,
				impl.Latency.MaxMillis,
			)
		}
		fmt.Printf("  ratios upgrade/contract mean=%.2fx gas=%.2fx; upgrade/precompile mean=%.2fx gas=%.2fx; contract/precompile mean=%.2fx gas=%.2fx\n",
			algorithm.Ratios.UpgradeOverContract.MeanTime,
			algorithm.Ratios.UpgradeOverContract.GasEstimate,
			algorithm.Ratios.UpgradeOverPrecompile.MeanTime,
			algorithm.Ratios.UpgradeOverPrecompile.GasEstimate,
			algorithm.Ratios.ContractOverPrecompile.MeanTime,
			algorithm.Ratios.ContractOverPrecompile.GasEstimate,
		)
	}
	if len(res.Skipped) > 0 {
		fmt.Println()
		fmt.Println("Skipped algorithms")
		for _, entry := range res.Skipped {
			fmt.Printf("  %s: %s\n", entry.Algorithm, entry.Reason)
		}
	}
	fmt.Printf("\nJSON result: %s\n", res.OutputJSON)
}

func formatInput(input map[string]string) string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := input[key]
		if len(value) > 72 {
			value = value[:69] + "..."
		}
		parts = append(parts, key+"="+value)
	}
	return strings.Join(parts, ",")
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

func millis(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / float64(time.Millisecond)
}

func safeRatio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
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

func algorithmKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return name
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
