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
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	gethblake2b "github.com/ethereum/go-ethereum/crypto/blake2b"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	changeName     = "lab1-upgrade-latency"
	experimentName = "multi-node-upgrade-latency"
	defaultConfig  = "experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml"

	rfc3526PrimeHex = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3" +
		"404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BF" +
		"B5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C6" +
		"2F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28" +
		"FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF"
)

type config struct {
	repoRoot      string
	configPath    string
	resultDir     string
	outputJSON    string
	outputCSV     string
	outputNodeCSV string

	senderID string
	nodeIDs  string
	from     string

	sourcePath  string
	algorithm   string
	algorithms  string
	uniqueNames bool
	inputType   string
	outputType  string
	algoGas     uint64
	txGas       uint64
	callGas     uint64
	a           string
	b           string

	rounds           int
	pollInterval     time.Duration
	timeout          time.Duration
	preflightTimeout time.Duration
	preflight        bool
}

type selectedNetwork struct {
	cfg     *network.Config
	sender  network.NodeConfig
	targets []network.NodeConfig
	from    common.Address
}

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type experimentResult struct {
	Experiment  string            `json:"experiment"`
	Change      string            `json:"change"`
	Timestamp   string            `json:"timestamp"`
	Command     []string          `json:"command"`
	Config      experimentConfig  `json:"config"`
	Artifacts   artifactResult    `json:"artifacts"`
	Preflight   *preflightResult  `json:"preflight,omitempty"`
	Rounds      []roundResult     `json:"rounds"`
	Summary     experimentSummary `json:"summary"`
	Limitations []string          `json:"limitations"`
	Completed   bool              `json:"completed"`
	Error       string            `json:"error,omitempty"`
}

type experimentConfig struct {
	ConfigPath          string   `json:"configPath"`
	ChainID             uint64   `json:"chainId"`
	NetworkID           uint64   `json:"networkId"`
	ConsensusType       string   `json:"consensusType"`
	ConsensusPeriod     uint64   `json:"consensusPeriod"`
	NodeCount           int      `json:"nodeCount"`
	TargetNodeIDs       []string `json:"targetNodeIds"`
	ExcludedNodeIDs     []string `json:"excludedNodeIds,omitempty"`
	SenderNodeID        string   `json:"senderNodeId"`
	SenderRPC           string   `json:"senderRpc"`
	From                string   `json:"from"`
	Rounds              int      `json:"rounds"`
	PollIntervalMillis  float64  `json:"pollIntervalMillis"`
	TimeoutMillis       float64  `json:"timeoutMillis"`
	Preflight           bool     `json:"preflight"`
	PreflightTimeoutMS  float64  `json:"preflightTimeoutMillis"`
	AlgorithmBase       string   `json:"algorithmBase"`
	Algorithms          []string `json:"algorithms"`
	UniqueNames         bool     `json:"uniqueNames"`
	InputType           string   `json:"inputType"`
	OutputType          string   `json:"outputType"`
	AlgoGas             uint64   `json:"algoGas"`
	TransactionGasLimit uint64   `json:"transactionGasLimit"`
	CallGas             uint64   `json:"callGas"`
}

type artifactResult struct {
	ResultDir    string `json:"resultDir"`
	ResultJSON   string `json:"resultJson"`
	RoundCSV     string `json:"roundCsv"`
	NodeCSV      string `json:"nodeCsv"`
	SourcePath   string `json:"sourcePath"`
	GeneratedDir string `json:"generatedSourceDir,omitempty"`
}

type preflightResult struct {
	StartedAt         string          `json:"startedAt"`
	FinishedAt        string          `json:"finishedAt"`
	DurationMillis    float64         `json:"durationMillis"`
	ExpectedPeerCount int             `json:"expectedPeerCount"`
	Nodes             []preflightNode `json:"nodes"`
	OK                bool            `json:"ok"`
	Error             string          `json:"error,omitempty"`
}

type preflightNode struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	RPCURL     string `json:"rpcURL"`
	ChainID    uint64 `json:"chainId,omitempty"`
	PeerCount  uint64 `json:"peerCount,omitempty"`
	StartBlock uint64 `json:"startBlock,omitempty"`
	EndBlock   uint64 `json:"endBlock,omitempty"`
	Error      string `json:"error,omitempty"`
}

type roundResult struct {
	Index                int               `json:"index"`
	Algorithm            string            `json:"algorithm"`
	UpgradeName          string            `json:"upgradeName"`
	SourcePath           string            `json:"sourcePath"`
	ABIInputTypes        []string          `json:"abiInputTypes,omitempty"`
	ABIOutputTypes       []string          `json:"abiOutputTypes,omitempty"`
	AlgoGas              uint64            `json:"algoGas,omitempty"`
	InputSummary         map[string]string `json:"inputSummary,omitempty"`
	ExpectedOutput       string            `json:"expectedOutput,omitempty"`
	SenderNodeID         string            `json:"senderNodeId"`
	SenderRPC            string            `json:"senderRpc"`
	From                 string            `json:"from"`
	StartedAt            string            `json:"startedAt"`
	FinishedAt           string            `json:"finishedAt,omitempty"`
	SubmissionStartedAt  string            `json:"submissionStartedAt,omitempty"`
	SubmissionReturnedAt string            `json:"submissionReturnedAt,omitempty"`
	SubmitBlockNumber    uint64            `json:"submitBlockNumber,omitempty"`
	TransactionHash      string            `json:"transactionHash,omitempty"`
	ReceiptStatus        uint64            `json:"receiptStatus,omitempty"`
	ReceiptGasUsed       uint64            `json:"receiptGasUsed,omitempty"`
	ReceiptLogCount      int               `json:"receiptLogCount,omitempty"`
	ReceiptBlockNumber   uint64            `json:"receiptBlockNumber,omitempty"`
	ReceiptObservedAt    string            `json:"receiptObservedAt,omitempty"`
	PhaseTimeline        upgradeTimeline   `json:"phaseTimeline"`
	Payload              payloadResult     `json:"payload"`
	Nodes                []nodeRoundResult `json:"nodes"`
	Summary              roundSummary      `json:"summary"`
	Completed            bool              `json:"completed"`
	Error                string            `json:"error,omitempty"`
}

type upgradeTimeline struct {
	TransactionSubmittedAt string `json:"transactionSubmittedAt,omitempty"`
	TxHashObservedAt       string `json:"txHashObservedAt,omitempty"`
	ReceiptObservedAt      string `json:"receiptObservedAt,omitempty"`
	EventObservedAt        string `json:"eventObservedAt,omitempty"`
	ActivationObservedAt   string `json:"activationObservedAt,omitempty"`
}

type payloadResult struct {
	SourceBytes           int    `json:"sourceBytes"`
	CompressedGzipBytes   int    `json:"compressedGzipBytes"`
	CompressedBase64Bytes int    `json:"compressedBase64Bytes"`
	UploadCalldataBytes   int    `json:"uploadCalldataBytes"`
	UploadCalldataSHA256  string `json:"uploadCalldataSha256"`
}

type nodeRoundResult struct {
	ID                        string  `json:"id"`
	Role                      string  `json:"role"`
	RPCURL                    string  `json:"rpcURL"`
	PreflightBlock            uint64  `json:"preflightBlock,omitempty"`
	ReceiptObservedAt         string  `json:"receiptObservedAt,omitempty"`
	ReceiptBlockNumber        uint64  `json:"receiptBlockNumber,omitempty"`
	ReceiptLatestBlockNumber  uint64  `json:"receiptLatestBlockNumber,omitempty"`
	CompletionBlockNumber     uint64  `json:"completionBlockNumber,omitempty"`
	CompletedAt               string  `json:"completedAt,omitempty"`
	ValidationStartedAt       string  `json:"validationStartedAt,omitempty"`
	ValidationFinishedAt      string  `json:"validationFinishedAt,omitempty"`
	ValidationLatencyMillis   float64 `json:"validationLatencyMillis,omitempty"`
	ValidationOutputHex       string  `json:"validationOutputHex,omitempty"`
	DecodedResult             string  `json:"decodedResult,omitempty"`
	ExpectedResult            string  `json:"expectedResult,omitempty"`
	SubmitToReceiptMillis     float64 `json:"submitToReceiptMillis,omitempty"`
	TxHashToReceiptMillis     float64 `json:"txHashToReceiptMillis,omitempty"`
	SenderReceiptToReceiptMS  float64 `json:"senderReceiptToReceiptMillis,omitempty"`
	ReceiptToCompleteMillis   float64 `json:"receiptToCompleteMillis,omitempty"`
	SubmitToCompleteMillis    float64 `json:"submitToCompleteMillis,omitempty"`
	TxHashToCompleteMillis    float64 `json:"txHashToCompleteMillis,omitempty"`
	SenderReceiptToCompleteMS float64 `json:"senderReceiptToCompleteMillis,omitempty"`
	EventToCompleteMillis     float64 `json:"eventToCompleteMillis,omitempty"`
	Completed                 bool    `json:"completed"`
	TimedOut                  bool    `json:"timedOut,omitempty"`
	Error                     string  `json:"error,omitempty"`

	receiptTime  time.Time
	completeTime time.Time
}

