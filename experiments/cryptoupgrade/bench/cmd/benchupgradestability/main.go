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
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/cryptoupgrade/wasmtool"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	changeName     = "lab3-upgrade-stability"
	experimentName = "multi-node-upgrade-stability"
	defaultConfig  = "experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml"
)

type config struct {
	repoRoot      string
	configPath    string
	resultDir     string
	outputJSON    string
	outputCSV     string
	outputSummary string
	senderID      string
	nodeIDs       string
	from          string
	mode          string
	algorithm     string
	rounds        int
	activationLag uint64
	txGas         uint64
	callGas       uint64
	pollInterval  time.Duration
	timeout       time.Duration
	preflight     bool
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
	ConfigPath         string   `json:"configPath"`
	ChainID            uint64   `json:"chainId"`
	NetworkID          uint64   `json:"networkId"`
	ConsensusType      string   `json:"consensusType"`
	ConsensusPeriod    uint64   `json:"consensusPeriod"`
	NodeCount          int      `json:"nodeCount"`
	TargetNodeIDs      []string `json:"targetNodeIds"`
	SenderNodeID       string   `json:"senderNodeId"`
	SenderRPC          string   `json:"senderRpc"`
	From               string   `json:"from"`
	Mode               string   `json:"mode"`
	Algorithm          string   `json:"algorithm"`
	Rounds             int      `json:"rounds"`
	ActivationLag      uint64   `json:"activationLag"`
	PollIntervalMillis float64  `json:"pollIntervalMillis"`
	TimeoutMillis      float64  `json:"timeoutMillis"`
	Preflight          bool     `json:"preflight"`
}

type artifactResult struct {
	ResultDir     string `json:"resultDir"`
	ResultJSON    string `json:"resultJson"`
	ResultCSV     string `json:"resultCsv"`
	ResultSummary string `json:"resultSummary"`
}

type preflightResult struct {
	Nodes []preflightNode `json:"nodes"`
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
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
	Index           int               `json:"index"`
	Mode            string            `json:"mode"`
	Algorithm       string            `json:"algorithm"`
	VersionPlan     versionPlan       `json:"versionPlan"`
	SubmittedAt     string            `json:"submittedAt,omitempty"`
	TransactionHash string            `json:"transactionHash,omitempty"`
	Receipt         receiptResult     `json:"receipt"`
	Event           eventResult       `json:"event"`
	Samples         []sampleResult    `json:"samples"`
	Nodes           []nodeSummary     `json:"nodes"`
	Checks          consistencyChecks `json:"checks"`
	Completed       bool              `json:"completed"`
	Error           string            `json:"error,omitempty"`
	metadataHash    common.Hash       `json:"-"`
	oldExpectedRaw  []byte            `json:"-"`
	newExpectedRaw  []byte            `json:"-"`
	oldProbe        validationProbe   `json:"-"`
	newProbe        validationProbe   `json:"-"`
	receiptObj      *types.Receipt    `json:"-"`
}

type versionPlan struct {
	OldVersion       uint64 `json:"oldVersion"`
	NewVersion       uint64 `json:"newVersion"`
	ActivationBlock  uint64 `json:"activationBlock"`
	WasmHash         string `json:"wasmHash"`
	OldExpectedValue string `json:"oldExpectedValue"`
	NewExpectedValue string `json:"newExpectedValue"`
	MetadataHash     string `json:"metadataHash"`
}

type receiptResult struct {
	Status      uint64 `json:"status,omitempty"`
	GasUsed     uint64 `json:"gasUsed,omitempty"`
	BlockNumber uint64 `json:"blockNumber,omitempty"`
	BlockHash   string `json:"blockHash,omitempty"`
	LogCount    int    `json:"logCount,omitempty"`
}

type eventResult struct {
	Found           bool   `json:"found"`
	Name            string `json:"name,omitempty"`
	Version         uint64 `json:"version,omitempty"`
	ActivationBlock uint64 `json:"activationBlock,omitempty"`
	Consistent      bool   `json:"consistent"`
	Error           string `json:"error,omitempty"`
}

type sampleResult struct {
	Stage           string             `json:"stage"`
	BlockNumber     uint64             `json:"blockNumber"`
	BlockHash       string             `json:"blockHash"`
	ExpectedVersion uint64             `json:"expectedVersion"`
	ExpectedOutput  string             `json:"expectedOutput"`
	Nodes           []nodeSampleResult `json:"nodes"`
	Checks          consistencyChecks  `json:"checks"`
}

type nodeSampleResult struct {
	NodeID           string `json:"nodeId"`
	Role             string `json:"role"`
	RPCURL           string `json:"rpcURL"`
	HeadNumber       uint64 `json:"headNumber,omitempty"`
	HeadHash         string `json:"headHash,omitempty"`
	BlockHash        string `json:"blockHash,omitempty"`
	ReceiptVisible   bool   `json:"receiptVisible"`
	ReceiptBlock     uint64 `json:"receiptBlock,omitempty"`
	ReceiptBlockHash string `json:"receiptBlockHash,omitempty"`
	Version          uint64 `json:"version,omitempty"`
	ActivationBlock  uint64 `json:"activationBlock,omitempty"`
	MetadataHash     string `json:"metadataHash,omitempty"`
	Output           string `json:"output,omitempty"`
	OutputHex        string `json:"outputHex,omitempty"`
	OK               bool   `json:"ok"`
	Error            string `json:"error,omitempty"`
}

type nodeSummary struct {
	NodeID    string `json:"nodeId"`
	Role      string `json:"role"`
	RPCURL    string `json:"rpcURL"`
	Completed bool   `json:"completed"`
	Error     string `json:"error,omitempty"`
}

