// benchupgradelatency 测量多节点网络中 WASM 升级交易的 Gas 与端到端延迟（Lab1）。
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
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

const experimentName = "lab1-upgrade-latency"

type phaseTimeline struct {
	TransactionSubmittedAt time.Time `json:"transactionSubmittedAt"`
	TxHashObservedAt         time.Time `json:"txHashObservedAt"`
	ReceiptObservedAt        time.Time `json:"receiptObservedAt"`
	EventObservedAt          time.Time `json:"eventObservedAt"`
	ActivationObservedAt     time.Time `json:"activationObservedAt,omitempty"`
}

type roundSummary struct {
	SubmitToTxHashMillis          int64 `json:"submitToTxHashMillis"`
	TxHashToSenderReceiptMillis   int64 `json:"txHashToSenderReceiptMillis"`
	EventToAllCompleteMillis      int64 `json:"eventToAllCompleteMillis"`
	SubmitToAllCompleteMillis     int64 `json:"submitToAllCompleteMillis"`
	MeanNodeSubmitToCompleteMillis int64 `json:"meanNodeSubmitToCompleteMillis"`
}

type nodeObservation struct {
	ID                         string     `json:"id"`
	Role                       string     `json:"role"`
	RPCURL                     string     `json:"rpcURL"`
	ReceiptObservedAt          *time.Time `json:"receiptObservedAt,omitempty"`
	CompletedAt                *time.Time `json:"completedAt,omitempty"`
	SubmitToReceiptMillis      int64      `json:"submitToReceiptMillis,omitempty"`
	ReceiptToCompleteMillis      int64      `json:"receiptToCompleteMillis,omitempty"`
	SubmitToCompleteMillis     int64      `json:"submitToCompleteMillis,omitempty"`
	EventToCompleteMillis      int64      `json:"eventToCompleteMillis,omitempty"`
	Completed                  bool       `json:"completed"`
	Error                      string     `json:"error,omitempty"`
}

type roundResult struct {
	Round            int             `json:"round"`
	Algorithm        string          `json:"algorithm"`
	UpgradeName      string          `json:"upgradeName"`
	SourcePath       string          `json:"sourcePath"`
	TxHash           string          `json:"txHash"`
	GasUsed          uint64          `json:"gasUsed"`
	ReceiptBlock     uint64          `json:"receiptBlock"`
	Completed        bool            `json:"completed"`
	PhaseTimeline    phaseTimeline   `json:"phaseTimeline"`
	Summary          roundSummary    `json:"summary"`
	Nodes            []nodeObservation `json:"nodes"`
	Error            string          `json:"error,omitempty"`
}

type experimentResult struct {
	Experiment      string        `json:"experiment"`
	ConfigPath      string        `json:"configPath"`
	ChainID         uint64        `json:"chainId"`
	NodeCount       int           `json:"nodeCount"`
	SenderNode      string        `json:"senderNode"`
	SenderRPC       string        `json:"senderRPC"`
	PollInterval    string        `json:"pollInterval"`
	Timeout         string        `json:"timeout"`
	Limitation      string        `json:"limitation"`
	Rounds          []roundResult `json:"rounds"`
	GeneratedAt     time.Time     `json:"generatedAt"`
}

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type addFixture struct {
	sourcePath  string
	inputTypes  []string
	outputTypes []string
	values      []interface{}
	expected    *big.Int
	algoGas     uint64
}