type roundSummary struct {
	TargetNodeCount                   int     `json:"targetNodeCount"`
	CompletedNodeCount                int     `json:"completedNodeCount"`
	SubmitToTxHashMillis              float64 `json:"submitToTxHashMillis,omitempty"`
	SubmitToSenderReceiptMS           float64 `json:"submitToSenderReceiptMillis,omitempty"`
	TxHashToSenderReceiptMillis       float64 `json:"txHashToSenderReceiptMillis,omitempty"`
	SubmitToAllReceiptMillis          float64 `json:"submitToAllReceiptMillis,omitempty"`
	TxHashToAllReceiptMillis          float64 `json:"txHashToAllReceiptMillis,omitempty"`
	SenderReceiptToAllReceiptMillis   float64 `json:"senderReceiptToAllReceiptMillis,omitempty"`
	SubmitToAllCompleteMillis         float64 `json:"submitToAllCompleteMillis,omitempty"`
	TxHashToAllCompleteMillis         float64 `json:"txHashToAllCompleteMillis,omitempty"`
	SenderReceiptToAllCompleteMillis  float64 `json:"senderReceiptToAllCompleteMillis,omitempty"`
	ReceiptToEventMillis              float64 `json:"receiptToEventMillis,omitempty"`
	EventToAllCompleteMillis          float64 `json:"eventToAllCompleteMillis,omitempty"`
	AllReceiptToAllCompleteMillis     float64 `json:"allReceiptToAllCompleteMillis,omitempty"`
	MeanNodeSubmitToReceiptMillis     float64 `json:"meanNodeSubmitToReceiptMillis,omitempty"`
	MeanNodeTxHashToReceiptMillis     float64 `json:"meanNodeTxHashToReceiptMillis,omitempty"`
	MeanNodeSenderReceiptToReceiptMS  float64 `json:"meanNodeSenderReceiptToReceiptMillis,omitempty"`
	MeanNodeReceiptToCompleteMillis   float64 `json:"meanNodeReceiptToCompleteMillis,omitempty"`
	MeanNodeSubmitToCompleteMillis    float64 `json:"meanNodeSubmitToCompleteMillis,omitempty"`
	MeanNodeTxHashToCompleteMillis    float64 `json:"meanNodeTxHashToCompleteMillis,omitempty"`
	MeanNodeSenderReceiptToCompleteMS float64 `json:"meanNodeSenderReceiptToCompleteMillis,omitempty"`
	SlowestNodeID                     string  `json:"slowestNodeId,omitempty"`
	Completed                         bool    `json:"completed"`
}

type experimentSummary struct {
	RoundCount            int     `json:"roundCount"`
	CompletedRoundCount   int     `json:"completedRoundCount"`
	TargetNodeCount       int     `json:"targetNodeCount"`
	Completed             bool    `json:"completed"`
	MeanAllCompleteMillis float64 `json:"meanAllCompleteMillis,omitempty"`
	MaxAllCompleteMillis  float64 `json:"maxAllCompleteMillis,omitempty"`
	MinAllCompleteMillis  float64 `json:"minAllCompleteMillis,omitempty"`
}

type roundSource struct {
	Algorithm   string
	UpgradeName string
	Path        string
	Raw         []byte
	Fixture     algorithmFixture
}

type payloadData struct {
	Encoded string
	Upload  []byte
	Result  payloadResult
}

type validationProbe struct {
	Algorithm      string
	CallData       []byte
	EncodedInput   []byte
	OutputArgs     abi.Arguments
	ExpectedRaw    []byte
	ExpectedValues []interface{}
	ExpectedText   string
}

type algorithmFixture struct {
	Algorithm      string
	UpgradeName    string
	SourceFunc     string
	SourcePath     string
	InputTypes     []string
	OutputTypes    []string
	Values         []interface{}
	ExpectedValues []interface{}
	InputSummary   map[string]string
	AlgoGas        uint64
}

type receiptObservation struct {
	Receipt    *types.Receipt
	ObservedAt time.Time
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "benchupgradelatency: %v\n", err)
		os.Exit(2)
	}
	res := newResult(cfg)
	err = run(context.Background(), cfg, &res)
	if err != nil {
		res.Error = err.Error()
	}
	if writeErr := writeJSONResult(cfg.outputJSON, res); writeErr != nil {
		if err == nil {
			err = writeErr
			res.Error = writeErr.Error()
			_ = writeJSONResult(cfg.outputJSON, res)
		} else {
			fmt.Fprintf(os.Stderr, "benchupgradelatency: failed to write JSON result: %v\n", writeErr)
		}
	}
	if csvErr := writeCSVResults(cfg, res); csvErr != nil {
		if err == nil {
			err = csvErr
			res.Error = csvErr.Error()
			_ = writeJSONResult(cfg.outputJSON, res)
		} else {
			fmt.Fprintf(os.Stderr, "benchupgradelatency: failed to write CSV result: %v\n", csvErr)
		}
	}
	printSummary(res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchupgradelatency: %v\n", err)
		os.Exit(1)
	}
}