type consistencyChecks struct {
	ChainViewConsistent bool     `json:"chainViewConsistent"`
	ReceiptConsistent   bool     `json:"receiptConsistent"`
	EventConsistent     bool     `json:"eventConsistent"`
	VersionConsistent   bool     `json:"versionConsistent"`
	OutputConsistent    bool     `json:"outputConsistent"`
	Passed              bool     `json:"passed"`
	Failures            []string `json:"failures,omitempty"`
}

type experimentSummary struct {
	RoundCount          int      `json:"roundCount"`
	CompletedRoundCount int      `json:"completedRoundCount"`
	Completed           bool     `json:"completed"`
	FailedRounds        []int    `json:"failedRounds,omitempty"`
	FailureReasons      []string `json:"failureReasons,omitempty"`
}

type validationProbe struct {
	CallData       []byte
	OutputArgs     abi.Arguments
	ExpectedRaw    []byte
	ExpectedValues []interface{}
	ExpectedText   string
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "benchupgradestability: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "benchupgradestability: failed to write JSON result: %v\n", writeErr)
		}
	}
	if csvErr := writeCSVResult(cfg.outputCSV, res); csvErr != nil {
		if err == nil {
			err = csvErr
			res.Error = csvErr.Error()
			_ = writeJSONResult(cfg.outputJSON, res)
		} else {
			fmt.Fprintf(os.Stderr, "benchupgradestability: failed to write CSV result: %v\n", csvErr)
		}
	}
	if summaryErr := writeSummaryResult(cfg.outputSummary, res); summaryErr != nil {
		if err == nil {
			err = summaryErr
			res.Error = summaryErr.Error()
			_ = writeJSONResult(cfg.outputJSON, res)
		} else {
			fmt.Fprintf(os.Stderr, "benchupgradestability: failed to write summary result: %v\n", summaryErr)
		}
	}
	fmt.Print(formatSummary(res))
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchupgradestability: %v\n", err)
		os.Exit(1)
	}
}

func parseConfig(args []string) (config, error) {
	cfg := config{}
	fs := flag.NewFlagSet("benchupgradestability", flag.ContinueOnError)
	fs.StringVar(&cfg.repoRoot, "repo-root", ".", "repository root")
	fs.StringVar(&cfg.configPath, "config", defaultConfig, "multi-node network YAML config path")
	fs.StringVar(&cfg.resultDir, "out", "", "directory for raw JSON/CSV results")
	fs.StringVar(&cfg.outputJSON, "output-json", "", "write raw experiment JSON to this path")
	fs.StringVar(&cfg.outputCSV, "output-csv", "", "write node/block sample CSV to this path")
	fs.StringVar(&cfg.outputSummary, "output-summary", "", "write text summary to this path")
	fs.StringVar(&cfg.senderID, "sender", "", "node ID used to submit upgrade transactions; defaults to first signer")
	fs.StringVar(&cfg.nodeIDs, "nodes", "", "comma-separated node IDs to observe; defaults to all configured nodes")
	fs.StringVar(&cfg.from, "from", "", "transaction sender address; defaults to sender node account")
	fs.StringVar(&cfg.mode, "mode", "both", "upgrade strategy to verify: scheduled, immediate, or both")
	fs.StringVar(&cfg.algorithm, "algorithm", "Add", "algorithm name used by the stability fixture")
	fs.IntVar(&cfg.rounds, "rounds", 1, "number of repeated stability rounds")
	fs.Uint64Var(&cfg.activationLag, "activation-lag", 2, "scheduled mode activation block offset from submission head")
	fs.Uint64Var(&cfg.txGas, "tx-gas", 5000000, "upgrade transaction gas limit")
	fs.Uint64Var(&cfg.callGas, "call-gas", 5000000, "eth_call gas limit for validation")
	fs.DurationVar(&cfg.pollInterval, "poll-interval", 200*time.Millisecond, "polling interval for block and receipt visibility")
	fs.DurationVar(&cfg.timeout, "timeout", 2*time.Minute, "per-round timeout")
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
	cfg.mode = strings.ToLower(strings.TrimSpace(cfg.mode))
	if cfg.mode != "scheduled" && cfg.mode != "immediate" && cfg.mode != "both" {
		return cfg, errors.New("-mode must be scheduled, immediate, or both")
	}
	cfg.algorithm = exportedIdentifier(cfg.algorithm)
	if cfg.algorithm == "" {
		return cfg, errors.New("-algorithm must contain at least one letter")
	}
	if cfg.rounds <= 0 {
		return cfg, errors.New("-rounds must be positive")
	}
	if cfg.activationLag == 0 {
		return cfg, errors.New("-activation-lag must be positive")
	}
	if cfg.txGas == 0 || cfg.callGas == 0 {
		return cfg, errors.New("-tx-gas and -call-gas must be non-zero")
	}
	if cfg.pollInterval <= 0 || cfg.timeout <= 0 {
		return cfg, errors.New("-poll-interval and -timeout must be positive")
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return cfg, fmt.Errorf("invalid -from address %q", cfg.from)
	}
	timestamp := time.Now().Format("20060102-150405")
	if cfg.resultDir == "" {
		cfg.resultDir = filepath.Join(cfg.repoRoot, "experiments", "cryptoupgrade", "results", "upgrade-stability", "run-"+timestamp)
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
		cfg.outputCSV = filepath.Join(cfg.resultDir, "samples.csv")
	} else if !filepath.IsAbs(cfg.outputCSV) {
		cfg.outputCSV = filepath.Join(cfg.repoRoot, cfg.outputCSV)
	}
	if cfg.outputSummary == "" {
		cfg.outputSummary = filepath.Join(cfg.resultDir, "summary.txt")
	} else if !filepath.IsAbs(cfg.outputSummary) {
		cfg.outputSummary = filepath.Join(cfg.repoRoot, cfg.outputSummary)
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
			ResultDir:     cfg.resultDir,
			ResultJSON:    cfg.outputJSON,
			ResultCSV:     cfg.outputCSV,
			ResultSummary: cfg.outputSummary,
		},
		Limitations: []string{
			"稳定性实验通过 JSON-RPC 从控制端采样，短暂网络传播差异会体现在采样结果中。",
			"版本选择以指定 block number 的 eth_call 结果为准，不使用节点日志时间作为一致性判据。",
			"本命令默认使用 Add fixture 的两个版本区分旧输出和新输出，不统计执行效率或 Gas 对比作为主结论。",
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
		preflight, err := runPreflight(ctx, selected, cfg.timeout)
		res.Preflight = preflight
		if err != nil {
			return err
		}
	}
	modes := []string{cfg.mode}
	if cfg.mode == "both" {
		modes = []string{"scheduled", "immediate"}
	}
	index := 1
	for repeat := 1; repeat <= cfg.rounds; repeat++ {
		for _, mode := range modes {
			round, err := runRound(ctx, cfg, selected, index, repeat, mode)
			res.Rounds = append(res.Rounds, round)
			if err != nil {
				res.Summary = summarizeExperiment(res.Rounds)
				return err
			}
			index++
		}
	}
	res.Summary = summarizeExperiment(res.Rounds)
	res.Completed = res.Summary.Completed
	return nil
}