func main() {
	var (
		configPath   = flag.String("config", "", "render 后的 network.yaml 路径")
		senderID     = flag.String("sender", "", "发送升级交易的节点 ID（默认第一个 signer）")
		wasmSource   = flag.String("source", "experiments/cryptoupgrade/algorithm/go/archive/add.go", "WASM 源码或 .wasm 路径")
		algorithm    = flag.String("algorithm", "Add", "算法展示名")
		upgradeName  = flag.String("name", "", "uploadCode 函数名（默认 algorithm + Latency001）")
		rounds       = flag.Int("rounds", 1, "重复轮次")
		pollInterval = flag.Duration("poll-interval", 500*time.Millisecond, "节点轮询间隔")
		timeout      = flag.Duration("timeout", 5*time.Minute, "单轮超时")
		uploadGas    = flag.Uint64("tx-gas", 8000000, "升级交易 gas 上限")
		algoGas      = flag.Uint64("algo-gas", 3000, "算法 gas metadata")
		outputDir    = flag.String("out", "", "结果输出目录")
		skipPreflight = flag.Bool("skip-preflight", false, "跳过网络前置检查")
	)
	flag.Parse()
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "usage: benchupgradelatency -config <network.yaml> [-out <dir>]")
		os.Exit(2)
	}

	cfg, err := network.LoadConfig(*configPath)
	if err != nil {
		fatal(err)
	}
	sender, err := selectSender(cfg, *senderID)
	if err != nil {
		fatal(err)
	}
	targets := cfg.Nodes
	if len(targets) == 0 {
		fatal(fmt.Errorf("network config has no nodes"))
	}

	outDir := *outputDir
	if outDir == "" {
		ts := time.Now().UTC().Format("20060102-150405")
		outDir = filepath.Join("experiments", "cryptoupgrade", "results", "upgrade-latency", fmt.Sprintf("lab1-30nodes-%s", ts))
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fatal(err)
	}

	ctx := context.Background()
	if !*skipPreflight {
		fmt.Println("preflight: validating network...")
		preflightCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		vr, err := network.ValidateNetwork(preflightCtx, cfg, network.ValidateOptions{MinPeerCount: max(0, len(cfg.Nodes)-1)})
		if err != nil {
			fatal(err)
		}
		if !vr.OK {
			fatal(fmt.Errorf("network preflight failed"))
		}
		fmt.Printf("preflight ok: %d nodes reachable\n", len(vr.Nodes))
	}

	senderRPC := network.RPCURL(sender)
	senderClient, err := rpc.DialContext(ctx, senderRPC)
	if err != nil {
		fatal(fmt.Errorf("dial sender rpc %s: %w", senderRPC, err))
	}
	defer senderClient.Close()

	from, err := resolveSender(ctx, senderClient, sender.Account)
	if err != nil {
		fatal(err)
	}

	fixture := addFixture{
		sourcePath:  *wasmSource,
		inputTypes:  []string{"int256", "int256"},
		outputTypes: []string{"int256"},
		values:      []interface{}{big.NewInt(7), big.NewInt(11)},
		expected:    big.NewInt(18),
		algoGas:     *algoGas,
	}

	codeStorageABI := cryptoupgrade.CodeStorageABI
	result := experimentResult{
		Experiment:   experimentName,
		ConfigPath:   *configPath,
		ChainID:      cfg.Network.ChainID,
		NodeCount:    len(targets),
		SenderNode:   sender.ID,
		SenderRPC:    senderRPC,
		PollInterval: pollInterval.String(),
		Timeout:      timeout.String(),
		Limitation:   "control-plane observable latency including RPC round-trip and poll interval",
		GeneratedAt:  time.Now().UTC(),
	}

	for round := 1; round <= *rounds; round++ {
		name := *upgradeName
		if name == "" {
			name = fmt.Sprintf("%sLatency%03d", *algorithm, round)
		}
		fmt.Printf("\nround %d/%d: uploading %s...\n", round, *rounds, name)
		rr, err := runRound(ctx, cfg, sender, senderClient, from, targets, fixture, name, *algorithm, round, codeStorageABI, *uploadGas, *pollInterval, *timeout)
		if err != nil {
			rr.Error = err.Error()
			fmt.Printf("round %d failed: %v\n", round, err)
		} else {
			fmt.Printf("round %d: gasUsed=%d submitToAllComplete=%dms\n", round, rr.GasUsed, rr.Summary.SubmitToAllCompleteMillis)
		}
		result.Rounds = append(result.Rounds, rr)
	}

	jsonPath := filepath.Join(outDir, "result.json")
	if err := writeJSON(jsonPath, result); err != nil {
		fatal(err)
	}
	if err := writeRoundCSV(filepath.Join(outDir, "rounds.csv"), result.Rounds); err != nil {
		fatal(err)
	}
	if err := writeNodeCSV(filepath.Join(outDir, "nodes.csv"), result.Rounds); err != nil {
		fatal(err)
	}
	fmt.Printf("\nresults saved to %s\n", outDir)
}