func parseConfig(args []string) (config, error) {
	cfg := config{}
	fs := flag.NewFlagSet("benchupgradelatency", flag.ContinueOnError)
	fs.StringVar(&cfg.repoRoot, "repo-root", ".", "repository root")
	fs.StringVar(&cfg.configPath, "config", defaultConfig, "multi-node network YAML config path")
	fs.StringVar(&cfg.resultDir, "out", "", "directory for raw JSON/CSV results and generated sources")
	fs.StringVar(&cfg.outputJSON, "output-json", "", "write raw experiment JSON to this path")
	fs.StringVar(&cfg.outputCSV, "output-csv", "", "write round-level CSV to this path")
	fs.StringVar(&cfg.outputNodeCSV, "output-node-csv", "", "write node-level CSV to this path")
	fs.StringVar(&cfg.senderID, "sender", "", "node ID used to submit the upgrade transaction; defaults to the first signer")
	fs.StringVar(&cfg.nodeIDs, "nodes", "", "comma-separated node IDs to observe; defaults to all configured nodes")
	fs.StringVar(&cfg.from, "from", "", "transaction sender address; defaults to the sender node account")
	fs.StringVar(&cfg.sourcePath, "source", "cryptoupgrade/algorithm/go/add.go", "Add-compatible Go source")
	fs.StringVar(&cfg.algorithm, "algorithm", "Add", "algorithm name to upload")
	fs.StringVar(&cfg.algorithms, "algorithms", "", "comma-separated built-in algorithms to run; use all for all fixtures")
	fs.BoolVar(&cfg.uniqueNames, "unique-names", true, "append a per-run suffix to fixture upload function names to avoid polluted baselines")
	fs.StringVar(&cfg.inputType, "input-type", "uint256,uint256", "CodeStorage input ABI type list")
	fs.StringVar(&cfg.outputType, "output-type", "uint256", "CodeStorage output ABI type list")
	fs.Uint64Var(&cfg.algoGas, "algo-gas", 1, "algorithm gas recorded in CodeStorage")
	fs.Uint64Var(&cfg.txGas, "tx-gas", 5000000, "upload transaction gas limit")
	fs.Uint64Var(&cfg.callGas, "call-gas", 5000000, "eth_call gas limit for validation")
	fs.StringVar(&cfg.a, "a", "100", "first Add argument")
	fs.StringVar(&cfg.b, "b", "100", "second Add argument")
	fs.IntVar(&cfg.rounds, "rounds", 1, "number of upgrade latency rounds")
	fs.DurationVar(&cfg.pollInterval, "poll-interval", 200*time.Millisecond, "per-node receipt and callFunc polling interval")
	fs.DurationVar(&cfg.timeout, "timeout", 2*time.Minute, "per-round timeout after transaction submission")
	fs.DurationVar(&cfg.preflightTimeout, "preflight-timeout", 30*time.Second, "timeout for network preflight")
	fs.BoolVar(&cfg.preflight, "preflight", true, "check RPC, chain ID, peers, and block growth before measuring")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	var err error
	cfg.repoRoot, err = filepath.Abs(cfg.repoRoot)
	if err != nil {
		return cfg, err
	}
	if !filepath.IsAbs(cfg.configPath) {
		cfg.configPath = filepath.Join(cfg.repoRoot, cfg.configPath)
	}
	if !filepath.IsAbs(cfg.sourcePath) {
		cfg.sourcePath = filepath.Join(cfg.repoRoot, cfg.sourcePath)
	}
	timestamp := time.Now().Format("20060102-150405")
	if cfg.resultDir == "" {
		cfg.resultDir = filepath.Join(cfg.repoRoot, "experiments", "cryptoupgrade", "results", "upgrade-latency", "run-"+timestamp)
	} else if !filepath.IsAbs(cfg.resultDir) {
		cfg.resultDir = filepath.Join(cfg.repoRoot, cfg.resultDir)
	}
	if err := os.MkdirAll(cfg.resultDir, 0755); err != nil {
		return cfg, fmt.Errorf("create result dir: %w", err)
	}
	if cfg.outputJSON == "" {
		cfg.outputJSON = filepath.Join(cfg.resultDir, "result.json")
	} else if !filepath.IsAbs(cfg.outputJSON) {
		cfg.outputJSON = filepath.Join(cfg.repoRoot, cfg.outputJSON)
	}
	if cfg.outputCSV == "" {
		cfg.outputCSV = filepath.Join(cfg.resultDir, "rounds.csv")
	} else if !filepath.IsAbs(cfg.outputCSV) {
		cfg.outputCSV = filepath.Join(cfg.repoRoot, cfg.outputCSV)
	}
	if cfg.outputNodeCSV == "" {
		cfg.outputNodeCSV = filepath.Join(cfg.resultDir, "nodes.csv")
	} else if !filepath.IsAbs(cfg.outputNodeCSV) {
		cfg.outputNodeCSV = filepath.Join(cfg.repoRoot, cfg.outputNodeCSV)
	}
	if _, err := os.Stat(cfg.configPath); err != nil {
		return cfg, fmt.Errorf("config file %s: %w", cfg.configPath, err)
	}
	if _, err := os.Stat(cfg.sourcePath); err != nil {
		return cfg, fmt.Errorf("source file %s: %w", cfg.sourcePath, err)
	}
	if cfg.rounds <= 0 {
		return cfg, errors.New("-rounds must be positive")
	}
	if cfg.pollInterval <= 0 {
		return cfg, errors.New("-poll-interval must be positive")
	}
	if cfg.timeout <= 0 {
		return cfg, errors.New("-timeout must be positive")
	}
	if cfg.preflightTimeout <= 0 {
		return cfg, errors.New("-preflight-timeout must be positive")
	}
	if cfg.txGas == 0 {
		return cfg, errors.New("-tx-gas must be non-zero; gas estimation can mutate plugin state")
	}
	if cfg.callGas == 0 {
		return cfg, errors.New("-call-gas must be non-zero")
	}
	cfg.algorithm = exportedIdentifier(cfg.algorithm)
	cfg.algorithms = strings.TrimSpace(cfg.algorithms)
	if cfg.algorithm == "" && cfg.algorithms == "" {
		return cfg, errors.New("-algorithm must contain at least one letter")
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return cfg, fmt.Errorf("invalid -from address %q", cfg.from)
	}
	if cfg.algorithms == "" {
		a, err := parseBig("a", cfg.a)
		if err != nil {
			return cfg, err
		}
		b, err := parseBig("b", cfg.b)
		if err != nil {
			return cfg, err
		}
		expected := new(big.Int).Add(a, b)
		if _, _, _, err := buildValidationProbe(cfg.algorithm, splitABITypeList(cfg.inputType), splitABITypeList(cfg.outputType), []interface{}{a, b}, []interface{}{expected}); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func newResult(cfg config) experimentResult {
	return experimentResult{
		Experiment: experimentName,
		Change:     changeName,
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Command:    os.Args,
		Artifacts: artifactResult{
			ResultDir:    cfg.resultDir,
			ResultJSON:   cfg.outputJSON,
			RoundCSV:     cfg.outputCSV,
			NodeCSV:      cfg.outputNodeCSV,
			SourcePath:   cfg.sourcePath,
			GeneratedDir: filepath.Join(cfg.resultDir, "sources"),
		},
		Limitations: []string{
			"升级耗时由实验控制端通过 JSON-RPC 轮询观测，包含 RPC 往返和 poll interval 带来的观测误差。",
			"节点 completed 判据为该节点 eth_call CodeStorage.callFunc 返回期望结果，说明该节点本地算法信息和 plugin 已可用于调用。",
			"节点日志仅用于排查，不作为跨节点耗时计算的权威时间源。",
			"preflight 阶段不执行 cryptoupgrade smoke test，避免在正式测量前上传同名算法污染结果。",
		},
	}
}

func run(ctx context.Context, cfg config, res *experimentResult) error {
	netCfg, err := network.LoadConfig(cfg.configPath)
	if err != nil {
		return fmt.Errorf("load network config: %w", err)
	}
	selected, err := selectNetwork(netCfg, cfg.senderID, cfg.nodeIDs, cfg.from)
	if err != nil {
		return err
	}
	fillConfigResult(res, cfg, selected)
	if cfg.preflight {
		preflight, err := runPreflight(ctx, selected, cfg.preflightTimeout)
		res.Preflight = preflight
		if err != nil {
			return err
		}
	}
	fixtures, err := selectAlgorithmFixtures(cfg)
	if err != nil {
		return err
	}
	roundIndex := 1
	for repeat := 1; repeat <= cfg.rounds; repeat++ {
		for _, fixture := range fixtures {
			rr, err := runRound(ctx, cfg, selected, roundIndex, repeat, fixture)
			res.Rounds = append(res.Rounds, rr)
			if err != nil {
				res.Summary = summarizeExperiment(res.Rounds, len(selected.targets))
				return err
			}
			roundIndex++
		}
	}
	res.Summary = summarizeExperiment(res.Rounds, len(selected.targets))
	res.Completed = res.Summary.Completed
	return nil
}

func selectedAlgorithmNames(cfg config) []string {
	fixtures, err := selectAlgorithmFixtures(cfg)
	if err != nil {
		if cfg.algorithms != "" {
			return splitAlgorithmList(cfg.algorithms)
		}
		return []string{cfg.algorithm}
	}
	names := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		names = append(names, fixture.Algorithm)
	}
	return names
}

func selectAlgorithmFixtures(cfg config) ([]algorithmFixture, error) {
	fixtures, err := builtinFixtures(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.algorithms == "" {
		a, err := parseBig("a", cfg.a)
		if err != nil {
			return nil, err
		}
		b, err := parseBig("b", cfg.b)
		if err != nil {
			return nil, err
		}
		fixture := algorithmFixture{
			Algorithm:      cfg.algorithm,
			UpgradeName:    cfg.algorithm,
			SourceFunc:     "Add",
			SourcePath:     cfg.sourcePath,
			InputTypes:     splitABITypeList(cfg.inputType),
			OutputTypes:    splitABITypeList(cfg.outputType),
			Values:         []interface{}{a, b},
			ExpectedValues: []interface{}{new(big.Int).Add(a, b)},
			InputSummary: map[string]string{
				"a":        a.String(),
				"b":        b.String(),
				"expected": new(big.Int).Add(a, b).String(),
			},
			AlgoGas: cfg.algoGas,
		}
		return []algorithmFixture{fixture}, nil
	}
	requested := splitAlgorithmList(cfg.algorithms)
	if len(requested) == 0 || (len(requested) == 1 && algorithmKey(requested[0]) == "all") {
		return fixtures, nil
	}
	byName := make(map[string]algorithmFixture, len(fixtures))
	for _, fixture := range fixtures {
		byName[algorithmKey(fixture.Algorithm)] = fixture
		byName[algorithmKey(fixture.UpgradeName)] = fixture
	}
	var selected []algorithmFixture
	for _, name := range requested {
		fixture, ok := byName[algorithmKey(name)]
		if !ok {
			return nil, fmt.Errorf("unknown algorithm %q", name)
		}
		selected = append(selected, fixture)
	}
	return selected, nil
}

func splitAlgorithmList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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

func algorithmKey(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, "_", "")
	return raw
}

func parseBig(label, raw string) (*big.Int, error) {
	value, ok := new(big.Int).SetString(strings.TrimSpace(raw), 10)
	if !ok {
		return nil, fmt.Errorf("invalid -%s value %q", label, raw)
	}
	return value, nil
}

func builtinFixtures(cfg config) ([]algorithmFixture, error) {
	a, err := parseBig("a", cfg.a)
	if err != nil {
		return nil, err
	}
	b, err := parseBig("b", cfg.b)
	if err != nil {
		return nil, err
	}
	expectedAdd := new(big.Int).Add(a, b)
	sampleData := []byte("hello cryptoupgrade")
	sha256Sum := sha256.Sum256(sampleData)
	blake2bSum := gethblake2b.Sum256(sampleData)
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

	return []algorithmFixture{
		{
			Algorithm:      "Add",
			UpgradeName:    "Add",
			SourceFunc:     "Add",
			SourcePath:     "cryptoupgrade/algorithm/go/add.go",
			InputTypes:     []string{"uint256", "uint256"},
			OutputTypes:    []string{"uint256"},
			Values:         []interface{}{a, b},
			ExpectedValues: []interface{}{expectedAdd},
			InputSummary: map[string]string{
				"a":        a.String(),
				"b":        b.String(),
				"expected": expectedAdd.String(),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:      "Sha256",
			UpgradeName:    "Sha256",
			SourceFunc:     "Sha256",
			SourcePath:     "cryptoupgrade/algorithm/go/sha256.go",
			InputTypes:     []string{"bytes"},
			OutputTypes:    []string{"bytes"},
			Values:         []interface{}{sampleData},
			ExpectedValues: []interface{}{append([]byte(nil), sha256Sum[:]...)},
			InputSummary: map[string]string{
				"dataHex":   hexutil.Encode(sampleData),
				"outputHex": hexutil.Encode(sha256Sum[:]),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:      "Blake2bSum256",
			UpgradeName:    "Sum256",
			SourceFunc:     "Sum256",
			SourcePath:     "cryptoupgrade/algorithm/go/blake2b.go",
			InputTypes:     []string{"bytes"},
			OutputTypes:    []string{"bytes32"},
			Values:         []interface{}{sampleData},
			ExpectedValues: []interface{}{blake2bSum},
			InputSummary: map[string]string{
				"dataHex":   hexutil.Encode(sampleData),
				"outputHex": hexutil.Encode(blake2bSum[:]),
			},
			AlgoGas: 3000,
		},
		{
			Algorithm:      "Pbkdf2Sha256",
			UpgradeName:    "Pbkdf2Sha256",
			SourceFunc:     "Pbkdf2Sha256",
			SourcePath:     "cryptoupgrade/algorithm/go/pbkdf2_sha256.go",
			InputTypes:     []string{"bytes", "bytes", "uint256", "uint256"},
			OutputTypes:    []string{"bytes"},
			Values:         []interface{}{pbkdf2Password, pbkdf2Salt, pbkdf2Iterations, pbkdf2KeyLength},
			ExpectedValues: []interface{}{pbkdf2Sha256Expected(pbkdf2Password, pbkdf2Salt, pbkdf2Iterations, pbkdf2KeyLength)},
			InputSummary: map[string]string{
				"passwordHex": hexutil.Encode(pbkdf2Password),
				"saltHex":     hexutil.Encode(pbkdf2Salt),
				"iterations":  pbkdf2Iterations.String(),
				"keyLength":   pbkdf2KeyLength.String(),
			},
			AlgoGas: 25000,
		},
		{
			Algorithm:      "Dh2048Secret",
			UpgradeName:    "Dh2048Secret",
			SourceFunc:     "Dh2048Secret",
			SourcePath:     "cryptoupgrade/algorithm/go/dh2048.go",
			InputTypes:     []string{"bytes", "bytes"},
			OutputTypes:    []string{"bytes"},
			Values:         []interface{}{dhPrivate, dhPeerPublic},
			ExpectedValues: []interface{}{dh2048Secret(dhPrivate, dhPeerPublic)},
			InputSummary: map[string]string{
				"privateKeyHex":        hexutil.Encode(dhPrivate),
				"peerPublicKeySha256":  sha256Hex(dhPeerPublic),
				"peerPublicKeyByteLen": fmt.Sprintf("%d", len(dhPeerPublic)),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:      "PedersenCommit",
			UpgradeName:    "PedersenCommit",
			SourceFunc:     "PedersenCommit",
			SourcePath:     "cryptoupgrade/algorithm/go/pedersen_commit.go",
			InputTypes:     []string{"bytes", "bytes"},
			OutputTypes:    []string{"bytes"},
			Values:         []interface{}{pedersenMessage, pedersenBlinding},
			ExpectedValues: []interface{}{pedersenCommit(pedersenMessage, pedersenBlinding)},
			InputSummary: map[string]string{
				"messageHex":  hexutil.Encode(pedersenMessage),
				"blindingHex": hexutil.Encode(pedersenBlinding),
			},
			AlgoGas: 200000,
		},
		{
			Algorithm:      "SchnorrVerify",
			UpgradeName:    "SchnorrVerify",
			SourceFunc:     "SchnorrVerify",
			SourcePath:     "cryptoupgrade/algorithm/go/schnorr_proof.go",
			InputTypes:     []string{"bytes", "bytes"},
			OutputTypes:    []string{"bool"},
			Values:         []interface{}{schnorrMessage, schnorrProof},
			ExpectedValues: []interface{}{true},
			InputSummary: map[string]string{
				"messageHex":      hexutil.Encode(schnorrMessage),
				"proofSha256":     sha256Hex(schnorrProof),
				"proofByteLength": fmt.Sprintf("%d", len(schnorrProof)),
			},
			AlgoGas: 200000,
		},
	}, nil
}

func fillConfigResult(res *experimentResult, cfg config, selected selectedNetwork) {
	targetIDs := make([]string, 0, len(selected.targets))
	targetSet := make(map[string]struct{}, len(selected.targets))
	for _, node := range selected.targets {
		targetIDs = append(targetIDs, node.ID)
		targetSet[node.ID] = struct{}{}
	}
	var excluded []string
	for _, node := range selected.cfg.Nodes {
		if _, ok := targetSet[node.ID]; !ok {
			excluded = append(excluded, node.ID)
		}
	}
	res.Config = experimentConfig{
		ConfigPath:          selected.cfg.ConfigPath(),
		ChainID:             selected.cfg.Network.ChainID,
		NetworkID:           selected.cfg.Network.NetworkID,
		ConsensusType:       selected.cfg.Consensus.Type,
		ConsensusPeriod:     selected.cfg.Consensus.Period,
		NodeCount:           len(selected.cfg.Nodes),
		TargetNodeIDs:       targetIDs,
		ExcludedNodeIDs:     excluded,
		SenderNodeID:        selected.sender.ID,
		SenderRPC:           network.RPCURL(selected.sender),
		From:                selected.from.Hex(),
		Rounds:              cfg.rounds,
		PollIntervalMillis:  millis(cfg.pollInterval),
		TimeoutMillis:       millis(cfg.timeout),
		Preflight:           cfg.preflight,
		PreflightTimeoutMS:  millis(cfg.preflightTimeout),
		AlgorithmBase:       cfg.algorithm,
		Algorithms:          selectedAlgorithmNames(cfg),
		UniqueNames:         cfg.uniqueNames,
		InputType:           cfg.inputType,
		OutputType:          cfg.outputType,
		AlgoGas:             cfg.algoGas,
		TransactionGasLimit: cfg.txGas,
		CallGas:             cfg.callGas,
	}
}

func selectNetwork(cfg *network.Config, senderID, nodeIDs, from string) (selectedNetwork, error) {
	sender, err := selectSender(cfg.Nodes, senderID)
	if err != nil {
		return selectedNetwork{}, err
	}
	targets, err := selectTargetNodes(cfg.Nodes, nodeIDs)
	if err != nil {
		return selectedNetwork{}, err
	}
	var fromAddr common.Address
	switch {
	case from != "":
		fromAddr = common.HexToAddress(from)
	case common.IsHexAddress(sender.Account):
		fromAddr = common.HexToAddress(sender.Account)
	default:
		return selectedNetwork{}, fmt.Errorf("sender node %s has no account; set -from explicitly", sender.ID)
	}
	return selectedNetwork{cfg: cfg, sender: sender, targets: targets, from: fromAddr}, nil
}

func selectSender(nodes []network.NodeConfig, senderID string) (network.NodeConfig, error) {
	if senderID != "" {
		for _, node := range nodes {
			if node.ID == senderID {
				return node, nil
			}
		}
		return network.NodeConfig{}, fmt.Errorf("sender node %q not found", senderID)
	}
	for _, node := range nodes {
		if node.Role == "signer" && common.IsHexAddress(node.Account) {
			return node, nil
		}
	}
	return network.NodeConfig{}, errors.New("no signer node with account found; set -sender and -from explicitly")
}

func selectTargetNodes(nodes []network.NodeConfig, nodeIDs string) ([]network.NodeConfig, error) {
	if strings.TrimSpace(nodeIDs) == "" {
		if len(nodes) == 0 {
			return nil, errors.New("network config contains no nodes")
		}
		return append([]network.NodeConfig(nil), nodes...), nil
	}
	byID := make(map[string]network.NodeConfig, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	seen := make(map[string]struct{})
	var out []network.NodeConfig
	for _, raw := range strings.Split(nodeIDs, ",") {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("duplicate target node %q", id)
		}
		node, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("target node %q not found", id)
		}
		seen[id] = struct{}{}
		out = append(out, node)
	}
	if len(out) == 0 {
		return nil, errors.New("no target nodes selected")
	}
	return out, nil
}

func runPreflight(parent context.Context, selected selectedNetwork, timeout time.Duration) (*preflightResult, error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	result := &preflightResult{
		StartedAt:         started.Format(time.RFC3339Nano),
		ExpectedPeerCount: max(0, len(selected.cfg.Nodes)-1),
		OK:                true,
	}
	wait := time.Duration(selected.cfg.Consensus.Period+1) * time.Second
	results := make([]preflightNode, len(selected.targets))
	var wg sync.WaitGroup
	for i, node := range selected.targets {
		wg.Add(1)
		go func(i int, node network.NodeConfig) {
			defer wg.Done()
			results[i] = preflightOneNode(ctx, selected.cfg, node, wait, result.ExpectedPeerCount)
		}(i, node)
	}
	wg.Wait()
	result.Nodes = results
	for _, node := range result.Nodes {
		if node.Error != "" {
			result.OK = false
		}
	}
	finished := time.Now()
	result.FinishedAt = finished.Format(time.RFC3339Nano)
	result.DurationMillis = millis(finished.Sub(started))
	if !result.OK {
		result.Error = "preflight failed"
		return result, errors.New("preflight failed; inspect result.json for node-level errors")
	}
	return result, nil
}

func preflightOneNode(ctx context.Context, cfg *network.Config, node network.NodeConfig, wait time.Duration, expectedPeerCount int) preflightNode {
	var last preflightNode
	for {
		result := preflightOneNodeOnce(ctx, cfg, node, wait, expectedPeerCount)
		if result.Error == "" {
			return result
		}
		last = result
		select {
		case <-ctx.Done():
			if last.Error == "" {
				last.Error = ctx.Err().Error()
			}
			return last
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func preflightOneNodeOnce(ctx context.Context, cfg *network.Config, node network.NodeConfig, wait time.Duration, expectedPeerCount int) preflightNode {
	out := preflightNode{ID: node.ID, Role: node.Role, RPCURL: network.RPCURL(node)}
	client, err := rpc.DialContext(ctx, out.RPCURL)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer client.Close()
	var chainID hexutil.Big
	if err := client.CallContext(ctx, &chainID, "eth_chainId"); err != nil {
		out.Error = err.Error()
		return out
	}
	out.ChainID = (*big.Int)(&chainID).Uint64()
	if out.ChainID != cfg.Network.ChainID {
		out.Error = fmt.Sprintf("chain id mismatch: want %d got %d", cfg.Network.ChainID, out.ChainID)
		return out
	}
	peerCount, err := readPeerCount(ctx, client)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.PeerCount = peerCount
	start, err := blockNumber(ctx, client)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.StartBlock = start
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		out.Error = ctx.Err().Error()
		return out
	case <-timer.C:
	}
	end, err := blockNumber(ctx, client)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.EndBlock = end
	// 10 节点网络启动时 peer 连接可能在出块等待期间继续收敛，
	// 因此使用等待后的最终 peer count 作为 preflight 判定依据。
	peerCount, err = readPeerCount(ctx, client)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.PeerCount = peerCount
	if int(out.PeerCount) < expectedPeerCount {
		out.Error = fmt.Sprintf("peer count too low: want >= %d got %d", expectedPeerCount, out.PeerCount)
		return out
	}
	if out.EndBlock <= out.StartBlock {
		out.Error = fmt.Sprintf("block height did not increase: start %d end %d", out.StartBlock, out.EndBlock)
		return out
	}
	return out
}

func runRound(parent context.Context, cfg config, selected selectedNetwork, round, repeat int, fixture algorithmFixture) (roundResult, error) {
	started := time.Now()
	rr := roundResult{
		Index:        round,
		SenderNodeID: selected.sender.ID,
		SenderRPC:    network.RPCURL(selected.sender),
		From:         selected.from.Hex(),
		StartedAt:    started.Format(time.RFC3339Nano),
	}
	source, err := prepareRoundSource(cfg, round, repeat, fixture)
	if err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, err
	}
	rr.Algorithm = source.Algorithm
	rr.UpgradeName = source.UpgradeName
	rr.SourcePath = source.Path
	rr.ABIInputTypes = append([]string(nil), source.Fixture.InputTypes...)
	rr.ABIOutputTypes = append([]string(nil), source.Fixture.OutputTypes...)
	rr.AlgoGas = source.Fixture.AlgoGas
	rr.InputSummary = cloneStringMap(source.Fixture.InputSummary)
	probe, _, _, err := buildValidationProbe(source.UpgradeName, source.Fixture.InputTypes, source.Fixture.OutputTypes, source.Fixture.Values, source.Fixture.ExpectedValues)
	if err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, err
	}
	rr.ExpectedOutput = probe.ExpectedText
	if err := ensureAlgorithmUnavailable(parent, selected.targets, selected.from, probe, cfg); err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, err
	}
	payload, err := buildPayload(source.Raw, source.UpgradeName, source.Fixture)
	if err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, err
	}
	rr.Payload = payload.Result
	senderClient, err := rpc.DialContext(parent, rr.SenderRPC)
	if err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, fmt.Errorf("dial sender RPC %s: %w", rr.SenderRPC, err)
	}
	defer senderClient.Close()
	submitBlock, err := blockNumber(parent, senderClient)
	if err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, fmt.Errorf("read submit block: %w", err)
	}
	rr.SubmitBlockNumber = submitBlock
	to := common.CodeStorageAddress
	gas := hexutil.Uint64(cfg.txGas)
	submitStarted := time.Now()
	rr.SubmissionStartedAt = submitStarted.Format(time.RFC3339Nano)
	txHash, err := sendTransaction(parent, senderClient, txArgs{
		From: selected.from,
		To:   &to,
		Gas:  &gas,
		Data: payload.Upload,
	})
	submitReturned := time.Now()
	rr.SubmissionReturnedAt = submitReturned.Format(time.RFC3339Nano)
	if err != nil {
		rr.Error = err.Error()
		rr.FinishedAt = time.Now().Format(time.RFC3339Nano)
		return rr, fmt.Errorf("send upgrade transaction: %w", wrapPluginHint(err))
	}
	rr.TransactionHash = txHash.Hex()

	roundCtx, cancel := context.WithTimeout(parent, cfg.timeout)
	defer cancel()
	senderReceiptCh := make(chan receiptObservation, 1)
	senderErrCh := make(chan error, 1)
	go func() {
		receipt, observed, err := waitReceipt(roundCtx, rr.SenderRPC, txHash, cfg.pollInterval)
		if err != nil {
			senderErrCh <- err
			return
		}
		senderReceiptCh <- receiptObservation{Receipt: receipt, ObservedAt: observed}
	}()

	nodeResults := observeTargets(roundCtx, selected.targets, selected.from, txHash, probe, cfg, submitStarted)
	rr.Nodes = nodeResults
	var senderReceiptTime time.Time
	var eventObservedTime time.Time
	select {
	case obs := <-senderReceiptCh:
		rr.ReceiptObservedAt = obs.ObservedAt.Format(time.RFC3339Nano)
		senderReceiptTime = obs.ObservedAt
		rr.ReceiptStatus = obs.Receipt.Status
		rr.ReceiptGasUsed = obs.Receipt.GasUsed
		rr.ReceiptLogCount = len(obs.Receipt.Logs)
		if obs.Receipt.BlockNumber != nil {
			rr.ReceiptBlockNumber = obs.Receipt.BlockNumber.Uint64()
		}
		if receiptContainsCodeUploadedEvent(obs.Receipt, rr.UpgradeName) {
			eventObservedTime = obs.ObservedAt
		}
	case err := <-senderErrCh:
		rr.Error = err.Error()
	case <-roundCtx.Done():
		rr.Error = roundCtx.Err().Error()
	}
	annotateNodeStages(rr.Nodes, submitReturned, senderReceiptTime, eventObservedTime)
	rr.Summary = summarizeRound(rr.Nodes, submitStarted, submitReturned, senderReceiptTime, eventObservedTime)
	rr.PhaseTimeline = buildUpgradeTimeline(submitStarted, submitReturned, senderReceiptTime, eventObservedTime, rr.Nodes)
	rr.Completed = rr.Summary.Completed && rr.ReceiptStatus == types.ReceiptStatusSuccessful && rr.Error == ""
	finished := time.Now()
	rr.FinishedAt = finished.Format(time.RFC3339Nano)
	if rr.ReceiptStatus != 0 && rr.ReceiptStatus != types.ReceiptStatusSuccessful {
		rr.Error = fmt.Sprintf("upgrade transaction failed with status %d", rr.ReceiptStatus)
	}
	if !rr.Completed && rr.Error == "" {
		rr.Error = "round incomplete; at least one target node did not complete"
	}
	if rr.Error != "" {
		return rr, errors.New(rr.Error)
	}
	return rr, nil
}