func fillConfigResult(res *experimentResult, cfg config, selected selectedNetwork) {
	targetIDs := make([]string, 0, len(selected.targets))
	for _, node := range selected.targets {
		targetIDs = append(targetIDs, node.ID)
	}
	res.Config = experimentConfig{
		ConfigPath:         selected.cfg.ConfigPath(),
		ChainID:            selected.cfg.Network.ChainID,
		NetworkID:          selected.cfg.Network.NetworkID,
		ConsensusType:      selected.cfg.Consensus.Type,
		ConsensusPeriod:    selected.cfg.Consensus.Period,
		NodeCount:          len(selected.cfg.Nodes),
		TargetNodeIDs:      targetIDs,
		SenderNodeID:       selected.sender.ID,
		SenderRPC:          network.RPCURL(selected.sender),
		From:               selected.from.Hex(),
		Mode:               cfg.mode,
		Algorithm:          cfg.algorithm,
		Rounds:             cfg.rounds,
		ActivationLag:      cfg.activationLag,
		PollIntervalMillis: millis(cfg.pollInterval),
		TimeoutMillis:      millis(cfg.timeout),
		Preflight:          cfg.preflight,
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
		return append([]network.NodeConfig(nil), nodes...), nil
	}
	byID := make(map[string]network.NodeConfig, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	var out []network.NodeConfig
	seen := make(map[string]struct{})
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
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	result := &preflightResult{OK: true}
	wait := time.Duration(selected.cfg.Consensus.Period+1) * time.Second
	nodes := make([]preflightNode, len(selected.targets))
	var wg sync.WaitGroup
	for i, node := range selected.targets {
		wg.Add(1)
		go func(i int, node network.NodeConfig) {
			defer wg.Done()
			nodes[i] = preflightOneNode(ctx, selected.cfg, node, wait, len(selected.cfg.Nodes)-1)
		}(i, node)
	}
	wg.Wait()
	result.Nodes = nodes
	for _, node := range nodes {
		if node.Error != "" {
			result.OK = false
		}
	}
	if !result.OK {
		result.Error = "preflight failed"
		return result, errors.New("preflight failed; inspect result.json for node-level errors")
	}
	return result, nil
}

func preflightOneNode(ctx context.Context, cfg *network.Config, node network.NodeConfig, wait time.Duration, expectedPeers int) preflightNode {
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
	peers, err := readPeerCount(ctx, client)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.PeerCount = peers
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
	peers, err = readPeerCount(ctx, client)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.PeerCount = peers
	if out.EndBlock <= out.StartBlock {
		out.Error = fmt.Sprintf("block height did not increase: start %d end %d", out.StartBlock, out.EndBlock)
		return out
	}
	if expectedPeers > 0 && int(out.PeerCount) < expectedPeers {
		out.Error = fmt.Sprintf("peer count too low: want >= %d got %d", expectedPeers, out.PeerCount)
	}
	return out
}

func runRound(parent context.Context, cfg config, selected selectedNetwork, index, repeat int, mode string) (roundResult, error) {
	ctx, cancel := context.WithTimeout(parent, cfg.timeout)
	defer cancel()
	rr := roundResult{
		Index:     index,
		Mode:      mode,
		Algorithm: cfg.algorithm,
	}
	senderClient, err := rpc.DialContext(ctx, network.RPCURL(selected.sender))
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	defer senderClient.Close()
	submitHead, err := blockNumber(ctx, senderClient)
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	oldVersion, newVersion := roundVersions(submitHead, index)
	oldProbe, err := buildAddProbe(cfg.algorithm, 0)
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	newProbe, err := buildAddProbe(cfg.algorithm, 5)
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	rr.oldProbe = oldProbe
	rr.newProbe = newProbe
	rr.oldExpectedRaw = oldProbe.ExpectedRaw
	rr.newExpectedRaw = newProbe.ExpectedRaw

	if err := submitVersion(ctx, senderClient, selected.from, cfg, cfg.algorithm, oldVersion, submitHead, addSource(0)); err != nil {
		rr.Error = fmt.Sprintf("submit baseline version: %v", err)
		return rr, errors.New(rr.Error)
	}
	if err := waitForAllNodesAtLeast(ctx, selected.targets, submitHead, cfg.pollInterval); err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	if err := waitVersionCallable(ctx, selected.targets, selected.from, cfg, oldProbe, oldVersion, submitHead); err != nil {
		rr.Error = fmt.Sprintf("baseline version not callable: %v", err)
		return rr, errors.New(rr.Error)
	}

	planHead, err := blockNumber(ctx, senderClient)
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	activationBlock := planHead + cfg.activationLag
	if mode == "immediate" {
		activationBlock = planHead
	}
	source := addSource(5)
	wasm, encoded, err := wasmtool.BuildEncodedSource(ctx, []byte(source), wasmtool.Spec{
		Function:    "Add",
		InputTypes:  []string{"int256", "int256"},
		OutputTypes: []string{"int256"},
	})
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	newMetadataHash := metadataHash(encoded, 3000, "uint256,uint256", "uint256", newVersion, activationBlock)
	rr.metadataHash = newMetadataHash
	rr.VersionPlan = versionPlan{
		OldVersion:       oldVersion,
		NewVersion:       newVersion,
		ActivationBlock:  activationBlock,
		WasmHash:         wasmtool.WasmHash(wasm),
		OldExpectedValue: oldProbe.ExpectedText,
		NewExpectedValue: newProbe.ExpectedText,
		MetadataHash:     newMetadataHash.Hex(),
	}
	var data []byte
	if mode == "immediate" {
		data, err = cryptoupgrade.CodeStorageABI.Pack("uploadCodeImmediate", cfg.algorithm, newVersion, encoded, uint64(3000), "uint256,uint256", "uint256")
	} else {
		data, err = cryptoupgrade.CodeStorageABI.Pack("uploadCodeVersion", cfg.algorithm, newVersion, encoded, uint64(3000), "uint256,uint256", "uint256", activationBlock)
	}
	if err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	to := common.CodeStorageAddress
	gas := hexutil.Uint64(cfg.txGas)
	rr.SubmittedAt = time.Now().Format(time.RFC3339Nano)
	txHash, err := sendTransaction(ctx, senderClient, txArgs{From: selected.from, To: &to, Gas: &gas, Data: data})
	if err != nil {
		rr.Error = fmt.Sprintf("send upgrade transaction: %v", err)
		return rr, errors.New(rr.Error)
	}
	rr.TransactionHash = txHash.Hex()
	receipt, err := waitReceipt(ctx, senderClient, txHash, cfg.pollInterval)
	if err != nil {
		rr.Error = fmt.Sprintf("wait sender receipt: %v", err)
		return rr, errors.New(rr.Error)
	}
	rr.receiptObj = receipt
	rr.Receipt = receiptSummary(receipt)
	rr.Event = parseVersionEvent(receipt, cfg.algorithm, newVersion, activationBlock, mode == "immediate")
	if !rr.Event.Consistent {
		rr.Error = rr.Event.Error
		return rr, errors.New(rr.Error)
	}
	if mode == "immediate" {
		activationBlock = rr.Event.ActivationBlock
		rr.VersionPlan.ActivationBlock = activationBlock
		rr.VersionPlan.MetadataHash = metadataHash(encoded, 3000, "uint256,uint256", "uint256", newVersion, activationBlock).Hex()
	} else if rr.Receipt.BlockNumber > activationBlock {
		rr.Error = fmt.Sprintf("scheduled activationBlock %d is before receipt block %d; increase -activation-lag", activationBlock, rr.Receipt.BlockNumber)
		return rr, errors.New(rr.Error)
	}
	if err := waitForAllNodesAtLeast(ctx, selected.targets, maxUint64(activationBlock, rr.Receipt.BlockNumber), cfg.pollInterval); err != nil {
		rr.Error = err.Error()
		return rr, err
	}
	if err := waitVersionCallable(ctx, selected.targets, selected.from, cfg, newProbe, newVersion, activationBlock); err != nil {
		rr.Error = fmt.Sprintf("new version not callable: %v", err)
		return rr, errors.New(rr.Error)
	}
	beforeBlock := activationBlock - 1
	if beforeBlock < submitHead {
		beforeBlock = submitHead
	}
	sampleBlocks := []samplePoint{
		{Stage: "before", BlockNumber: beforeBlock, Probe: oldProbe, ExpectedVersion: oldVersion},
		{Stage: "after", BlockNumber: activationBlock, Probe: newProbe, ExpectedVersion: newVersion},
	}
	if mode == "immediate" || activationBlock == 0 {
		sampleBlocks = []samplePoint{{Stage: "after", BlockNumber: activationBlock, Probe: newProbe, ExpectedVersion: newVersion}}
	}
	for _, point := range sampleBlocks {
		sample := collectSample(ctx, selected.targets, selected.from, cfg, rr, point, txHash)
		rr.Samples = append(rr.Samples, sample)
	}
	rr.Checks = summarizeRoundChecks(rr)
	rr.Nodes = summarizeNodes(selected.targets, rr.Samples)
	rr.Completed = rr.Checks.Passed
	if !rr.Completed {
		rr.Error = strings.Join(rr.Checks.Failures, "; ")
		return rr, errors.New(rr.Error)
	}
	return rr, nil
}

type samplePoint struct {
	Stage           string
	BlockNumber     uint64
	Probe           validationProbe
	ExpectedVersion uint64
}

func submitVersion(ctx context.Context, client *rpc.Client, from common.Address, cfg config, name string, version, activationBlock uint64, source string) error {
	encoded, err := wasmtool.EncodeSource(ctx, []byte(source), wasmtool.Spec{
		Function:    "Add",
		InputTypes:  []string{"int256", "int256"},
		OutputTypes: []string{"int256"},
	})
	if err != nil {
		return err
	}
	data, err := cryptoupgrade.CodeStorageABI.Pack("uploadCodeVersion", name, version, encoded, uint64(3000), "uint256,uint256", "uint256", activationBlock)
	if err != nil {
		return err
	}
	to := common.CodeStorageAddress
	gas := hexutil.Uint64(cfg.txGas)
	hash, err := sendTransaction(ctx, client, txArgs{From: from, To: &to, Gas: &gas, Data: data})
	if err != nil {
		return err
	}
	receipt, err := waitReceipt(ctx, client, hash, cfg.pollInterval)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("baseline receipt status=%d", receipt.Status)
	}
	return nil
}