func runRound(
	ctx context.Context,
	cfg *network.Config,
	sender network.NodeConfig,
	senderClient *rpc.Client,
	from common.Address,
	targets []network.NodeConfig,
	fixture addFixture,
	upgradeName, algorithm string,
	round int,
	codeStorageABI abi.ABI,
	uploadGas uint64,
	pollInterval, timeout time.Duration,
) (roundResult, error) {
	rr := roundResult{
		Round:       round,
		Algorithm:   algorithm,
		UpgradeName: upgradeName,
		SourcePath:  mustAbs(fixture.sourcePath),
	}

	wasm, encoded, err := wasmtool.BuildEncodedPath(ctx, fixture.sourcePath, wasmtool.Spec{
		Function:    upgradeName,
		InputTypes:  fixture.inputTypes,
		OutputTypes: fixture.outputTypes,
	})
	if err != nil {
		return rr, fmt.Errorf("encode wasm: %w", err)
	}
	_ = wasm

	data, err := codeStorageABI.Pack("uploadCode", upgradeName, encoded, fixture.algoGas,
		strings.Join(fixture.inputTypes, ","), strings.Join(fixture.outputTypes, ","))
	if err != nil {
		return rr, fmt.Errorf("pack uploadCode: %w", err)
	}

	to := common.CodeStorageAddress
	gas := hexutil.Uint64(uploadGas)
	submittedAt := time.Now()
	rr.PhaseTimeline.TransactionSubmittedAt = submittedAt

	txHash, err := sendTransaction(ctx, senderClient, txArgs{From: from, To: &to, Gas: &gas, Data: data})
	if err != nil {
		return rr, fmt.Errorf("send upgrade tx: %w", err)
	}
	txHashAt := time.Now()
	rr.PhaseTimeline.TxHashObservedAt = txHashAt
	rr.TxHash = txHash.Hex()

	receipt, err := waitReceipt(ctx, senderClient, txHash, timeout)
	if err != nil {
		return rr, err
	}
	receiptAt := time.Now()
	rr.PhaseTimeline.ReceiptObservedAt = receiptAt
	rr.PhaseTimeline.EventObservedAt = receiptAt
	rr.GasUsed = receipt.GasUsed
	rr.ReceiptBlock = receipt.BlockNumber.Uint64()

	if receipt.Status != types.ReceiptStatusSuccessful {
		return rr, fmt.Errorf("upgrade tx reverted")
	}
	if !hasCodeUploadedEvent(receipt) {
		return rr, fmt.Errorf("codeUploaded event not found in receipt")
	}

	callData, err := buildCallFuncData(codeStorageABI, upgradeName, fixture)
	if err != nil {
		return rr, err
	}

	roundCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nodeResults := make([]nodeObservation, len(targets))
	var wg sync.WaitGroup
	for i, node := range targets {
		wg.Add(1)
		go func(i int, node network.NodeConfig) {
			defer wg.Done()
			nodeResults[i] = observeNode(roundCtx, node, txHash, submittedAt, receiptAt, callData, fixture, codeStorageABI, pollInterval)
		}(i, node)
	}
	wg.Wait()
	rr.Nodes = nodeResults

	allCompleted := true
	var latestComplete time.Time
	var completeSum int64
	var completeCount int64
	for _, nr := range nodeResults {
		if !nr.Completed {
			allCompleted = false
			continue
		}
		if nr.CompletedAt != nil && nr.CompletedAt.After(latestComplete) {
			latestComplete = *nr.CompletedAt
		}
		completeSum += nr.SubmitToCompleteMillis
		completeCount++
	}

	rr.Summary = roundSummary{
		SubmitToTxHashMillis:        millis(submittedAt, txHashAt),
		TxHashToSenderReceiptMillis: millis(txHashAt, receiptAt),
	}
	if allCompleted && !latestComplete.IsZero() {
		rr.Completed = true
		rr.PhaseTimeline.ActivationObservedAt = latestComplete
		rr.Summary.EventToAllCompleteMillis = millis(receiptAt, latestComplete)
		rr.Summary.SubmitToAllCompleteMillis = millis(submittedAt, latestComplete)
		if completeCount > 0 {
			rr.Summary.MeanNodeSubmitToCompleteMillis = completeSum / completeCount
		}
	}
	return rr, nil
}

func observeNode(
	ctx context.Context,
	node network.NodeConfig,
	txHash common.Hash,
	submittedAt, eventAt time.Time,
	callData []byte,
	fixture addFixture,
	codeStorageABI abi.ABI,
	pollInterval time.Duration,
) nodeObservation {
	rpcURL := network.RPCURL(node)
	out := nodeObservation{ID: node.ID, Role: node.Role, RPCURL: rpcURL}
	if node.HTTPHostPort == 0 {
		out.Error = "node RPC not exposed to host"
		return out
	}

	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer client.Close()

	var receiptAt, completedAt time.Time

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if out.Error == "" {
				out.Error = ctx.Err().Error()
			}
			return out
		case <-ticker.C:
			if receiptAt.IsZero() {
				var receipt *types.Receipt
				if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err == nil && receipt != nil {
					now := time.Now()
					receiptAt = now
					out.ReceiptObservedAt = &now
				}
			}
			if !receiptAt.IsZero() && !out.Completed {
				output, err := ethCall(ctx, client, txArgs{
					To:   ptr(common.CodeStorageAddress),
					Data: callData,
				})
				if err == nil {
					if matched, _ := outputMatches(codeStorageABI, output, fixture); matched {
						now := time.Now()
						completedAt = now
						out.CompletedAt = &now
						out.Completed = true
					}
				}
			}
			if out.Completed {
				out.SubmitToReceiptMillis = millis(submittedAt, receiptAt)
				out.ReceiptToCompleteMillis = millis(receiptAt, completedAt)
				out.SubmitToCompleteMillis = millis(submittedAt, completedAt)
				out.EventToCompleteMillis = millis(eventAt, completedAt)
				return out
			}
		}
	}
}

