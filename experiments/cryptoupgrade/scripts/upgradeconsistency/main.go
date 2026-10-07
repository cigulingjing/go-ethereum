// upgradeconsistency runs one v1 -> v2 activation consistency experiment.
package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/ethclient"
)

type contractArtifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

type observation struct {
	NodeID          string `json:"node_id"`
	RPC             string `json:"rpc"`
	TxHash          string `json:"tx_hash"`
	NodeError       string `json:"node_error,omitempty"`
	BlockNumber     uint64 `json:"block_number,omitempty"`
	BlockHash       string `json:"block_hash,omitempty"`
	SelectedVersion uint64 `json:"selected_version,omitempty"`
	ModuleHash      string `json:"module_hash,omitempty"`
	Output          string `json:"output,omitempty"`
	GasUsed         uint64 `json:"gas_used,omitempty"`
	ReceiptStatus   uint64 `json:"receipt_status,omitempty"`
	StateRoot       string `json:"state_root,omitempty"`
	LocalStateRoot  string `json:"local_state_root,omitempty"`
	Pending         bool   `json:"pending"`
}

type runResult struct {
	Algorithm       string        `json:"algorithm"`
	V1Hash          string        `json:"v1_module_hash"`
	V2Hash          string        `json:"v2_module_hash"`
	ActivationBlock uint64        `json:"activation_block"`
	UpgradeTxHash   string        `json:"upgrade_tx_hash"`
	Contract        string        `json:"contract"`
	CallTxHashes    []string      `json:"call_tx_hashes"`
	Observations    []observation `json:"observations"`
	StartedAt       string        `json:"started_at"`
	FinishedAt      string        `json:"finished_at"`
}

