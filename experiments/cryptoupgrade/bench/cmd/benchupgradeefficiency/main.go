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
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	changeName     = "benchmark-upgrade-efficiency"
	experimentName = "upgrade-efficiency-add-single"
	defaultRPC     = "http://127.0.0.1:8666"
)

type config struct {
	repoRoot        string
	resultDir       string
	outputJSON      string
	logPath         string
	pluginDir       string
	reuseRPC        bool
	rpc             string
	rpcReadyTimeout time.Duration
	receiptTimeout  time.Duration

	gethWorkdir string
	gethBin     string
	datadir     string
	password    string
	networkID   string
	httpAddr    string
	httpPort    string
	httpAPI     string

	sourcePath string
	name       string
	inputType  string
	outputType string
	algoGas    uint64
	txGas      uint64
	from       string
	a          string
	b          string
}

type txArgs struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  *hexutil.Uint64 `json:"gas,omitempty"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type experimentResult struct {
	Experiment  string           `json:"experiment"`
	Change      string           `json:"change"`
	Timestamp   string           `json:"timestamp"`
	Command     []string         `json:"command"`
	Chain       chainResult      `json:"chain"`
	Artifacts   artifactResult   `json:"artifacts"`
	Upgrade     upgradeResult    `json:"upgrade"`
	Activation  activationResult `json:"activation"`
	Validation  validationResult `json:"validation"`
	Latencies   latencyResult    `json:"latencies"`
	Limitations []string         `json:"limitations"`
	Completed   bool             `json:"completed"`
	Error       string           `json:"error,omitempty"`
}

type chainResult struct {
	RPC           string   `json:"rpc"`
	ReuseRPC      bool     `json:"reuseRpc"`
	GethBinary    string   `json:"gethBinary,omitempty"`
	GethWorkdir   string   `json:"gethWorkdir,omitempty"`
	GethArgs      []string `json:"gethArgs,omitempty"`
	GethPID       int      `json:"gethPid,omitempty"`
	ClientVersion string   `json:"clientVersion,omitempty"`
	Sender        string   `json:"sender,omitempty"`
	RPCReadyAt    string   `json:"rpcReadyAt,omitempty"`
	RPCReadyBlock uint64   `json:"rpcReadyBlock,omitempty"`
}

type artifactResult struct {
	ResultDir  string `json:"resultDir"`
	ResultJSON string `json:"resultJson"`
	GethLog    string `json:"gethLog"`
	PluginDir  string `json:"pluginDir"`
	SourcePath string `json:"sourcePath"`
}

type upgradeResult struct {
	Algorithm               string        `json:"algorithm"`
	InputType               string        `json:"inputType"`
	OutputType              string        `json:"outputType"`
	AlgoGas                 uint64        `json:"algoGas"`
	TransactionGasLimit     uint64        `json:"transactionGasLimit"`
	TransactionHash         string        `json:"transactionHash,omitempty"`
	ReceiptStatus           uint64        `json:"receiptStatus"`
	ReceiptGasUsed          uint64        `json:"receiptGasUsed"`
	ReceiptLogCount         int           `json:"receiptLogCount"`
	SubmissionStartedAt     string        `json:"submissionStartedAt,omitempty"`
	SubmissionReturnedAt    string        `json:"submissionReturnedAt,omitempty"`
	ConfirmationObservedAt  string        `json:"confirmationObservedAt,omitempty"`
	SubmitBlockNumber       uint64        `json:"submitBlockNumber"`
	ConfirmationBlockNumber uint64        `json:"confirmationBlockNumber"`
	Payload                 payloadResult `json:"payload"`
}

type payloadResult struct {
	SourceBytes           int    `json:"sourceBytes"`
	CompressedGzipBytes   int    `json:"compressedGzipBytes"`
	CompressedBase64Bytes int    `json:"compressedBase64Bytes"`
	UploadCalldataBytes   int    `json:"uploadCalldataBytes"`
	UploadCalldataHex     string `json:"uploadCalldataHex"`
	UploadCalldataSHA256  string `json:"uploadCalldataSha256"`
}

type activationResult struct {
	BlockNumber                        uint64 `json:"blockNumber"`
	BlockNumberSource                  string `json:"blockNumberSource"`
	ObservationMethod                  string `json:"observationMethod"`
	ObservationBlockNumber             uint64 `json:"observationBlockNumber"`
	ObservedAt                         string `json:"observedAt,omitempty"`
	Available                          bool   `json:"available"`
	CodeMatchesUploadPayload           bool   `json:"codeMatchesUploadPayload"`
	Gas                                uint64 `json:"gas"`
	InputType                          string `json:"inputType,omitempty"`
	OutputType                         string `json:"outputType,omitempty"`
	ConfirmationAndActivationSamePoint bool   `json:"confirmationAndActivationSameExecutionPoint"`
}

type validationResult struct {
	Method          string `json:"method"`
	BlockNumber     uint64 `json:"blockNumber"`
	StartedAt       string `json:"startedAt,omitempty"`
	FinishedAt      string `json:"finishedAt,omitempty"`
	InputA          string `json:"inputA"`
	InputB          string `json:"inputB"`
	EncodedInputHex string `json:"encodedInputHex"`
	CallDataBytes   int    `json:"callDataBytes"`
	CallDataHex     string `json:"callDataHex"`
	RawOutputHex    string `json:"rawOutputHex,omitempty"`
	DecodedResult   string `json:"decodedResult,omitempty"`
	ExpectedResult  string `json:"expectedResult"`
	Success         bool   `json:"success"`
}

type latencyResult struct {
	SubmissionRPCMillis              float64 `json:"submissionRpcMillis"`
	SubmitToConfirmMillis            float64 `json:"submitToConfirmMillis"`
	ConfirmToActivateObservedMillis  float64 `json:"confirmToActivateObservedMillis"`
	ConfirmToActivateEffectiveMillis float64 `json:"confirmToActivateEffectiveMillis"`
	SubmitToAlgorithmAvailableMillis float64 `json:"submitToAlgorithmAvailableMillis"`
	ValidationCallMillis             float64 `json:"validationCallMillis"`
}

type gethProcess struct {
	cmd     *exec.Cmd
	errCh   <-chan error
	logFile *os.File
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchupgradeefficiency: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "benchupgradeefficiency: failed to write JSON result: %v\n", writeErr)
		}
	}
	printSummary(res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchupgradeefficiency: %v\n", err)
		os.Exit(1)
	}
}

func parseConfig() (config, error) {
	cfg := config{}
	flag.StringVar(&cfg.repoRoot, "repo-root", ".", "repository root")
	flag.StringVar(&cfg.resultDir, "result-dir", "", "directory for JSON result, geth log, and isolated plugin artifacts")
	flag.StringVar(&cfg.outputJSON, "output-json", "", "write raw experiment JSON to this path")
	flag.StringVar(&cfg.logPath, "geth-log", "", "write geth stdout/stderr to this path")
	flag.StringVar(&cfg.pluginDir, "plugin-dir", "", "GETH_CRYPTOUPGRADE_PLUGIN_DIR for the geth process")
	flag.BoolVar(&cfg.reuseRPC, "reuse-rpc", false, "reuse an already running RPC endpoint instead of starting geth")
	flag.StringVar(&cfg.rpc, "rpc", defaultRPC, "execution JSON-RPC endpoint")
	flag.DurationVar(&cfg.rpcReadyTimeout, "rpc-ready-timeout", 45*time.Second, "timeout waiting for JSON-RPC readiness")
	flag.DurationVar(&cfg.receiptTimeout, "receipt-timeout", 2*time.Minute, "timeout waiting for the upload transaction receipt")

	flag.StringVar(&cfg.gethWorkdir, "geth-workdir", "build/bin/chain", "working directory for the geth command from chain_setup.md")
	flag.StringVar(&cfg.gethBin, "geth-bin", "./geth", "geth executable path, relative to -geth-workdir unless absolute")
	flag.StringVar(&cfg.datadir, "datadir", "chain/node1", "geth --datadir value")
	flag.StringVar(&cfg.password, "password", "chain/password.txt", "geth --password value")
	flag.StringVar(&cfg.networkID, "networkid", "11223344", "geth --networkid value")
	flag.StringVar(&cfg.httpAddr, "http-addr", "0.0.0.0", "geth --http.addr value")
	flag.StringVar(&cfg.httpPort, "http-port", "8666", "geth --http.port value")
	flag.StringVar(&cfg.httpAPI, "http-api", "web3,eth,debug,net,admin", "geth --http.api value")

	flag.StringVar(&cfg.sourcePath, "source", "cryptoupgrade/algorithm/go/add.go", "Add algorithm Go source")
	flag.StringVar(&cfg.name, "name", "Add", "algorithm name")
	flag.StringVar(&cfg.inputType, "itype", "int256,int256", "CodeStorage input ABI type list")
	flag.StringVar(&cfg.outputType, "otype", "int256", "CodeStorage output ABI type list")
	flag.Uint64Var(&cfg.algoGas, "algo-gas", 1, "algorithm gas recorded in CodeStorage")
	flag.Uint64Var(&cfg.txGas, "tx-gas", 8000000, "upload transaction gas limit")
	flag.StringVar(&cfg.from, "from", "", "sender address; defaults to eth_accounts[0]")
	flag.StringVar(&cfg.a, "a", "100", "first Add int256 argument")
	flag.StringVar(&cfg.b, "b", "100", "second Add int256 argument")
	flag.Parse()

	var err error
	cfg.repoRoot, err = filepath.Abs(cfg.repoRoot)
	if err != nil {
		return cfg, err
	}
	timestamp := time.Now().Format("20060102-150405")
	if cfg.resultDir == "" {
		cfg.resultDir = filepath.Join(cfg.repoRoot, "experiments", "cryptoupgrade", "results", "upgrade-efficiency", "add-"+timestamp)
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
	if cfg.logPath == "" {
		cfg.logPath = filepath.Join(cfg.resultDir, "geth.log")
	} else if !filepath.IsAbs(cfg.logPath) {
		cfg.logPath = filepath.Join(cfg.repoRoot, cfg.logPath)
	}
	if cfg.pluginDir == "" {
		cfg.pluginDir = filepath.Join(cfg.resultDir, "plugin")
	} else if !filepath.IsAbs(cfg.pluginDir) {
		cfg.pluginDir = filepath.Join(cfg.repoRoot, cfg.pluginDir)
	}
	if !filepath.IsAbs(cfg.gethWorkdir) {
		cfg.gethWorkdir = filepath.Join(cfg.repoRoot, cfg.gethWorkdir)
	}
	if !filepath.IsAbs(cfg.sourcePath) {
		cfg.sourcePath = filepath.Join(cfg.repoRoot, cfg.sourcePath)
	}
	if cfg.rpc == "" {
		return cfg, errors.New("-rpc is required")
	}
	if cfg.name != "Add" {
		return cfg, errors.New("this experiment phase is Add-only; -name must be Add")
	}
	if cfg.inputType != "int256,int256" || cfg.outputType != "int256" {
		return cfg, errors.New("this experiment phase requires -itype int256,int256 and -otype int256")
	}
	if cfg.txGas == 0 {
		return cfg, errors.New("-tx-gas must be non-zero; upload gas estimation can mutate plugin state")
	}
	if cfg.from != "" && !common.IsHexAddress(cfg.from) {
		return cfg, fmt.Errorf("invalid -from address %q", cfg.from)
	}
	if _, err := os.Stat(cfg.sourcePath); err != nil {
		return cfg, fmt.Errorf("source file %s: %w", cfg.sourcePath, err)
	}
	return cfg, nil
}

func newResult(cfg config) experimentResult {
	return experimentResult{
		Experiment: experimentName,
		Change:     changeName,
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Command:    os.Args,
		Chain: chainResult{
			RPC:         cfg.rpc,
			ReuseRPC:    cfg.reuseRPC,
			GethBinary:  resolveExecutableForResult(cfg),
			GethWorkdir: cfg.gethWorkdir,
			GethArgs:    gethArgs(cfg),
		},
		Artifacts: artifactResult{
			ResultDir:  cfg.resultDir,
			ResultJSON: cfg.outputJSON,
			GethLog:    cfg.logPath,
			PluginDir:  cfg.pluginDir,
			SourcePath: cfg.sourcePath,
		},
		Upgrade: upgradeResult{
			Algorithm:           cfg.name,
			InputType:           cfg.inputType,
			OutputType:          cfg.outputType,
			AlgoGas:             cfg.algoGas,
			TransactionGasLimit: cfg.txGas,
		},
		Activation: activationResult{
			BlockNumberSource:                  "latest_block_when_callFunc_succeeded",
			ObservationMethod:                  "CodeStorage.callFunc",
			ConfirmationAndActivationSamePoint: false,
		},
		Validation: validationResult{
			Method: `eth_call CodeStorage.callFunc("Add", encodedInput)`,
			InputA: cfg.a,
			InputB: cfg.b,
		},
		Limitations: []string{
			"uploadCode receipt 只表示升级数据和 codeUploaded event 已进入链上执行结果；本地 activation 由节点事件监听线程完成。",
			"activation block 使用首次 CodeStorage.callFunc 验证成功时的 latest block 记录，表示控制端可观测的本地可调用点。",
			"confirm-to-activate observed latency 包含事件订阅分发、getInfo 查询、plugin 编译和轮询间隔。",
			"validation 使用 eth_call，因此 validation block 表示 latest 调用上下文，不是验证交易所在区块。",
			"geth --dev 按需出块，submit-to-confirm latency 包含本地 dev miner 调度，但不再包含 uploadCode 内部 plugin 编译时间。",
			"payload size 同时报告源码、gzip、base64 和完整 ABI calldata；当前实现没有单一字段能代表所有 payload 成本。",
		},
	}
}

func run(ctx context.Context, cfg config, res *experimentResult) error {
	var proc *gethProcess
	var err error
	if !cfg.reuseRPC {
		proc, err = startGeth(ctx, cfg)
		if err != nil {
			return err
		}
		res.Chain.GethPID = proc.cmd.Process.Pid
		defer proc.stop()
	}

	client, clientVersion, readyBlock, err := waitRPCReady(ctx, cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	res.Chain.ClientVersion = clientVersion
	res.Chain.RPCReadyAt = time.Now().Format(time.RFC3339Nano)
	res.Chain.RPCReadyBlock = readyBlock

	from, err := resolveSender(ctx, client, cfg.from)
	if err != nil {
		return err
	}
	res.Chain.Sender = from.Hex()

	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return err
	}
	sourceBytes, gzBytes, compressedPayload, err := compressSource(cfg.sourcePath)
	if err != nil {
		return err
	}
	uploadData, err := codeStorageABI.Pack("uploadCode", cfg.name, compressedPayload, cfg.algoGas, cfg.inputType, cfg.outputType)
	if err != nil {
		return fmt.Errorf("pack CodeStorage.uploadCode: %w", err)
	}
	sum := sha256.Sum256(uploadData)
	res.Upgrade.Payload = payloadResult{
		SourceBytes:           sourceBytes,
		CompressedGzipBytes:   gzBytes,
		CompressedBase64Bytes: len(compressedPayload),
		UploadCalldataBytes:   len(uploadData),
		UploadCalldataHex:     hexutil.Encode(uploadData),
		UploadCalldataSHA256:  hex.EncodeToString(sum[:]),
	}

	to := common.CodeStorageAddress
	gas := hexutil.Uint64(cfg.txGas)
	submitBlock, err := blockNumber(ctx, client)
	if err != nil {
		return fmt.Errorf("read submit block number: %w", err)
	}
	res.Upgrade.SubmitBlockNumber = submitBlock
	submitStarted := time.Now()
	res.Upgrade.SubmissionStartedAt = submitStarted.Format(time.RFC3339Nano)
	txHash, err := sendTransaction(ctx, client, txArgs{From: from, To: &to, Gas: &gas, Data: uploadData})
	submitReturned := time.Now()
	res.Upgrade.SubmissionReturnedAt = submitReturned.Format(time.RFC3339Nano)
	res.Latencies.SubmissionRPCMillis = millis(submitReturned.Sub(submitStarted))
	if err != nil {
		return fmt.Errorf("send upgrade transaction: %w", wrapPluginHint(err))
	}
	res.Upgrade.TransactionHash = txHash.Hex()

	receipt, confirmationObserved, err := waitReceipt(ctx, client, txHash, cfg.receiptTimeout)
	if err != nil {
		return err
	}
	res.Upgrade.ConfirmationObservedAt = confirmationObserved.Format(time.RFC3339Nano)
	res.Upgrade.ReceiptStatus = receipt.Status
	res.Upgrade.ReceiptGasUsed = receipt.GasUsed
	res.Upgrade.ReceiptLogCount = len(receipt.Logs)
	if receipt.BlockNumber != nil {
		res.Upgrade.ConfirmationBlockNumber = receipt.BlockNumber.Uint64()
	}
	res.Latencies.SubmitToConfirmMillis = millis(confirmationObserved.Sub(submitStarted))
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("upgrade transaction %s failed: status=%d gasUsed=%d", txHash.Hex(), receipt.Status, receipt.GasUsed)
	}

	validation, validationElapsed, activationObserved, err := waitActivation(ctx, client, codeStorageABI, from, cfg, 30*time.Second)
	if err != nil {
		return err
	}
	res.Validation = validation
	res.Latencies.ValidationCallMillis = millis(validationElapsed)
	res.Activation.BlockNumber = validation.BlockNumber
	res.Activation.ObservationBlockNumber = validation.BlockNumber
	res.Activation.ObservedAt = activationObserved.Format(time.RFC3339Nano)
	res.Activation.Available = true
	res.Activation.CodeMatchesUploadPayload = true
	res.Activation.Gas = cfg.algoGas
	res.Activation.InputType = cfg.inputType
	res.Activation.OutputType = cfg.outputType
	res.Latencies.ConfirmToActivateObservedMillis = millis(activationObserved.Sub(confirmationObserved))
	res.Latencies.ConfirmToActivateEffectiveMillis = res.Latencies.ConfirmToActivateObservedMillis
	res.Latencies.SubmitToAlgorithmAvailableMillis = millis(parseTime(validation.FinishedAt).Sub(submitStarted))
	res.Completed = true
	return nil
}

func startGeth(ctx context.Context, cfg config) (*gethProcess, error) {
	if _, err := os.Stat(cfg.gethWorkdir); err != nil {
		return nil, fmt.Errorf("geth workdir %s: %w", cfg.gethWorkdir, err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.logPath), 0755); err != nil {
		return nil, fmt.Errorf("create geth log dir: %w", err)
	}
	if err := os.MkdirAll(cfg.pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("create plugin dir: %w", err)
	}
	logFile, err := os.OpenFile(cfg.logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("open geth log: %w", err)
	}
	executable := cfg.gethBin
	if filepath.IsAbs(executable) {
		if _, err := os.Stat(executable); err != nil {
			_ = logFile.Close()
			return nil, fmt.Errorf("geth binary %s: %w", executable, err)
		}
	} else if _, err := os.Stat(filepath.Join(cfg.gethWorkdir, executable)); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("geth binary %s relative to %s: %w", executable, cfg.gethWorkdir, err)
	}

	cmd := exec.CommandContext(ctx, executable, gethArgs(cfg)...)
	cmd.Dir = cfg.gethWorkdir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(),
		"GETH_CRYPTOUPGRADE_PLUGIN_DIR="+cfg.pluginDir,
		"CRYPTOUPGRADE_MODULE="+cfg.repoRoot,
	)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start geth: %w", err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()
	return &gethProcess{cmd: cmd, errCh: errCh, logFile: logFile}, nil
}

func (p *gethProcess) stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Signal(syscall.SIGINT)
	select {
	case <-p.errCh:
	case <-time.After(10 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.errCh
	}
	_ = p.logFile.Close()
}

func gethArgs(cfg config) []string {
	return []string{
		"--dev",
		"--datadir", cfg.datadir,
		"--password", cfg.password,
		"--networkid", cfg.networkID,
		"--http",
		"--http.addr", cfg.httpAddr,
		"--http.port", cfg.httpPort,
		"--http.corsdomain", "*",
		"--http.vhosts", "*",
		"--http.api", cfg.httpAPI,
	}
}

func resolveExecutableForResult(cfg config) string {
	if filepath.IsAbs(cfg.gethBin) {
		return cfg.gethBin
	}
	return filepath.Join(cfg.gethWorkdir, cfg.gethBin)
}

func waitRPCReady(ctx context.Context, cfg config) (*rpc.Client, string, uint64, error) {
	deadline := time.Now().Add(cfg.rpcReadyTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		client, err := rpc.DialContext(ctx, cfg.rpc)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var version string
		if err := client.CallContext(ctx, &version, "web3_clientVersion"); err != nil {
			lastErr = err
			client.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var accounts []common.Address
		if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
			lastErr = err
			client.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if len(accounts) == 0 {
			lastErr = errors.New("eth_accounts returned no unlocked dev account")
			client.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		block, err := blockNumber(ctx, client)
		if err != nil {
			lastErr = err
			client.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return client, version, block, nil
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	return nil, "", 0, fmt.Errorf("wait RPC ready at %s: %w", cfg.rpc, lastErr)
}

func resolveSender(ctx context.Context, client *rpc.Client, fromText string) (common.Address, error) {
	if fromText != "" {
		return common.HexToAddress(fromText), nil
	}
	var accounts []common.Address
	if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil {
		return common.Address{}, fmt.Errorf("eth_accounts: %w", err)
	}
	if len(accounts) == 0 {
		return common.Address{}, errors.New("eth_accounts returned no unlocked accounts")
	}
	return accounts[0], nil
}

func compressSource(path string) (sourceBytes int, gzipBytes int, encoded string, err error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, "", fmt.Errorf("read source: %w", err)
	}
	encoded, err = cryptoupgrade.EncodeSource(source)
	if err != nil {
		return 0, 0, "", err
	}
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, 0, "", fmt.Errorf("decode shared source payload: %w", err)
	}
	return len(source), len(compressed), encoded, nil
}

func sendTransaction(ctx context.Context, client *rpc.Client, args txArgs) (common.Hash, error) {
	var txHash common.Hash
	if err := client.CallContext(ctx, &txHash, "eth_sendTransaction", args); err != nil {
		return common.Hash{}, err
	}
	return txHash, nil
}

func waitReceipt(ctx context.Context, client *rpc.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, time.Time, error) {
	deadline := time.Now().Add(timeout)
	for {
		var receipt *types.Receipt
		if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", txHash); err != nil {
			if !isReceiptIndexing(err) {
				return nil, time.Time{}, fmt.Errorf("eth_getTransactionReceipt %s: %w", txHash.Hex(), err)
			}
		}
		if receipt != nil {
			return receipt, time.Now(), nil
		}
		if time.Now().After(deadline) {
			return nil, time.Time{}, fmt.Errorf("timed out waiting for receipt %s", txHash.Hex())
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func isReceiptIndexing(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "transaction indexing is in progress")
}

func waitActivation(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config, timeout time.Duration) (validationResult, time.Duration, time.Time, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		validation, elapsed, err := validateAdd(ctx, client, codeStorageABI, from, cfg)
		if err == nil {
			return validation, elapsed, time.Now(), nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timeout")
	}
	return validationResult{}, 0, time.Time{}, fmt.Errorf("activation observation failed: %w", lastErr)
}

func validateAdd(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, cfg config) (validationResult, time.Duration, error) {
	result := validationResult{
		Method: `eth_call CodeStorage.callFunc("Add", encodedInput)`,
		InputA: cfg.a,
		InputB: cfg.b,
	}
	a, ok := new(big.Int).SetString(cfg.a, 10)
	if !ok {
		return result, 0, fmt.Errorf("invalid -a value %q", cfg.a)
	}
	b, ok := new(big.Int).SetString(cfg.b, 10)
	if !ok {
		return result, 0, fmt.Errorf("invalid -b value %q", cfg.b)
	}
	expected := new(big.Int).Add(a, b)
	result.ExpectedResult = expected.String()

	int256Type, err := abi.NewType("int256", "", nil)
	if err != nil {
		return result, 0, err
	}
	inputArgs := abi.Arguments{{Type: int256Type}, {Type: int256Type}}
	outputArgs := abi.Arguments{{Type: int256Type}}
	encodedInput, err := inputArgs.Pack(a, b)
	if err != nil {
		return result, 0, err
	}
	callData, err := codeStorageABI.Pack("callFunc", cfg.name, encodedInput)
	if err != nil {
		return result, 0, fmt.Errorf("pack CodeStorage.callFunc: %w", err)
	}
	block, err := blockNumber(ctx, client)
	if err != nil {
		return result, 0, err
	}
	result.BlockNumber = block
	result.EncodedInputHex = hexutil.Encode(encodedInput)
	result.CallDataBytes = len(callData)
	result.CallDataHex = hexutil.Encode(callData)

	started := time.Now()
	result.StartedAt = started.Format(time.RFC3339Nano)
	raw, err := ethCall(ctx, client, from, common.CodeStorageAddress, callData)
	finished := time.Now()
	result.FinishedAt = finished.Format(time.RFC3339Nano)
	if err != nil {
		return result, finished.Sub(started), fmt.Errorf("validate Add callFunc: %w", wrapPluginHint(err))
	}
	result.RawOutputHex = hexutil.Encode(raw)
	decoded, err := outputArgs.Unpack(raw)
	if err != nil {
		return result, finished.Sub(started), fmt.Errorf("decode Add output: %w", err)
	}
	got := decoded[0].(*big.Int)
	result.DecodedResult = got.String()
	result.Success = got.Cmp(expected) == 0
	if !result.Success {
		return result, finished.Sub(started), fmt.Errorf("Add validation failed: got %s want %s", got, expected)
	}
	return result, finished.Sub(started), nil
}

func ethCall(ctx context.Context, client *rpc.Client, from common.Address, to common.Address, data []byte) ([]byte, error) {
	gas := hexutil.Uint64(5000000)
	var out hexutil.Bytes
	args := txArgs{
		To:   &to,
		Gas:  &gas,
		Data: data,
	}
	if from != (common.Address{}) {
		args.From = from
	}
	if err := client.CallContext(ctx, &out, "eth_call", args, "latest"); err != nil {
		return nil, err
	}
	return out, nil
}

func blockNumber(ctx context.Context, client *rpc.Client) (uint64, error) {
	var number hexutil.Uint64
	if err := client.CallContext(ctx, &number, "eth_blockNumber"); err != nil {
		return 0, err
	}
	return uint64(number), nil
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

func printSummary(res experimentResult) {
	fmt.Printf("Upgrade efficiency experiment: %s\n", res.Experiment)
	fmt.Printf("  change:       %s\n", res.Change)
	fmt.Printf("  rpc:          %s\n", res.Chain.RPC)
	fmt.Printf("  sender:       %s\n", res.Chain.Sender)
	fmt.Printf("  tx:           %s\n", res.Upgrade.TransactionHash)
	fmt.Printf("  gas-used:     %d\n", res.Upgrade.ReceiptGasUsed)
	fmt.Printf("  blocks:       submit=%d confirm=%d activate=%d validate=%d\n",
		res.Upgrade.SubmitBlockNumber, res.Upgrade.ConfirmationBlockNumber,
		res.Activation.BlockNumber, res.Validation.BlockNumber)
	fmt.Printf("  latency-ms:   submit->confirm=%.3f confirm->activate(observed)=%.3f submit->available=%.3f\n",
		res.Latencies.SubmitToConfirmMillis, res.Latencies.ConfirmToActivateObservedMillis,
		res.Latencies.SubmitToAlgorithmAvailableMillis)
	fmt.Printf("  add-result:   %s + %s = %s success=%t\n",
		res.Validation.InputA, res.Validation.InputB, res.Validation.DecodedResult, res.Validation.Success)
	fmt.Printf("  json:         %s\n", res.Artifacts.ResultJSON)
	fmt.Printf("  geth-log:     %s\n", res.Artifacts.GethLog)
	if res.Error != "" {
		fmt.Printf("  error:        %s\n", res.Error)
	}
}

func millis(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / float64(time.Millisecond)
}

func parseTime(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func wrapPluginHint(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "can not compile code") || strings.Contains(msg, "cannot compile code") ||
		strings.Contains(msg, "plugin") || strings.Contains(msg, "buildmode=plugin") {
		return fmt.Errorf("%w (check geth log and Go plugin toolchain; this experiment sets CRYPTOUPGRADE_MODULE to the repo root)", err)
	}
	return err
}