func buildCallFuncData(codeStorageABI abi.ABI, name string, fixture addFixture) ([]byte, error) {
	args, err := abiArguments(fixture.inputTypes...)
	if err != nil {
		return nil, err
	}
	encoded, err := args.Pack(fixture.values...)
	if err != nil {
		return nil, fmt.Errorf("pack call input: %w", err)
	}
	return codeStorageABI.Pack("callFunc", name, encoded)
}

func outputMatches(codeStorageABI abi.ABI, output []byte, fixture addFixture) (bool, error) {
	retArgs, err := abiArguments(fixture.outputTypes...)
	if err != nil {
		return false, err
	}
	values, err := retArgs.Unpack(output)
	if err != nil {
		return false, err
	}
	if len(values) != 1 {
		return false, fmt.Errorf("unexpected output arity %d", len(values))
	}
	got, ok := values[0].(*big.Int)
	if !ok {
		return false, fmt.Errorf("unexpected output type %T", values[0])
	}
	return got.Cmp(fixture.expected) == 0, nil
}

func hasCodeUploadedEvent(receipt *types.Receipt) bool {
	for _, lg := range receipt.Logs {
		if len(lg.Topics) > 0 && lg.Topics[0] == cryptoupgrade.CodeStorageABI.Events["codeUploaded"].ID {
			return true
		}
	}
	return false
}

func selectSender(cfg *network.Config, senderID string) (network.NodeConfig, error) {
	if senderID != "" {
		for _, node := range cfg.Nodes {
			if node.ID == senderID {
				return node, nil
			}
		}
		return network.NodeConfig{}, fmt.Errorf("sender node %q not found", senderID)
	}
	for _, node := range cfg.Nodes {
		if node.Role == "signer" {
			return node, nil
		}
	}
	return network.NodeConfig{}, fmt.Errorf("no signer node in config")
}

func resolveSender(ctx context.Context, client *rpc.Client, account string) (common.Address, error) {
	if account != "" {
		return common.HexToAddress(account), nil
	}
	var accounts []common.Address
	if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
		return common.Address{}, err
	}
	if len(accounts) == 0 {
		return common.Address{}, fmt.Errorf("no unlocked accounts on sender node")
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
			return nil, err
		}
		if receipt != nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for receipt %s", txHash.Hex())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func ethCall(ctx context.Context, client *rpc.Client, args txArgs) ([]byte, error) {
	var output hexutil.Bytes
	if err := client.CallContext(ctx, &output, "eth_call", args, "latest"); err != nil {
		return nil, err
	}
	return output, nil
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

func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

func writeRoundCSV(path string, rounds []roundResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{
		"round", "algorithm", "upgradeName", "txHash", "gasUsed", "receiptBlock", "completed",
		"submitToTxHashMs", "txHashToSenderReceiptMs", "eventToAllCompleteMs", "submitToAllCompleteMs",
		"meanNodeSubmitToCompleteMs",
	})
	for _, r := range rounds {
		_ = w.Write([]string{
			fmt.Sprintf("%d", r.Round),
			r.Algorithm,
			r.UpgradeName,
			r.TxHash,
			fmt.Sprintf("%d", r.GasUsed),
			fmt.Sprintf("%d", r.ReceiptBlock),
			fmt.Sprintf("%t", r.Completed),
			fmt.Sprintf("%d", r.Summary.SubmitToTxHashMillis),
			fmt.Sprintf("%d", r.Summary.TxHashToSenderReceiptMillis),
			fmt.Sprintf("%d", r.Summary.EventToAllCompleteMillis),
			fmt.Sprintf("%d", r.Summary.SubmitToAllCompleteMillis),
			fmt.Sprintf("%d", r.Summary.MeanNodeSubmitToCompleteMillis),
		})
	}
	return nil
}

func writeNodeCSV(path string, rounds []roundResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{
		"round", "nodeId", "role", "rpcURL", "completed",
		"submitToReceiptMs", "receiptToCompleteMs", "submitToCompleteMs", "eventToCompleteMs", "error",
	})
	for _, r := range rounds {
		for _, n := range r.Nodes {
			_ = w.Write([]string{
				fmt.Sprintf("%d", r.Round),
				n.ID,
				n.Role,
				n.RPCURL,
				fmt.Sprintf("%t", n.Completed),
				fmt.Sprintf("%d", n.SubmitToReceiptMillis),
				fmt.Sprintf("%d", n.ReceiptToCompleteMillis),
				fmt.Sprintf("%d", n.SubmitToCompleteMillis),
				fmt.Sprintf("%d", n.EventToCompleteMillis),
				n.Error,
			})
		}
	}
	return nil
}

func millis(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func ptr[T any](v T) *T { return &v }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