func prepareRoundSource(cfg config, round, repeat int, fixture algorithmFixture) (roundSource, error) {
	sourcePath := fixture.SourcePath
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(cfg.repoRoot, sourcePath)
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return roundSource{}, fmt.Errorf("read source: %w", err)
	}
	uploadName := fixture.UpgradeName
	if shouldUseUniqueName(cfg) {
		uploadName = uniqueUploadName(fixture.UpgradeName, fixture.Algorithm, repeat)
	}
	sourceFunc := fixture.SourceFunc
	if sourceFunc == "" {
		sourceFunc = fixture.UpgradeName
	}
	if uploadName == sourceFunc {
		return roundSource{Algorithm: fixture.Algorithm, UpgradeName: uploadName, Path: sourcePath, Raw: raw, Fixture: fixture}, nil
	}
	raw, err = renameExportedFunction(raw, sourceFunc, uploadName)
	if err != nil {
		return roundSource{}, err
	}
	dir := filepath.Join(cfg.resultDir, "sources")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return roundSource{}, fmt.Errorf("create generated source dir: %w", err)
	}
	path := filepath.Join(dir, strings.ToLower(uploadName)+".go")
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return roundSource{}, fmt.Errorf("write generated source: %w", err)
	}
	return roundSource{Algorithm: fixture.Algorithm, UpgradeName: uploadName, Path: path, Raw: raw, Fixture: fixture}, nil
}