func roundVersions(blockNumber uint64, index int) (uint64, uint64) {
	// 同一条私链可能连续运行多次实验，版本号必须高于历史轮次，避免旧高版本覆盖新基线。
	base := uint64(time.Now().UnixMilli())*1000 + blockNumber*10 + uint64(index*2)
	return base + 1, base + 2
}

func waitVersionCallable(ctx context.Context, targets []network.NodeConfig, from common.Address, cfg config, probe validationProbe, version, blockNumber uint64) error {
	for _, node := range targets {
		client, err := rpc.DialContext(ctx, network.RPCURL(node))
		if err != nil {
			return err
		}
		for {
			active, _, versionErr := getActiveVersion(ctx, client, cfg.algorithm, blockNumber, from, cfg.callGas)
			result, err := callAtBlock(ctx, client, from, cfg.callGas, probe, blockNumber)
			if versionErr == nil && err == nil && active == version && result.OK {
				client.Close()
				break
			}
			select {
			case <-ctx.Done():
				client.Close()
				if versionErr != nil {
					return fmt.Errorf("node %s active version: %w", node.ID, versionErr)
				}
				if err != nil {
					return fmt.Errorf("node %s: %w", node.ID, err)
				}
				return fmt.Errorf("node %s version %d not callable at block %d", node.ID, version, blockNumber)
			case <-time.After(cfg.pollInterval):
			}
		}
	}
	return nil
}