func main() {
	var (
		rpc           = flag.String("rpc", "http://127.0.0.1:8761", "node1 RPC")
		upgradeRPC    = flag.String("upgrade-rpc", "", "RPC used to submit the v2 upgrade transaction; defaults to -rpc")
		rpcBase       = flag.Int("rpc-base", 8761, "host RPC port for node1")
		nodes         = flag.Int("nodes", 20, "node count")
		chainID       = flag.Uint64("chain-id", 11223344, "chain ID")
		keyHex        = flag.String("key", "", "signer private key")
		v1Path        = flag.String("v1", "", "v1 WASM path")
		v2Path        = flag.String("v2", "", "v2 WASM path")
		artifactPath  = flag.String("artifact", "", "compiled Solidity artifact")
		pluginRoot    = flag.String("plugin-root", "", "rendered network root containing node plugin dirs")
		out           = flag.String("out", "", "raw result JSON")
		activationGap = flag.Uint64("activation-gap", 100, "blocks between v2 receipt and H")
		releaseFile   = flag.String("release-file", "", "host path for node20 preparation barrier")
		callInterval  = flag.Duration("call-interval", 500*time.Millisecond, "interval between real probe transactions")
		logWait       = flag.Duration("log-wait", 3*time.Minute, "maximum time to wait for node stage logs")
		wait          = flag.Duration("wait", 3*time.Minute, "experiment timeout")
		upgradeAt     = flag.Uint64("upgrade-submit-block", 0, "node1 block height to reach before submitting the v2 upgrade transaction")
		activationAt  = flag.Uint64("activation-block", 0, "explicit activation block for v2; defaults to current head plus -activation-gap")
		preparedNodes = flag.Int("initial-prepared-nodes", 20, "number of nodes that must prepare v1 before the scenario starts")
		composeDir    = flag.String("compose-dir", "", "docker compose directory for optional staged node startup")
		midStart      = flag.Uint64("mid-start-block", 0, "node1 block height at which to start -mid-services")
		midServices   = flag.String("mid-services", "", "comma-separated docker compose services to start at -mid-start-block")
		lateStart     = flag.Uint64("late-start-block", 0, "node1 block height at which to start -late-services")
		lateServices  = flag.String("late-services", "", "comma-separated docker compose services to start at -late-start-block")
	)
	flag.Parse()
	// 保留旧参数以兼容已有启动命令；采集阶段不再访问节点 RPC。
	_ = rpcBase
	for name, value := range map[string]string{"-key": *keyHex, "-v1": *v1Path, "-v2": *v2Path, "-artifact": *artifactPath, "-plugin-root": *pluginRoot, "-out": *out} {
		if strings.TrimSpace(value) == "" {
			fatal(errors.New(name + " is required"))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *wait)
	defer cancel()
	key, err := crypto.HexToECDSA(strings.TrimPrefix(strings.TrimSpace(*keyHex), "0x"))
	if err != nil {
		fatal(err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)
	client, err := ethclient.DialContext(ctx, *rpc)
	if err != nil {
		fatal(err)
	}
	defer client.Close()
	upgradeURL := strings.TrimSpace(*upgradeRPC)
	if upgradeURL == "" {
		upgradeURL = *rpc
	}
	upgradeClient, err := ethclient.DialContext(ctx, upgradeURL)
	if err != nil {
		fatal(err)
	}
	defer upgradeClient.Close()
	codeABI := cryptoupgrade.CodeStorageABI
	v1Encoded := encodeWASM(*v1Path)
	v2Encoded := encodeWASM(*v2Path)
	v1Tx := send(ctx, client, key, *chainID, &common.CodeStorageAddress, mustPack(codeABI, "uploadCode", "UpgradeConsistencyProbe", v1Encoded, uint64(7), "uint256,uint256", "uint256"), 8_000_000)
	if waitExecutionLog(*pluginRoot, 1, v1Tx.Hash(), ctx).ReceiptStatus != types.ReceiptStatusSuccessful {
		fatal(errors.New("v1 upload reverted"))
	}
	waitPrepared(*pluginRoot, 1, *preparedNodes, ctx)
	artifact := readArtifact(*artifactPath)
	probeABI, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		fatal(err)
	}
	creation, err := hex.DecodeString(strings.TrimPrefix(artifact.Bytecode.Object, "0x"))
	if err != nil {
		fatal(err)
	}
	deployTx := send(ctx, client, key, *chainID, nil, creation, 5_000_000)
	if waitExecutionLog(*pluginRoot, 1, deployTx.Hash(), ctx).ReceiptStatus != types.ReceiptStatusSuccessful {
		fatal(errors.New("probe deployment reverted"))
	}
	contract := crypto.CreateAddress(from, deployTx.Nonce())
	node1StageLog := filepath.Join(*pluginRoot, "node1", "plugin", "stage_timing.jsonl")
	if *upgradeAt > 0 {
		waitLoggedBlock(node1StageLog, *upgradeAt, ctx)
	}
	head := maxLoggedBlock(node1StageLog)
	activation := *activationAt
	if activation == 0 {
		activation = head + *activationGap
	}
	if activation <= head {
		fatal(fmt.Errorf("activation block %d must be greater than current node1 block %d", activation, head))
	}
	result := runResult{Algorithm: "UpgradeConsistencyProbe", V1Hash: keccakFile(*v1Path), V2Hash: keccakFile(*v2Path), ActivationBlock: activation, Contract: contract.Hex(), StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	ticker := time.NewTicker(*callInterval)
	defer ticker.Stop()
	upgradeSubmitted := false
	midStarted := false
	lateStarted := false
	var upgradeTx *types.Transaction
	for {
		head = maxLoggedBlock(node1StageLog)
		if !midStarted && *midStart > 0 && head >= *midStart {
			composeUp(ctx, *composeDir, *midServices)
			midStarted = true
		}
		if !lateStarted && *lateStart > 0 && head >= *lateStart {
			composeUp(ctx, *composeDir, *lateServices)
			lateStarted = true
		}
		if !upgradeSubmitted && (*upgradeAt == 0 || head >= *upgradeAt) {
			upgradeTx = send(ctx, upgradeClient, key, *chainID, &common.CodeStorageAddress, mustPack(codeABI, "uploadCodeVersion", "UpgradeConsistencyProbe", uint64(2), v2Encoded, uint64(7), "uint256,uint256", "uint256", activation), 8_000_000)
			result.UpgradeTxHash = upgradeTx.Hash().Hex()
			upgradeSubmitted = true
		}
		if head > activation+3 && upgradeSubmitted {
			break
		}
		callData, err := probeABI.Pack("probe", big.NewInt(17), big.NewInt(29))
		if err != nil {
			fatal(err)
		}
		tx := send(ctx, client, key, *chainID, &contract, callData, 500_000)
		result.CallTxHashes = append(result.CallTxHashes, tx.Hash().Hex())
		select {
		case <-ctx.Done():
			fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	if !upgradeSubmitted {
		fatal(errors.New("v2 upgrade was not submitted"))
	}
	if waitExecutionLog(*pluginRoot, 1, upgradeTx.Hash(), ctx).ReceiptStatus != types.ReceiptStatusSuccessful {
		fatal(errors.New("v2 upload reverted"))
	}
	if *releaseFile != "" {
		_ = os.WriteFile(*releaseFile, []byte("released\n"), 0644)
	}
	result.Observations = collectLogs(ctx, *nodes, result.CallTxHashes, activation, *pluginRoot, *logWait)
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	data, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
		fatal(err)
	}
	fmt.Printf("activationBlock=%d calls=%d output=%s\n", activation, len(result.CallTxHashes), *out)
}

func send(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, chainID uint64, to *common.Address, data []byte, gas uint64) *types.Transaction {
	nonce, err := client.PendingNonceAt(ctx, crypto.PubkeyToAddress(key.PublicKey))
	if err != nil {
		fatal(err)
	}
	price, err := client.SuggestGasPrice(ctx)
	if err != nil {
		fatal(err)
	}
	tx := types.NewTx(&types.LegacyTx{Nonce: nonce, To: to, Gas: gas, GasPrice: price, Data: data})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(new(big.Int).SetUint64(chainID)), key)
	if err != nil {
		fatal(err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		fatal(err)
	}
	return signed
}

func waitExecutionLog(root string, nodeID int, hash common.Hash, ctx context.Context) stageEvent {
	for {
		path := filepath.Join(root, fmt.Sprintf("node%d", nodeID), "plugin", "stage_timing.jsonl")
		for _, event := range readStageEvents(path) {
			if event.Stage == "transaction_executed" && strings.EqualFold(event.TxHash, hash.Hex()) {
				return event
			}
		}
		select {
		case <-ctx.Done():
			fatal(ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitLoggedBlock(path string, target uint64, ctx context.Context) {
	for {
		if maxLoggedBlock(path) >= target {
			return
		}
		select {
		case <-ctx.Done():
			fatal(ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func mustPack(a abi.ABI, method string, args ...interface{}) []byte {
	data, err := a.Pack(method, args...)
	if err != nil {
		fatal(err)
	}
	return data
}
func encodeWASM(path string) string {
	encoded, err := cryptoupgrade.EncodeWasmFile(path)
	if err != nil {
		fatal(err)
	}
	return encoded
}
func keccakFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	return crypto.Keccak256Hash(raw).Hex()
}
func readArtifact(path string) contractArtifact {
	raw, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	var a contractArtifact
	if err := json.Unmarshal(raw, &a); err != nil {
		fatal(err)
	}
	return a
}
func waitPrepared(root string, version uint64, count int, ctx context.Context) {
	for {
		ready := true
		for i := 1; i <= count; i++ {
			raw, _ := os.ReadFile(filepath.Join(root, fmt.Sprintf("node%d", i), "plugin", "algorithm_versions.json"))
			if !strings.Contains(string(raw), fmt.Sprintf(`"%d"`, version)) {
				ready = false
				break
			}
		}
		if ready {
			return
		}
		select {
		case <-ctx.Done():
			fatal(ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func composeUp(ctx context.Context, dir, services string) {
	serviceList := splitCSV(services)
	if strings.TrimSpace(dir) == "" || len(serviceList) == 0 {
		return
	}
	args := append([]string{"compose", "up", "-d"}, serviceList...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(fmt.Errorf("docker compose up %s: %w", strings.Join(serviceList, ","), err))
	}
}

func splitCSV(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func collectLogs(ctx context.Context, count int, hashes []string, activation uint64, root string, wait time.Duration) []observation {
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if logCoverage(root, count, hashes) {
			break
		}
		select {
		case <-ctx.Done():
			return collectLogSnapshot(count, hashes, activation, root)
		case <-time.After(250 * time.Millisecond):
		}
	}
	return collectLogSnapshot(count, hashes, activation, root)
}

func collectLogSnapshot(count int, hashes []string, activation uint64, root string) []observation {
	out := make([]observation, 0, count*len(hashes))
	for i := 1; i <= count; i++ {
		node := fmt.Sprintf("node%d", i)
		logPath := filepath.Join(root, node, "plugin", "stage_timing.jsonl")
		for _, rawHash := range hashes {
			out = append(out, observeLog(node, logPath, common.HexToHash(rawHash), activation, root, i))
		}
	}
	return out
}

type stageEvent struct {
	Stage         string `json:"stage"`
	TxHash        string `json:"txHash"`
	BlockHash     string `json:"blockHash"`
	BlockNumber   uint64 `json:"blockNumber"`
	Version       uint64 `json:"version"`
	Output        string `json:"output"`
	StateRoot     string `json:"stateRoot"`
	GasUsed       uint64 `json:"gasUsed"`
	ReceiptStatus uint64 `json:"receiptStatus"`
}

type versionMetadata map[string]map[string]struct {
	WasmHash string `json:"wasmHash"`
}

func observeLog(node, logPath string, hash common.Hash, activation uint64, root string, nodeID int) observation {
	o := observation{NodeID: node, TxHash: hash.Hex(), Pending: true}
	events := readStageEvents(logPath)
	for _, event := range events {
		if strings.EqualFold(event.TxHash, hash.Hex()) && event.Stage == "transaction_included" {
			o.BlockNumber = event.BlockNumber
			o.BlockHash = event.BlockHash
		}
		if strings.EqualFold(event.TxHash, hash.Hex()) && event.Stage == "transaction_executed" {
			o.Pending = false
			o.BlockNumber = event.BlockNumber
			o.BlockHash = event.BlockHash
			o.GasUsed = event.GasUsed
			o.ReceiptStatus = event.ReceiptStatus
		}
		if event.Stage == "local_state_root_computed" && strings.EqualFold(event.BlockHash, o.BlockHash) {
			o.LocalStateRoot = event.StateRoot
			continue
		}
		if strings.EqualFold(event.TxHash, hash.Hex()) {
			if event.Stage == "coprocessor_enter" || event.Stage == "coprocessor_exit" {
				o.SelectedVersion = event.Version
				if event.Stage == "coprocessor_exit" {
					o.Output = event.Output
				}
			}
		}
	}
	metadata := versionMetadata{}
	if raw, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("node%d", nodeID), "plugin", "algorithm_versions.json")); err == nil {
		_ = json.Unmarshal(raw, &metadata)
	}
	for version, byVersion := range metadata {
		for _, info := range byVersion {
			parsedVersion, _ := strconv.ParseUint(version, 10, 64)
			if parsedVersion == o.SelectedVersion {
				o.ModuleHash = info.WasmHash
			}
		}
	}
	if o.Pending {
		if o.BlockNumber > 0 {
			o.NodeError = "transaction included but local execution log is pending"
		} else {
			o.NodeError = "transaction not present in local stage log"
		}
	} else if o.SelectedVersion == 0 && o.BlockNumber >= activation {
		o.NodeError = "execution log exists but local coprocessor trace is missing"
	}
	return o
}

func logCoverage(root string, count int, hashes []string) bool {
	for i := 1; i <= count; i++ {
		events := readStageEvents(filepath.Join(root, fmt.Sprintf("node%d", i), "plugin", "stage_timing.jsonl"))
		executed := make(map[string]string)
		roots := make(map[string]bool)
		for _, event := range events {
			if event.Stage == "transaction_executed" {
				executed[strings.ToLower(event.TxHash)] = strings.ToLower(event.BlockHash)
			}
			if event.Stage == "local_state_root_computed" {
				roots[strings.ToLower(event.BlockHash)] = true
			}
		}
		for _, hash := range hashes {
			blockHash, ok := executed[strings.ToLower(hash)]
			if !ok || blockHash == "" || !roots[blockHash] {
				return false
			}
		}
	}
	return true
}

func maxLoggedBlock(path string) uint64 {
	var max uint64
	for _, event := range readStageEvents(path) {
		if event.BlockNumber > max {
			max = event.BlockNumber
		}
	}
	return max
}

func readStageEvents(path string) []stageEvent {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var events []stageEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event stageEvent
		if json.Unmarshal(scanner.Bytes(), &event) == nil {
			events = append(events, event)
		}
	}
	return events
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