func shouldUseUniqueName(cfg config) bool {
	return cfg.uniqueNames && (cfg.algorithms != "" || cfg.rounds > 1)
}

func roundAlgorithmName(base string, round, rounds int) string {
	base = exportedIdentifier(base)
	if rounds <= 1 {
		return base
	}
	stem := strings.TrimSuffix(base, "Latency")
	return fmt.Sprintf("%sLatency%03d", stem, round)
}

func uniqueUploadName(base, algorithm string, repeat int) string {
	base = exportedIdentifier(base)
	if base == "" {
		base = exportedIdentifier(algorithm)
	}
	stem := strings.TrimSuffix(base, "Latency")
	return fmt.Sprintf("%sLatency%03d", stem, repeat)
}

func renameExportedFunction(source []byte, oldName, newName string) ([]byte, error) {
	oldName = exportedIdentifier(oldName)
	newName = exportedIdentifier(newName)
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("invalid function rename %q -> %q", oldName, newName)
	}
	pattern := regexp.MustCompile(`\bfunc\s+` + regexp.QuoteMeta(oldName) + `\s*\(`)
	matches := pattern.FindAllIndex(source, -1)
	if len(matches) != 1 {
		return nil, fmt.Errorf("source function %s matched %d declarations", oldName, len(matches))
	}
	return pattern.ReplaceAll(source, []byte("func "+newName+"(")), nil
}

func exportedIdentifier(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	re := regexp.MustCompile(`[^A-Za-z0-9_]`)
	parts := re.Split(raw, -1)
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				b.WriteString(part[1:])
			}
		} else {
			b.WriteString(part)
		}
	}
	out := b.String()
	for out != "" && out[0] >= '0' && out[0] <= '9' {
		out = out[1:]
	}
	if out == "" {
		return ""
	}
	if out[0] >= 'a' && out[0] <= 'z' {
		out = strings.ToUpper(out[:1]) + out[1:]
	}
	return out
}

func buildPayload(source []byte, algorithm string, fixture algorithmFixture) (payloadData, error) {
	encoded, err := cryptoupgrade.EncodeSource(source)
	if err != nil {
		return payloadData{}, err
	}
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return payloadData{}, fmt.Errorf("decode shared source payload: %w", err)
	}
	uploadData, err := cryptoupgrade.CodeStorageABI.Pack("uploadCode", algorithm, encoded, fixture.AlgoGas, strings.Join(fixture.InputTypes, ","), strings.Join(fixture.OutputTypes, ","))
	if err != nil {
		return payloadData{}, fmt.Errorf("pack CodeStorage.uploadCode: %w", err)
	}
	sum := sha256.Sum256(uploadData)
	return payloadData{
		Encoded: encoded,
		Upload:  uploadData,
		Result: payloadResult{
			SourceBytes:           len(source),
			CompressedGzipBytes:   len(compressed),
			CompressedBase64Bytes: len(encoded),
			UploadCalldataBytes:   len(uploadData),
			UploadCalldataSHA256:  hex.EncodeToString(sum[:]),
		},
	}, nil
}

func ensureAlgorithmUnavailable(ctx context.Context, targets []network.NodeConfig, from common.Address, probe validationProbe, cfg config) error {
	for _, node := range targets {
		client, err := rpc.DialContext(ctx, network.RPCURL(node))
		if err != nil {
			return fmt.Errorf("dial target node %s for pollution check: %w", node.ID, err)
		}
		output, _, err := callAlgorithm(ctx, client, from, probe, cfg.callGas)
		client.Close()
		if err == nil {
			return fmt.Errorf("algorithm %s is already callable on node %s before upgrade, output=%s", probe.Algorithm, node.ID, output.DecodedResult)
		}
		if isUnexpectedPrecheckError(err) {
			return fmt.Errorf("pre-upgrade callFunc on node %s returned unexpected error: %w", node.ID, err)
		}
	}
	return nil
}

func isUnexpectedPrecheckError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return !(strings.Contains(msg, "not loaded") ||
		strings.Contains(msg, "missing trie node") ||
		strings.Contains(msg, "execution reverted") ||
		strings.Contains(msg, "algorithm ") && strings.Contains(msg, " is not loaded"))
}