func collectSample(ctx context.Context, targets []network.NodeConfig, from common.Address, cfg config, rr roundResult, point samplePoint, txHash common.Hash) sampleResult {
	sample := sampleResult{
		Stage:           point.Stage,
		BlockNumber:     point.BlockNumber,
		ExpectedVersion: point.ExpectedVersion,
		ExpectedOutput:  point.Probe.ExpectedText,
	}
	nodes := make([]nodeSampleResult, len(targets))
	var wg sync.WaitGroup
	for i, node := range targets {
		wg.Add(1)
		go func(i int, node network.NodeConfig) {
			defer wg.Done()
			nodes[i] = collectNodeSample(ctx, node, from, cfg, rr, point, txHash)
		}(i, node)
	}
	wg.Wait()
	sample.Nodes = nodes
	if len(nodes) > 0 {
		sample.BlockHash = nodes[0].BlockHash
	}
	sample.Checks = summarizeSampleChecks(sample)
	return sample
}

func collectNodeSample(ctx context.Context, node network.NodeConfig, from common.Address, cfg config, rr roundResult, point samplePoint, txHash common.Hash) nodeSampleResult {
	out := nodeSampleResult{NodeID: node.ID, Role: node.Role, RPCURL: network.RPCURL(node)}
	client, err := rpc.DialContext(ctx, out.RPCURL)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer client.Close()
	head, err := blockNumber(ctx, client)
	if err == nil {
		out.HeadNumber = head
		if hash, hashErr := blockHash(ctx, client, head); hashErr == nil {
			out.HeadHash = hash.Hex()
		}
	}
	hash, err := blockHash(ctx, client, point.BlockNumber)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.BlockHash = hash.Hex()
	receipt, err := transactionReceipt(ctx, client, txHash)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if receipt != nil {
		out.ReceiptVisible = true
		if receipt.BlockNumber != nil {
			out.ReceiptBlock = receipt.BlockNumber.Uint64()
		}
		out.ReceiptBlockHash = receipt.BlockHash.Hex()
	}
	version, activation, err := getActiveVersion(ctx, client, cfg.algorithm, point.BlockNumber, from, cfg.callGas)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Version = version
	out.ActivationBlock = activation
	info, err := getVersionInfo(ctx, client, cfg.algorithm, version, point.BlockNumber, from, cfg.callGas)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.MetadataHash = metadataHash(info.Code, info.Gas, info.IType, info.OType, info.Version, info.ActivationBlock).Hex()
	call, err := callAtBlock(ctx, client, from, cfg.callGas, point.Probe, point.BlockNumber)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Output = call.Output
	out.OutputHex = call.OutputHex
	out.OK = out.ReceiptVisible &&
		out.ReceiptBlockHash == rr.Receipt.BlockHash &&
		out.Version == point.ExpectedVersion
	if out.OK && point.ExpectedVersion == rr.VersionPlan.NewVersion {
		out.OK = out.ActivationBlock == rr.VersionPlan.ActivationBlock && out.MetadataHash == rr.VersionPlan.MetadataHash
	}
	if out.OK && call.Output != point.Probe.ExpectedText {
		out.OK = false
		out.Error = fmt.Sprintf("output mismatch: got %s want %s", call.Output, point.Probe.ExpectedText)
	}
	if !out.OK && out.Error == "" {
		out.Error = "node sample did not match expected receipt/version/output"
	}
	return out
}

type callResult struct {
	Output    string
	OutputHex string
	OK        bool
}

func callAtBlock(ctx context.Context, client *rpc.Client, from common.Address, callGas uint64, probe validationProbe, blockNumber uint64) (callResult, error) {
	gas := hexutil.Uint64(callGas)
	args := txArgs{To: &common.CodeStorageAddress, Gas: &gas, Data: probe.CallData}
	if from != (common.Address{}) {
		args.From = from
	}
	var raw hexutil.Bytes
	if err := client.CallContext(ctx, &raw, "eth_call", args, blockTag(blockNumber)); err != nil {
		return callResult{}, err
	}
	decoded, err := probe.OutputArgs.Unpack(raw)
	if err != nil {
		return callResult{}, err
	}
	out := callResult{
		Output:    formatABIValues(decoded),
		OutputHex: hexutil.Encode(raw),
		OK:        bytes.Equal(raw, probe.ExpectedRaw),
	}
	return out, nil
}

func getActiveVersion(ctx context.Context, client *rpc.Client, name string, blockNumber uint64, from common.Address, callGas uint64) (uint64, uint64, error) {
	data, err := cryptoupgrade.CodeStorageABI.Pack("getActiveVersion", name)
	if err != nil {
		return 0, 0, err
	}
	raw, err := rawEthCall(ctx, client, from, data, blockNumber, callGas)
	if err != nil {
		return 0, 0, err
	}
	values, err := cryptoupgrade.CodeStorageABI.Unpack("getActiveVersion", raw)
	if err != nil {
		return 0, 0, err
	}
	return values[0].(uint64), values[1].(uint64), nil
}

func getVersionInfo(ctx context.Context, client *rpc.Client, name string, version, blockNumber uint64, from common.Address, callGas uint64) (struct {
	Code            string
	Gas             uint64
	IType           string
	OType           string
	Version         uint64
	ActivationBlock uint64
}, error) {
	var out struct {
		Code            string
		Gas             uint64
		IType           string
		OType           string
		Version         uint64
		ActivationBlock uint64
	}
	data, err := cryptoupgrade.CodeStorageABI.Pack("getVersionInfo", name, version)
	if err != nil {
		return out, err
	}
	raw, err := rawEthCall(ctx, client, from, data, blockNumber, callGas)
	if err != nil {
		return out, err
	}
	values, err := cryptoupgrade.CodeStorageABI.Unpack("getVersionInfo", raw)
	if err != nil {
		return out, err
	}
	out.Code = values[0].(string)
	out.Gas = values[1].(uint64)
	out.IType = values[2].(string)
	out.OType = values[3].(string)
	out.Version = values[4].(uint64)
	out.ActivationBlock = values[5].(uint64)
	return out, nil
}

func rawEthCall(ctx context.Context, client *rpc.Client, from common.Address, data []byte, blockNumber uint64, callGas uint64) ([]byte, error) {
	gas := hexutil.Uint64(callGas)
	args := txArgs{To: &common.CodeStorageAddress, Gas: &gas, Data: data}
	if from != (common.Address{}) {
		args.From = from
	}
	var out hexutil.Bytes
	if err := client.CallContext(ctx, &out, "eth_call", args, blockTag(blockNumber)); err != nil {
		return nil, err
	}
	return out, nil
}

func buildAddProbe(name string, delta int64) (validationProbe, error) {
	inputArgs, err := parseABIArguments([]string{"uint256", "uint256"})
	if err != nil {
		return validationProbe{}, err
	}
	outputArgs, err := parseABIArguments([]string{"uint256"})
	if err != nil {
		return validationProbe{}, err
	}
	a := big.NewInt(100)
	b := big.NewInt(100)
	expected := new(big.Int).Add(a, b)
	expected.Add(expected, big.NewInt(delta))
	encodedInput, err := inputArgs.Pack(a, b)
	if err != nil {
		return validationProbe{}, err
	}
	callData, err := cryptoupgrade.CodeStorageABI.Pack("callFunc", name, encodedInput)
	if err != nil {
		return validationProbe{}, err
	}
	expectedRaw, err := outputArgs.Pack(expected)
	if err != nil {
		return validationProbe{}, err
	}
	return validationProbe{
		CallData:       callData,
		OutputArgs:     outputArgs,
		ExpectedRaw:    expectedRaw,
		ExpectedValues: []interface{}{expected},
		ExpectedText:   formatABIValues([]interface{}{expected}),
	}, nil
}

func addSource(delta int64) string {
	if delta == 0 {
		return `package main

import "math/big"

func Add(a *big.Int, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}
`
	}
	return fmt.Sprintf(`package main

import "math/big"

func Add(a *big.Int, b *big.Int) *big.Int {
	out := new(big.Int).Add(a, b)
	return out.Add(out, big.NewInt(%d))
}
`, delta)
}

func parseABIArguments(types []string) (abi.Arguments, error) {
	args := make(abi.Arguments, 0, len(types))
	for _, typ := range types {
		abiType, err := abi.NewType(typ, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Type: abiType})
	}
	return args, nil
}

func metadataHash(code string, gas uint64, itype, otype string, version, activationBlock uint64) common.Hash {
	payload := fmt.Sprintf("%s|%d|%s|%s|%d|%d", code, gas, itype, otype, version, activationBlock)
	return sha256Hash([]byte(payload))
}