func buildValidationProbe(algorithm string, inputTypes, outputTypes []string, values, expectedValues []interface{}) (validationProbe, abi.Arguments, abi.Arguments, error) {
	inputArgs, err := parseABIArguments(inputTypes)
	if err != nil {
		return validationProbe{}, nil, nil, fmt.Errorf("parse -input-type: %w", err)
	}
	if len(inputArgs) != len(values) {
		return validationProbe{}, nil, nil, fmt.Errorf("input type count %d does not match value count %d", len(inputArgs), len(values))
	}
	outputArgs, err := parseABIArguments(outputTypes)
	if err != nil {
		return validationProbe{}, nil, nil, fmt.Errorf("parse -output-type: %w", err)
	}
	if len(outputArgs) != len(expectedValues) {
		return validationProbe{}, nil, nil, fmt.Errorf("output type count %d does not match expected value count %d", len(outputArgs), len(expectedValues))
	}
	encodedInput, err := inputArgs.Pack(values...)
	if err != nil {
		return validationProbe{}, nil, nil, fmt.Errorf("pack %s input: %w", algorithm, err)
	}
	callData, err := cryptoupgrade.CodeStorageABI.Pack("callFunc", algorithm, encodedInput)
	if err != nil {
		return validationProbe{}, nil, nil, fmt.Errorf("pack CodeStorage.callFunc: %w", err)
	}
	expectedRaw, err := outputArgs.Pack(expectedValues...)
	if err != nil {
		return validationProbe{}, nil, nil, fmt.Errorf("pack %s expected output: %w", algorithm, err)
	}
	return validationProbe{
		Algorithm:      algorithm,
		CallData:       callData,
		EncodedInput:   encodedInput,
		OutputArgs:     outputArgs,
		ExpectedRaw:    expectedRaw,
		ExpectedValues: append([]interface{}(nil), expectedValues...),
		ExpectedText:   formatABIValues(expectedValues),
	}, inputArgs, outputArgs, nil
}

func parseABIArguments(types []string) (abi.Arguments, error) {
	if len(types) == 0 {
		return nil, errors.New("empty ABI type list")
	}
	var args abi.Arguments
	for _, part := range types {
		typ := strings.TrimSpace(part)
		if typ == "" {
			return nil, fmt.Errorf("empty ABI type in %q", strings.Join(types, ","))
		}
		abiType, err := abi.NewType(typ, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Type: abiType})
	}
	return args, nil
}

func observeTargets(ctx context.Context, targets []network.NodeConfig, from common.Address, txHash common.Hash, probe validationProbe, cfg config, submitStarted time.Time) []nodeRoundResult {
	results := make([]nodeRoundResult, len(targets))
	var wg sync.WaitGroup
	for i, node := range targets {
		wg.Add(1)
		go func(i int, node network.NodeConfig) {
			defer wg.Done()
			results[i] = observeNode(ctx, node, from, txHash, probe, cfg, submitStarted)
		}(i, node)
	}
	wg.Wait()
	return results
}

func observeNode(ctx context.Context, node network.NodeConfig, from common.Address, txHash common.Hash, probe validationProbe, cfg config, submitStarted time.Time) nodeRoundResult {
	result := nodeRoundResult{
		ID:             node.ID,
		Role:           node.Role,
		RPCURL:         network.RPCURL(node),
		ExpectedResult: probe.ExpectedText,
	}
	client, err := rpc.DialContext(ctx, result.RPCURL)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer client.Close()
	if block, err := blockNumber(ctx, client); err == nil {
		result.PreflightBlock = block
	}
	var lastErr error
	for {
		if result.ReceiptObservedAt == "" {
			receipt, observedAt, err := waitReceiptOnce(ctx, client, txHash)
			if err != nil {
				if !isReceiptIndexing(err) {
					lastErr = err
				}
			} else if receipt != nil {
				result.receiptTime = observedAt
				result.ReceiptObservedAt = observedAt.Format(time.RFC3339Nano)
				if receipt.BlockNumber != nil {
					result.ReceiptBlockNumber = receipt.BlockNumber.Uint64()
				}
				if latest, err := blockNumber(ctx, client); err == nil {
					result.ReceiptLatestBlockNumber = latest
				}
				if receipt.Status != types.ReceiptStatusSuccessful {
					result.Error = fmt.Sprintf("receipt status=%d", receipt.Status)
					return result
				}
				result.SubmitToReceiptMillis = millis(observedAt.Sub(submitStarted))
			}
		}
		if result.ReceiptObservedAt != "" {
			validation, elapsed, err := callAlgorithm(ctx, client, from, probe, cfg.callGas)
			if err == nil {
				now := time.Now()
				result.ValidationStartedAt = validation.ValidationStartedAt
				result.ValidationFinishedAt = validation.ValidationFinishedAt
				result.ValidationLatencyMillis = millis(elapsed)
				result.ValidationOutputHex = validation.ValidationOutputHex
				result.DecodedResult = validation.DecodedResult
				result.CompletionBlockNumber = validation.CompletionBlockNumber
				result.completeTime = now
				result.CompletedAt = now.Format(time.RFC3339Nano)
				result.Completed = true
				result.ReceiptToCompleteMillis = millis(now.Sub(result.receiptTime))
				result.SubmitToCompleteMillis = millis(now.Sub(submitStarted))
				return result
			}
			lastErr = err
		}
		select {
		case <-ctx.Done():
			result.TimedOut = true
			if lastErr != nil {
				result.Error = lastErr.Error()
			} else {
				result.Error = ctx.Err().Error()
			}
			return result
		case <-time.After(cfg.pollInterval):
		}
	}
}

func waitReceipt(ctx context.Context, rpcURL string, txHash common.Hash, pollInterval time.Duration) (*types.Receipt, time.Time, error) {
	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer client.Close()
	for {
		receipt, observedAt, err := waitReceiptOnce(ctx, client, txHash)
		if err != nil && !isReceiptIndexing(err) {
			return nil, time.Time{}, err
		}
		if receipt != nil {
			return receipt, observedAt, nil
		}
		select {
		case <-ctx.Done():
			return nil, time.Time{}, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func waitReceiptOnce(ctx context.Context, client *rpc.Client, txHash common.Hash) (*types.Receipt, time.Time, error) {
	var receipt *types.Receipt
	if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
		return nil, time.Time{}, err
	}
	if receipt == nil {
		return nil, time.Time{}, nil
	}
	return receipt, time.Now(), nil
}

func isReceiptIndexing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "transaction indexing is in progress")
}

func callAlgorithm(ctx context.Context, client *rpc.Client, from common.Address, probe validationProbe, callGas uint64) (nodeRoundResult, time.Duration, error) {
	result := nodeRoundResult{ExpectedResult: probe.ExpectedText}
	block, err := blockNumber(ctx, client)
	if err != nil {
		return result, 0, err
	}
	result.CompletionBlockNumber = block
	started := time.Now()
	result.ValidationStartedAt = started.Format(time.RFC3339Nano)
	raw, err := ethCall(ctx, client, from, common.CodeStorageAddress, probe.CallData, callGas)
	finished := time.Now()
	result.ValidationFinishedAt = finished.Format(time.RFC3339Nano)
	if err != nil {
		return result, finished.Sub(started), wrapPluginHint(err)
	}
	result.ValidationOutputHex = hexutil.Encode(raw)
	decoded, err := probe.OutputArgs.Unpack(raw)
	if err != nil {
		return result, finished.Sub(started), fmt.Errorf("decode validation output: %w", err)
	}
	result.DecodedResult = formatABIValues(decoded)
	if !bytes.Equal(raw, probe.ExpectedRaw) {
		return result, finished.Sub(started), fmt.Errorf("validation output mismatch: got %s want %s", result.DecodedResult, probe.ExpectedText)
	}
	return result, finished.Sub(started), nil
}

func ethCall(ctx context.Context, client *rpc.Client, from common.Address, to common.Address, data []byte, callGas uint64) ([]byte, error) {
	gas := hexutil.Uint64(callGas)
	args := txArgs{
		To:   &to,
		Gas:  &gas,
		Data: data,
	}
	if from != (common.Address{}) {
		args.From = from
	}
	var out hexutil.Bytes
	if err := client.CallContext(ctx, &out, "eth_call", args, "latest"); err != nil {
		return nil, err
	}
	return out, nil
}

func sendTransaction(ctx context.Context, client *rpc.Client, args txArgs) (common.Hash, error) {
	var txHash common.Hash
	if err := client.CallContext(ctx, &txHash, "eth_sendTransaction", args); err != nil {
		return common.Hash{}, err
	}
	return txHash, nil
}

func blockNumber(ctx context.Context, client *rpc.Client) (uint64, error) {
	var number hexutil.Uint64
	if err := client.CallContext(ctx, &number, "eth_blockNumber"); err != nil {
		return 0, err
	}
	return uint64(number), nil
}