func sha256Hash(data []byte) common.Hash {
	sum := sha256.Sum256(data)
	return common.BytesToHash(sum[:])
}

func sendTransaction(ctx context.Context, client *rpc.Client, args txArgs) (common.Hash, error) {
	var txHash common.Hash
	if err := client.CallContext(ctx, &txHash, "eth_sendTransaction", args); err != nil {
		return common.Hash{}, err
	}
	return txHash, nil
}

func waitReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash, pollInterval time.Duration) (*types.Receipt, error) {
	for {
		receipt, err := transactionReceipt(ctx, client, txHash)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "transaction indexing is in progress") {
			return nil, err
		}
		if receipt != nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func transactionReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
		return nil, err
	}
	return receipt, nil
}

func receiptSummary(receipt *types.Receipt) receiptResult {
	out := receiptResult{}
	if receipt == nil {
		return out
	}
	out.Status = receipt.Status
	out.GasUsed = receipt.GasUsed
	out.BlockHash = receipt.BlockHash.Hex()
	out.LogCount = len(receipt.Logs)
	if receipt.BlockNumber != nil {
		out.BlockNumber = receipt.BlockNumber.Uint64()
	}
	return out
}

func parseVersionEvent(receipt *types.Receipt, name string, version, activationBlock uint64, allowDynamicActivation bool) eventResult {
	out := eventResult{}
	if receipt == nil {
		out.Error = "missing receipt"
		return out
	}
	event := cryptoupgrade.CodeStorageABI.Events["codeVersionUploaded"]
	for _, eventLog := range receipt.Logs {
		if eventLog.Address != common.CodeStorageAddress || len(eventLog.Topics) == 0 || eventLog.Topics[0] != event.ID {
			continue
		}
		values, err := cryptoupgrade.CodeStorageABI.Unpack("codeVersionUploaded", eventLog.Data)
		if err != nil {
			out.Error = err.Error()
			return out
		}
		out.Found = true
		out.Name = values[0].(string)
		out.Version = values[1].(uint64)
		out.ActivationBlock = values[2].(uint64)
		activationOK := out.ActivationBlock == activationBlock
		if allowDynamicActivation {
			activationOK = receipt.BlockNumber != nil && out.ActivationBlock == receipt.BlockNumber.Uint64()
		}
		out.Consistent = exportedIdentifier(out.Name) == exportedIdentifier(name) && out.Version == version && activationOK
		if !out.Consistent {
			out.Error = fmt.Sprintf("unexpected event fields: name=%s version=%d activationBlock=%d", out.Name, out.Version, out.ActivationBlock)
		}
		return out
	}
	out.Error = "codeVersionUploaded event not found"
	return out
}