func readPeerCount(ctx context.Context, client *rpc.Client) (uint64, error) {
	var count hexutil.Uint64
	if err := client.CallContext(ctx, &count, "net_peerCount"); err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func annotateNodeStages(nodes []nodeRoundResult, txHashReturned, senderReceipt, eventObserved time.Time) {
	for i := range nodes {
		node := &nodes[i]
		if !node.receiptTime.IsZero() && !txHashReturned.IsZero() {
			node.TxHashToReceiptMillis = millis(node.receiptTime.Sub(txHashReturned))
		}
		if !node.receiptTime.IsZero() && !senderReceipt.IsZero() {
			node.SenderReceiptToReceiptMS = millis(node.receiptTime.Sub(senderReceipt))
		}
		if !node.completeTime.IsZero() && !txHashReturned.IsZero() {
			node.TxHashToCompleteMillis = millis(node.completeTime.Sub(txHashReturned))
		}
		if !node.completeTime.IsZero() && !senderReceipt.IsZero() {
			node.SenderReceiptToCompleteMS = millis(node.completeTime.Sub(senderReceipt))
		}
		if !node.completeTime.IsZero() && !eventObserved.IsZero() {
			node.EventToCompleteMillis = millis(node.completeTime.Sub(eventObserved))
		}
	}
}

func summarizeRound(nodes []nodeRoundResult, submitStarted, txHashReturned, senderReceipt, eventObserved time.Time) roundSummary {
	summary := roundSummary{
		TargetNodeCount: len(nodes),
		Completed:       true,
	}
	var latestReceipt time.Time
	var latestComplete time.Time
	var submitToReceiptTotal float64
	var txHashToReceiptTotal float64
	var senderReceiptToReceiptTotal float64
	var receiptToCompleteTotal float64
	var submitToCompleteTotal float64
	var txHashToCompleteTotal float64
	var senderReceiptToCompleteTotal float64
	var receiptCount int
	var txHashToReceiptCount int
	var senderReceiptToReceiptCount int
	var receiptToCompleteCount int
	var completeCount int
	var txHashToCompleteCount int
	var senderReceiptToCompleteCount int
	for _, node := range nodes {
		if node.Completed {
			summary.CompletedNodeCount++
			if latestComplete.IsZero() || node.completeTime.After(latestComplete) {
				latestComplete = node.completeTime
				summary.SlowestNodeID = node.ID
			}
			if !node.completeTime.IsZero() {
				submitToCompleteTotal += millis(node.completeTime.Sub(submitStarted))
				completeCount++
				if !txHashReturned.IsZero() {
					txHashToCompleteTotal += millis(node.completeTime.Sub(txHashReturned))
					txHashToCompleteCount++
				}
				if !senderReceipt.IsZero() {
					senderReceiptToCompleteTotal += millis(node.completeTime.Sub(senderReceipt))
					senderReceiptToCompleteCount++
				}
				if !node.receiptTime.IsZero() {
					receiptToCompleteTotal += millis(node.completeTime.Sub(node.receiptTime))
					receiptToCompleteCount++
				}
			}
		} else {
			summary.Completed = false
		}
		if !node.receiptTime.IsZero() {
			submitToReceiptTotal += millis(node.receiptTime.Sub(submitStarted))
			receiptCount++
			if !txHashReturned.IsZero() {
				txHashToReceiptTotal += millis(node.receiptTime.Sub(txHashReturned))
				txHashToReceiptCount++
			}
			if !senderReceipt.IsZero() {
				senderReceiptToReceiptTotal += millis(node.receiptTime.Sub(senderReceipt))
				senderReceiptToReceiptCount++
			}
			if latestReceipt.IsZero() || node.receiptTime.After(latestReceipt) {
				latestReceipt = node.receiptTime
			}
		}
	}
	if summary.CompletedNodeCount != summary.TargetNodeCount {
		summary.Completed = false
	}
	if !latestReceipt.IsZero() {
		summary.SubmitToAllReceiptMillis = millis(latestReceipt.Sub(submitStarted))
		if !txHashReturned.IsZero() {
			summary.TxHashToAllReceiptMillis = millis(latestReceipt.Sub(txHashReturned))
		}
		if !senderReceipt.IsZero() {
			summary.SenderReceiptToAllReceiptMillis = millis(latestReceipt.Sub(senderReceipt))
		}
	}
	if !latestComplete.IsZero() {
		summary.SubmitToAllCompleteMillis = millis(latestComplete.Sub(submitStarted))
		if !txHashReturned.IsZero() {
			summary.TxHashToAllCompleteMillis = millis(latestComplete.Sub(txHashReturned))
		}
		if !senderReceipt.IsZero() {
			summary.SenderReceiptToAllCompleteMillis = millis(latestComplete.Sub(senderReceipt))
		}
		if !eventObserved.IsZero() {
			summary.EventToAllCompleteMillis = millis(latestComplete.Sub(eventObserved))
		}
	}
	if summary.Completed && !latestReceipt.IsZero() && !latestComplete.IsZero() {
		summary.AllReceiptToAllCompleteMillis = millis(latestComplete.Sub(latestReceipt))
	}
	if !senderReceipt.IsZero() && !eventObserved.IsZero() {
		summary.ReceiptToEventMillis = millis(eventObserved.Sub(senderReceipt))
	}
	if !txHashReturned.IsZero() {
		summary.SubmitToTxHashMillis = millis(txHashReturned.Sub(submitStarted))
	}
	if receiptCount > 0 {
		summary.MeanNodeSubmitToReceiptMillis = submitToReceiptTotal / float64(receiptCount)
	}
	if txHashToReceiptCount > 0 {
		summary.MeanNodeTxHashToReceiptMillis = txHashToReceiptTotal / float64(txHashToReceiptCount)
	}
	if senderReceiptToReceiptCount > 0 {
		summary.MeanNodeSenderReceiptToReceiptMS = senderReceiptToReceiptTotal / float64(senderReceiptToReceiptCount)
	}
	if receiptToCompleteCount > 0 {
		summary.MeanNodeReceiptToCompleteMillis = receiptToCompleteTotal / float64(receiptToCompleteCount)
	}
	if completeCount > 0 {
		summary.MeanNodeSubmitToCompleteMillis = submitToCompleteTotal / float64(completeCount)
	}
	if txHashToCompleteCount > 0 {
		summary.MeanNodeTxHashToCompleteMillis = txHashToCompleteTotal / float64(txHashToCompleteCount)
	}
	if senderReceiptToCompleteCount > 0 {
		summary.MeanNodeSenderReceiptToCompleteMS = senderReceiptToCompleteTotal / float64(senderReceiptToCompleteCount)
	}
	if !senderReceipt.IsZero() {
		summary.SubmitToSenderReceiptMS = millis(senderReceipt.Sub(submitStarted))
		if !txHashReturned.IsZero() {
			summary.TxHashToSenderReceiptMillis = millis(senderReceipt.Sub(txHashReturned))
		}
	}
	return summary
}

func buildUpgradeTimeline(submitStarted, txHashReturned, senderReceipt, eventObserved time.Time, nodes []nodeRoundResult) upgradeTimeline {
	return upgradeTimeline{
		TransactionSubmittedAt: formatTimestamp(submitStarted),
		TxHashObservedAt:       formatTimestamp(txHashReturned),
		ReceiptObservedAt:      formatTimestamp(senderReceipt),
		EventObservedAt:        formatTimestamp(eventObserved),
		ActivationObservedAt:   formatTimestamp(latestNodeCompletion(nodes)),
	}
}

func latestNodeCompletion(nodes []nodeRoundResult) time.Time {
	var latest time.Time
	for _, node := range nodes {
		if node.Completed && !node.completeTime.IsZero() && (latest.IsZero() || node.completeTime.After(latest)) {
			latest = node.completeTime
		}
	}
	return latest
}

func receiptContainsCodeUploadedEvent(receipt *types.Receipt, upgradeName string) bool {
	if receipt == nil {
		return false
	}
	event, ok := cryptoupgrade.CodeStorageABI.Events["codeUploaded"]
	if !ok {
		return false
	}
	want := exportedIdentifier(upgradeName)
	for _, eventLog := range receipt.Logs {
		if eventLog.Address != common.CodeStorageAddress || len(eventLog.Topics) == 0 || eventLog.Topics[0] != event.ID {
			continue
		}
		var got string
		if err := cryptoupgrade.CodeStorageABI.UnpackIntoInterface(&got, "codeUploaded", eventLog.Data); err != nil {
			continue
		}
		if exportedIdentifier(got) == want {
			return true
		}
	}
	return false
}

func summarizeExperiment(rounds []roundResult, targetNodeCount int) experimentSummary {
	summary := experimentSummary{
		RoundCount:      len(rounds),
		TargetNodeCount: targetNodeCount,
		Completed:       len(rounds) > 0,
	}
	var total float64
	for i, round := range rounds {
		if round.Completed {
			summary.CompletedRoundCount++
			latency := round.Summary.SubmitToAllCompleteMillis
			total += latency
			if i == 0 || latency > summary.MaxAllCompleteMillis {
				summary.MaxAllCompleteMillis = latency
			}
			if i == 0 || latency < summary.MinAllCompleteMillis {
				summary.MinAllCompleteMillis = latency
			}
		} else {
			summary.Completed = false
		}
	}
	if summary.CompletedRoundCount > 0 {
		summary.MeanAllCompleteMillis = total / float64(summary.CompletedRoundCount)
	}
	if summary.CompletedRoundCount != summary.RoundCount {
		summary.Completed = false
	}
	return summary
}

func writeJSONResult(path string, res experimentResult) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create JSON output dir: %w", err)
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func writeCSVResults(cfg config, res experimentResult) error {
	if cfg.outputCSV != "" {
		if err := writeRoundCSV(cfg.outputCSV, res); err != nil {
			return err
		}
	}
	if cfg.outputNodeCSV != "" {
		if err := writeNodeCSV(cfg.outputNodeCSV, res); err != nil {
			return err
		}
	}
	return nil
}

func writeRoundCSV(path string, res experimentResult) error {
	file, err := createCSV(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"round", "algorithm", "upgrade_name", "source_path", "abi_input_types", "abi_output_types", "algo_gas",
		"input_summary", "expected_output", "sender_node", "tx_hash",
		"transaction_submitted_at", "tx_hash_observed_at", "receipt_observed_at", "event_observed_at", "activation_observed_at",
		"receipt_status", "gas_used", "target_nodes",
		"completed_nodes", "submit_to_tx_hash_ms", "tx_hash_to_sender_receipt_ms", "submit_to_sender_receipt_ms",
		"submit_to_all_receipt_ms", "tx_hash_to_all_receipt_ms", "sender_receipt_to_all_receipt_ms",
		"submit_to_all_complete_ms", "tx_hash_to_all_complete_ms", "sender_receipt_to_all_complete_ms",
		"receipt_to_event_ms", "event_to_all_complete_ms", "all_receipt_to_all_complete_ms",
		"mean_node_submit_to_receipt_ms", "mean_node_tx_hash_to_receipt_ms", "mean_node_sender_receipt_to_receipt_ms",
		"mean_node_receipt_to_complete_ms", "mean_node_submit_to_complete_ms", "mean_node_tx_hash_to_complete_ms",
		"mean_node_sender_receipt_to_complete_ms",
		"slowest_node", "completed", "error",
	}); err != nil {
		return err
	}
	for _, round := range res.Rounds {
		record := []string{
			fmt.Sprint(round.Index),
			round.Algorithm,
			round.UpgradeName,
			round.SourcePath,
			strings.Join(round.ABIInputTypes, ","),
			strings.Join(round.ABIOutputTypes, ","),
			fmt.Sprint(round.AlgoGas),
			formatJSONForCSV(round.InputSummary),
			round.ExpectedOutput,
			round.SenderNodeID,
			round.TransactionHash,
			round.PhaseTimeline.TransactionSubmittedAt,
			round.PhaseTimeline.TxHashObservedAt,
			round.PhaseTimeline.ReceiptObservedAt,
			round.PhaseTimeline.EventObservedAt,
			round.PhaseTimeline.ActivationObservedAt,
			fmt.Sprint(round.ReceiptStatus),
			fmt.Sprint(round.ReceiptGasUsed),
			fmt.Sprint(round.Summary.TargetNodeCount),
			fmt.Sprint(round.Summary.CompletedNodeCount),
			formatFloat(round.Summary.SubmitToTxHashMillis),
			formatFloat(round.Summary.TxHashToSenderReceiptMillis),
			formatFloat(round.Summary.SubmitToSenderReceiptMS),
			formatFloat(round.Summary.SubmitToAllReceiptMillis),
			formatFloat(round.Summary.TxHashToAllReceiptMillis),
			formatFloat(round.Summary.SenderReceiptToAllReceiptMillis),
			formatFloat(round.Summary.SubmitToAllCompleteMillis),
			formatFloat(round.Summary.TxHashToAllCompleteMillis),
			formatFloat(round.Summary.SenderReceiptToAllCompleteMillis),
			formatFloat(round.Summary.ReceiptToEventMillis),
			formatFloat(round.Summary.EventToAllCompleteMillis),
			formatFloat(round.Summary.AllReceiptToAllCompleteMillis),
			formatFloat(round.Summary.MeanNodeSubmitToReceiptMillis),
			formatFloat(round.Summary.MeanNodeTxHashToReceiptMillis),
			formatFloat(round.Summary.MeanNodeSenderReceiptToReceiptMS),
			formatFloat(round.Summary.MeanNodeReceiptToCompleteMillis),
			formatFloat(round.Summary.MeanNodeSubmitToCompleteMillis),
			formatFloat(round.Summary.MeanNodeTxHashToCompleteMillis),
			formatFloat(round.Summary.MeanNodeSenderReceiptToCompleteMS),
			round.Summary.SlowestNodeID,
			fmt.Sprint(round.Completed),
			round.Error,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.Error()
}

func writeNodeCSV(path string, res experimentResult) error {
	file, err := createCSV(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"round", "algorithm", "upgrade_name", "node_id", "role", "rpc_url", "receipt_block", "completion_block",
		"submit_to_receipt_ms", "tx_hash_to_receipt_ms", "sender_receipt_to_receipt_ms",
		"receipt_to_complete_ms", "submit_to_complete_ms", "tx_hash_to_complete_ms", "sender_receipt_to_complete_ms", "event_to_complete_ms",
		"validation_call_ms", "decoded_result", "expected_result", "completed", "timed_out", "error",
	}); err != nil {
		return err
	}
	for _, round := range res.Rounds {
		for _, node := range round.Nodes {
			record := []string{
				fmt.Sprint(round.Index),
				round.Algorithm,
				round.UpgradeName,
				node.ID,
				node.Role,
				node.RPCURL,
				fmt.Sprint(node.ReceiptBlockNumber),
				fmt.Sprint(node.CompletionBlockNumber),
				formatFloat(node.SubmitToReceiptMillis),
				formatFloat(node.TxHashToReceiptMillis),
				formatFloat(node.SenderReceiptToReceiptMS),
				formatFloat(node.ReceiptToCompleteMillis),
				formatFloat(node.SubmitToCompleteMillis),
				formatFloat(node.TxHashToCompleteMillis),
				formatFloat(node.SenderReceiptToCompleteMS),
				formatFloat(node.EventToCompleteMillis),
				formatFloat(node.ValidationLatencyMillis),
				node.DecodedResult,
				node.ExpectedResult,
				fmt.Sprint(node.Completed),
				fmt.Sprint(node.TimedOut),
				node.Error,
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
	}
	return writer.Error()
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func formatABIValues(values []interface{}) string {
	if len(values) == 1 {
		return formatABIValue(values[0])
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, formatABIValue(value))
	}
	return strings.Join(parts, ",")
}

func formatABIValue(value interface{}) string {
	switch v := value.(type) {
	case *big.Int:
		return v.String()
	case bool:
		return fmt.Sprintf("%t", v)
	case []byte:
		return hexutil.Encode(v)
	case [32]byte:
		return hexutil.Encode(v[:])
	}
	if b, ok := fixedByteArray(value); ok {
		return hexutil.Encode(b)
	}
	return fmt.Sprintf("%v", value)
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hexutil.Encode(sum[:])
}

func pbkdf2Sha256Expected(password, salt []byte, iterations, keyLength *big.Int) []byte {
	iter, ok := boundedInt(iterations, 1, 1000000)
	if !ok {
		return nil
	}
	keyLen, ok := boundedInt(keyLength, 0, 1<<20)
	if !ok {
		return nil
	}
	if keyLen == 0 {
		return []byte{}
	}

	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	blocks := (keyLen + hashLen - 1) / hashLen
	var blockIndex [4]byte
	u := make([]byte, 0, hashLen)
	t := make([]byte, hashLen)
	derived := make([]byte, 0, blocks*hashLen)

	for block := 1; block <= blocks; block++ {
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(blockIndex[:], uint32(block))
		prf.Write(blockIndex[:])
		u = prf.Sum(u[:0])
		copy(t, u)
		for round := 2; round <= iter; round++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for i := range t {
				t[i] ^= u[i]
			}
		}
		derived = append(derived, t...)
	}
	return derived[:keyLen]
}

func boundedInt(v *big.Int, min, max int) (int, bool) {
	if v == nil || !v.IsInt64() {
		return 0, false
	}
	n := v.Int64()
	if n < int64(min) || n > int64(max) {
		return 0, false
	}
	return int(n), true
}

func dh2048PeerPublic(exponent int64) []byte {
	p := rfc3526Prime()
	y := new(big.Int).Exp(big.NewInt(2), big.NewInt(exponent), p)
	return fixedBytes(y, rfc3526FieldBytes())
}

func dh2048Secret(privateKey, peerPublicKey []byte) []byte {
	p := rfc3526Prime()
	x := new(big.Int).SetBytes(privateKey)
	if x.Cmp(big.NewInt(1)) <= 0 || x.Cmp(new(big.Int).Sub(p, big.NewInt(1))) >= 0 {
		return nil
	}
	y := new(big.Int).SetBytes(peerPublicKey)
	if y.Cmp(big.NewInt(1)) <= 0 || y.Cmp(new(big.Int).Sub(p, big.NewInt(1))) >= 0 {
		return nil
	}
	secret := new(big.Int).Exp(y, x, p)
	return fixedBytes(secret, rfc3526FieldBytes())
}

func pedersenCommit(message, blinding []byte) []byte {
	p, q := rfc3526Subgroup()
	g := big.NewInt(4)
	h := pedersenH(p)
	m := new(big.Int).SetBytes(message)
	m.Mod(m, q)
	r := new(big.Int).SetBytes(blinding)
	r.Mod(r, q)

	gm := new(big.Int).Exp(g, m, p)
	hr := new(big.Int).Exp(h, r, p)
	commitment := new(big.Int).Mul(gm, hr)
	commitment.Mod(commitment, p)
	return fixedBytes(commitment, rfc3526FieldBytes())
}

func pedersenH(p *big.Int) *big.Int {
	for counter := byte(0); ; counter++ {
		sum := sha256.Sum256([]byte("cryptoupgrade-pedersen-h-" + string([]byte{counter})))
		h := new(big.Int).SetBytes(sum[:])
		h.Mod(h, p)
		h.Exp(h, big.NewInt(2), p)
		if h.Cmp(big.NewInt(1)) > 0 {
			return h
		}
	}
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

func createCSV(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create CSV output dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func printSummary(res experimentResult) {
	fmt.Printf("Upgrade latency experiment: %s\n", res.Experiment)
	fmt.Printf("  change:         %s\n", res.Change)
	fmt.Printf("  config:         %s\n", res.Config.ConfigPath)
	fmt.Printf("  sender:         %s (%s)\n", res.Config.SenderNodeID, res.Config.From)
	fmt.Printf("  targets:        %s\n", strings.Join(res.Config.TargetNodeIDs, ","))
	fmt.Printf("  rounds:         %d/%d complete\n", res.Summary.CompletedRoundCount, res.Summary.RoundCount)
	if res.Summary.CompletedRoundCount > 0 {
		fmt.Printf("  all-complete:   mean=%.3fms min=%.3fms max=%.3fms\n",
			res.Summary.MeanAllCompleteMillis, res.Summary.MinAllCompleteMillis, res.Summary.MaxAllCompleteMillis)
	}
	fmt.Printf("  json:           %s\n", res.Artifacts.ResultJSON)
	fmt.Printf("  round-csv:      %s\n", res.Artifacts.RoundCSV)
	fmt.Printf("  node-csv:       %s\n", res.Artifacts.NodeCSV)
	if res.Error != "" {
		fmt.Printf("  error:          %s\n", res.Error)
	}
}

func formatFloat(value float64) string {
	return fmt.Sprintf("%.6f", value)
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func formatJSONForCSV(value interface{}) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func millis(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / float64(time.Millisecond)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wrapPluginHint(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "can not compile code") || strings.Contains(msg, "cannot compile code") ||
		strings.Contains(msg, "plugin") || strings.Contains(msg, "buildmode=plugin") {
		return fmt.Errorf("%w (check each node log, Go plugin toolchain, CRYPTOUPGRADE_MODULE, and GETH_CRYPTOUPGRADE_PLUGIN_DIR)", err)
	}
	return err
}