func waitForAllNodesAtLeast(ctx context.Context, targets []network.NodeConfig, block uint64, pollInterval time.Duration) error {
	for {
		allReady := true
		for _, node := range targets {
			client, err := rpc.DialContext(ctx, network.RPCURL(node))
			if err != nil {
				return err
			}
			height, err := blockNumber(ctx, client)
			client.Close()
			if err != nil {
				return err
			}
			if height < block {
				allReady = false
			}
		}
		if allReady {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func blockNumber(ctx context.Context, client *rpc.Client) (uint64, error) {
	var number hexutil.Uint64
	if err := client.CallContext(ctx, &number, "eth_blockNumber"); err != nil {
		return 0, err
	}
	return uint64(number), nil
}

func blockHash(ctx context.Context, client *rpc.Client, blockNumber uint64) (common.Hash, error) {
	var block map[string]json.RawMessage
	if err := client.CallContext(ctx, &block, "eth_getBlockByNumber", blockTag(blockNumber), false); err != nil {
		return common.Hash{}, err
	}
	raw, ok := block["hash"]
	if !ok {
		return common.Hash{}, fmt.Errorf("block %d has no hash", blockNumber)
	}
	var hash common.Hash
	if err := json.Unmarshal(raw, &hash); err != nil {
		return common.Hash{}, err
	}
	return hash, nil
}

func blockTag(blockNumber uint64) string {
	return hexutil.EncodeUint64(blockNumber)
}

func readPeerCount(ctx context.Context, client *rpc.Client) (uint64, error) {
	var count hexutil.Uint64
	if err := client.CallContext(ctx, &count, "net_peerCount"); err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func summarizeSampleChecks(sample sampleResult) consistencyChecks {
	checks := consistencyChecks{
		ChainViewConsistent: true,
		ReceiptConsistent:   true,
		EventConsistent:     true,
		VersionConsistent:   true,
		OutputConsistent:    true,
		Passed:              true,
	}
	var blockHash string
	var receiptHash string
	for _, node := range sample.Nodes {
		if node.Error != "" {
			checks.Passed = false
			checks.Failures = append(checks.Failures, fmt.Sprintf("%s: %s", node.NodeID, node.Error))
		}
		if blockHash == "" {
			blockHash = node.BlockHash
		} else if node.BlockHash != blockHash {
			checks.ChainViewConsistent = false
		}
		if !node.ReceiptVisible {
			checks.ReceiptConsistent = false
		}
		if receiptHash == "" {
			receiptHash = node.ReceiptBlockHash
		} else if node.ReceiptBlockHash != receiptHash {
			checks.ReceiptConsistent = false
		}
		if node.Version != sample.ExpectedVersion {
			checks.VersionConsistent = false
		}
		if node.Output != sample.ExpectedOutput {
			checks.OutputConsistent = false
		}
	}
	checks.Passed = checks.Passed &&
		checks.ChainViewConsistent &&
		checks.ReceiptConsistent &&
		checks.EventConsistent &&
		checks.VersionConsistent &&
		checks.OutputConsistent
	if !checks.ChainViewConsistent {
		checks.Failures = append(checks.Failures, sample.Stage+": chain view mismatch")
	}
	if !checks.ReceiptConsistent {
		checks.Failures = append(checks.Failures, sample.Stage+": receipt mismatch")
	}
	if !checks.VersionConsistent {
		checks.Failures = append(checks.Failures, sample.Stage+": version mismatch")
	}
	if !checks.OutputConsistent {
		checks.Failures = append(checks.Failures, sample.Stage+": output mismatch")
	}
	return checks
}

func summarizeRoundChecks(round roundResult) consistencyChecks {
	checks := consistencyChecks{
		ChainViewConsistent: true,
		ReceiptConsistent:   true,
		EventConsistent:     round.Event.Consistent,
		VersionConsistent:   true,
		OutputConsistent:    true,
		Passed:              round.Event.Consistent,
	}
	if !round.Event.Consistent {
		checks.Failures = append(checks.Failures, round.Event.Error)
	}
	for _, sample := range round.Samples {
		if !sample.Checks.ChainViewConsistent {
			checks.ChainViewConsistent = false
		}
		if !sample.Checks.ReceiptConsistent {
			checks.ReceiptConsistent = false
		}
		if !sample.Checks.VersionConsistent {
			checks.VersionConsistent = false
		}
		if !sample.Checks.OutputConsistent {
			checks.OutputConsistent = false
		}
		checks.Failures = append(checks.Failures, sample.Checks.Failures...)
	}
	checks.Passed = checks.Passed &&
		checks.ChainViewConsistent &&
		checks.ReceiptConsistent &&
		checks.EventConsistent &&
		checks.VersionConsistent &&
		checks.OutputConsistent &&
		len(checks.Failures) == 0
	return checks
}

func summarizeNodes(targets []network.NodeConfig, samples []sampleResult) []nodeSummary {
	out := make([]nodeSummary, len(targets))
	for i, node := range targets {
		out[i] = nodeSummary{NodeID: node.ID, Role: node.Role, RPCURL: network.RPCURL(node), Completed: true}
		for _, sample := range samples {
			for _, result := range sample.Nodes {
				if result.NodeID != node.ID {
					continue
				}
				if !result.OK {
					out[i].Completed = false
					out[i].Error = result.Error
				}
			}
		}
	}
	return out
}

func summarizeExperiment(rounds []roundResult) experimentSummary {
	summary := experimentSummary{RoundCount: len(rounds), Completed: len(rounds) > 0}
	for _, round := range rounds {
		if round.Completed {
			summary.CompletedRoundCount++
			continue
		}
		summary.Completed = false
		summary.FailedRounds = append(summary.FailedRounds, round.Index)
		if round.Error != "" {
			summary.FailureReasons = append(summary.FailureReasons, round.Error)
		}
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
		return err
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func writeCSVResult(path string, res experimentResult) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{
		"round", "mode", "stage", "block_number", "block_hash", "expected_version", "expected_output",
		"node_id", "role", "rpc_url", "head_number", "head_hash", "receipt_visible", "receipt_block", "receipt_block_hash",
		"version", "activation_block", "metadata_hash", "output", "output_hex", "ok", "error",
	}); err != nil {
		return err
	}
	for _, round := range res.Rounds {
		for _, sample := range round.Samples {
			for _, node := range sample.Nodes {
				if err := writer.Write([]string{
					fmt.Sprint(round.Index),
					round.Mode,
					sample.Stage,
					fmt.Sprint(sample.BlockNumber),
					sample.BlockHash,
					fmt.Sprint(sample.ExpectedVersion),
					sample.ExpectedOutput,
					node.NodeID,
					node.Role,
					node.RPCURL,
					fmt.Sprint(node.HeadNumber),
					node.HeadHash,
					fmt.Sprint(node.ReceiptVisible),
					fmt.Sprint(node.ReceiptBlock),
					node.ReceiptBlockHash,
					fmt.Sprint(node.Version),
					fmt.Sprint(node.ActivationBlock),
					node.MetadataHash,
					node.Output,
					node.OutputHex,
					fmt.Sprint(node.OK),
					node.Error,
				}); err != nil {
					return err
				}
			}
		}
	}
	return writer.Error()
}

func writeSummaryResult(path string, res experimentResult) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(formatSummary(res)), 0644)
}

func formatSummary(res experimentResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "experiment=%s change=%s completed=%t rounds=%d/%d\n",
		res.Experiment, res.Change, res.Summary.Completed, res.Summary.CompletedRoundCount, res.Summary.RoundCount)
	for _, round := range res.Rounds {
		fmt.Fprintf(&b, "round=%d mode=%s tx=%s completed=%t", round.Index, round.Mode, round.TransactionHash, round.Completed)
		if round.Error != "" {
			fmt.Fprintf(&b, " error=%s", round.Error)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "json=%s\ncsv=%s\nsummary=%s\n", res.Artifacts.ResultJSON, res.Artifacts.ResultCSV, res.Artifacts.ResultSummary)
	return b.String()
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
	case []byte:
		return hexutil.Encode(v)
	case [32]byte:
		return hexutil.Encode(v[:])
	case bool:
		return fmt.Sprint(v)
	default:
		return fmt.Sprint(v)
	}
}

func exportedIdentifier(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
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

func millis(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
