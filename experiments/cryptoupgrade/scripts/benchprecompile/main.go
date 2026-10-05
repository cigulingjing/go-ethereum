// benchprecompile 对比 EvoCrypt WASM 与客户端 native precompile 的验签时间。
// 该入口按算法分别保存输入向量和汇总结果，避免实验运行目录污染 results。
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench/aigis"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench/dilithium3"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench/mldsa65"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench/slhdsa"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench/groth16bls12381"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench/groth16bn254"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultChainID  = uint64(11223344)
	defaultWarmup   = 10
	defaultSamples  = 50
	defaultGasLimit = uint64(30_000_000)
)

type algorithmSpec struct {
	Name        string
	Kind        string
	Precompile  common.Address
	Verify      func([]byte, []byte, []byte) (bool, error)
	Description string
}

var algorithms = map[string]algorithmSpec{
	aigis.Name:           {Name: aigis.Name, Kind: "pqc", Precompile: common.CryptoUpgradeAigisSig2VerifyAddress, Verify: aigis.Verify, Description: "Aigis-Sig2"},
	dilithium3.Name:      {Name: dilithium3.Name, Kind: "pqc", Precompile: common.CryptoUpgradeDilithium3VerifyAddress, Verify: dilithium3.Verify, Description: "Dilithium3"},
	mldsa65.Name:         {Name: mldsa65.Name, Kind: "pqc", Precompile: common.CryptoUpgradeMlDsa65VerifyAddress, Verify: mldsa65.Verify, Description: "ML-DSA-65"},
	slhdsa.Name:          {Name: slhdsa.Name, Kind: "pqc", Precompile: common.CryptoUpgradeSlhDsaVerifyAddress, Verify: slhdsa.Verify, Description: "SLH-DSA-SHAKE-192f"},
	groth16bls12381.Name: {Name: groth16bls12381.Name, Kind: "groth", Precompile: common.CryptoUpgradeGroth16Bls12381Address, Verify: groth16bls12381.Verify, Description: "Groth16/BLS12-381"},
	groth16bn254.Name:    {Name: groth16bn254.Name, Kind: "groth", Precompile: common.CryptoUpgradeGroth16Bn254Address, Verify: groth16bn254.Verify, Description: "Groth16/BN254"},
}

type timing struct {
	MeanMillis   float64   `json:"meanMillis"`
	P50Millis    float64   `json:"p50Millis"`
	P95Millis    float64   `json:"p95Millis"`
	MinMillis    float64   `json:"minMillis"`
	MaxMillis    float64   `json:"maxMillis"`
	SampleMillis []float64 `json:"sampleMillis"`
}

type uploadResult struct {
	TxHash      string `json:"txHash"`
	BlockNumber uint64 `json:"blockNumber"`
	GasUsed     uint64 `json:"gasUsed"`
	Status      uint64 `json:"status"`
}

type vectorFile struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type pathResult struct {
	ExecutionTiming *timing `json:"executionTiming,omitempty"`
	EthCallTiming   *timing `json:"ethCallTiming,omitempty"`
	Success         bool    `json:"success"`
	OutputMatch     bool    `json:"outputMatch"`
	Error           string  `json:"error,omitempty"`
}

type comparison struct {
	EvoCryptEthCallOverPrecompile float64 `json:"evocryptEthCallOverPrecompile"`
	EvoCryptNodeOverPrecompile    float64 `json:"evocryptNodeOverPrecompile"`
}

type result struct {
	Experiment       string       `json:"experiment"`
	RunID            string       `json:"runId"`
	Algorithm        string       `json:"algorithm"`
	Description      string       `json:"description"`
	RPC              string       `json:"rpc"`
	ChainID          uint64       `json:"chainId"`
	Precompile       string       `json:"precompile"`
	Warmup           int          `json:"warmup"`
	Samples          int          `json:"samples"`
	StartedAt        time.Time    `json:"startedAt"`
	FinishedAt       time.Time    `json:"finishedAt"`
	Vectors          []vectorFile `json:"vectors"`
	Upload           uploadResult `json:"upload"`
	EvoCrypt         pathResult   `json:"evocrypt"`
	PrecompileResult pathResult   `json:"precompileResult"`
	Comparison       comparison   `json:"comparison"`
	ClientVersion    string       `json:"clientVersion"`
	BlockNumber      uint64       `json:"blockNumber"`
	Notes            []string     `json:"notes"`
}

type inputData struct {
	First  []byte
	Second []byte
	Third  []byte
}

type timed struct {
	samples []float64
	err     error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchprecompile: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootFlag := flag.String("root", "", "go-ethereum 仓库根目录")
	rpcURL := flag.String("rpc", "http://127.0.0.1:8545", "HTTP RPC URL")
	keyFile := flag.String("key-file", "", "签名私钥文件")
	stageLog := flag.String("stage-log", "", "EvoCrypt stage_timing.jsonl")
	outputRoot := flag.String("output-root", "", "结果根目录，例如 experiments/cryptoupgrade/results")
	runCode := flag.String("run-code", "", "本地实验代码；每个算法结果写入 <output-root>/<algorithm>/<run-code>")
	algorithmFlag := flag.String("algorithms", "", "逗号分隔的算法名；为空时测试全部六个算法")
	warmup := flag.Int("warmup", defaultWarmup, "预热次数")
	samples := flag.Int("n", defaultSamples, "采样次数")
	chainIDFlag := flag.Uint64("chain-id", defaultChainID, "链 ID")
	flag.Parse()
	if *warmup < 0 || *samples <= 0 {
		return errors.New("warmup must be non-negative and n must be positive")
	}
	if *outputRoot == "" || *stageLog == "" {
		return errors.New("-output-root and -stage-log are required")
	}
	root, err := findRepoRoot(*rootFlag)
	if err != nil {
		return err
	}
	if *runCode == "" {
		*runCode = time.Now().UTC().Format("20060102T150405Z")
	}
	specs, err := selectedAlgorithms(*algorithmFlag)
	if err != nil {
		return err
	}
	keyPath := *keyFile
	if keyPath == "" {
		keyPath = filepath.Join(root, "experiments", "cryptoupgrade", "config", "signer.key")
	}
	keyHex, err := readKey(keyPath)
	if err != nil {
		return err
	}
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)

	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, *rpcURL)
	if err != nil {
		return fmt.Errorf("dial ethclient: %w", err)
	}
	defer client.Close()
	rpcClient, err := rpc.DialContext(ctx, *rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer rpcClient.Close()
	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return err
	}
	clientVersion, _ := rpcString(ctx, rpcClient, "web3_clientVersion")

	for _, spec := range specs {
		if err := runAlgorithm(ctx, root, *rpcURL, *stageLog, *outputRoot, *runCode, *warmup, *samples, *chainIDFlag, key, from, client, rpcClient, codeStorageABI, clientVersion, spec); err != nil {
			return fmt.Errorf("%s: %w", spec.Name, err)
		}
	}
	return nil
}

func runAlgorithm(ctx context.Context, root, rpcURL, stageLog, outputRoot, runCode string, warmup, samples int, chainID uint64, key *ecdsa.PrivateKey, from common.Address, client *ethclient.Client, rpcClient *rpc.Client, codeStorageABI abi.ABI, clientVersion string, spec algorithmSpec) error {
	outputDir := filepath.Join(outputRoot, spec.Name, runCode)
	inputDir := filepath.Join(outputDir, "inputs")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		return err
	}
	data, wasmPath, vectors, err := loadInput(root, spec, inputDir)
	if err != nil {
		return err
	}
	ok, err := spec.Verify(data.First, data.Second, data.Third)
	if err != nil {
		return fmt.Errorf("native vector verification: %w", err)
	}
	if !ok {
		return errors.New("native vector verification returned false")
	}
	packedInput, err := packBytes(data.First, data.Second, data.Third)
	if err != nil {
		return err
	}
	callData, err := codeStorageABI.Pack("callFunc", spec.Name, packedInput)
	if err != nil {
		return err
	}
	precompileData, err := packBytes(data.First, data.Second, data.Third)
	if err != nil {
		return err
	}
	started := time.Now().UTC()
	upload, err := uploadWASM(ctx, client, codeStorageABI, key, from, new(big.Int).SetUint64(chainID), spec.Name, wasmPath)
	if err != nil {
		return fmt.Errorf("upload EvoCrypt WASM: %w", err)
	}
	if err := waitCallable(ctx, rpcClient, from, callData); err != nil {
		return err
	}
	if err := checkBoolCall(ctx, rpcClient, from, common.CodeStorageAddress, callData); err != nil {
		return fmt.Errorf("EvoCrypt validation call: %w", err)
	}
	if err := checkBoolCall(ctx, rpcClient, from, spec.Precompile, precompileData); err != nil {
		return fmt.Errorf("precompile validation call: %w", err)
	}
	evo, err := measureEvoCrypt(ctx, rpcClient, from, callData, stageLog, spec.Name, warmup, samples)
	if err != nil {
		return err
	}
	pre, err := measurePrecompile(ctx, rpcClient, from, spec.Precompile, precompileData, warmup, samples)
	if err != nil {
		return err
	}
	blockNumber, _ := client.BlockNumber(ctx)
	r := result{
		Experiment: filepath.Base(outputDir), RunID: time.Now().UTC().Format("20060102T150405Z"), Algorithm: spec.Name, Description: spec.Description,
		RPC: rpcURL, ChainID: chainID, Precompile: spec.Precompile.Hex(), Warmup: warmup, Samples: samples,
		StartedAt: started, FinishedAt: time.Now().UTC(), Vectors: vectorManifest(inputDir, vectors), Upload: upload,
		EvoCrypt: evo, PrecompileResult: pre, ClientVersion: clientVersion, BlockNumber: blockNumber,
		Notes: []string{
			"EvoCrypt executionTiming is coprocessor_exit.durationNs from the node stage log.",
			"ethCallTiming is the wall-clock interval of the same eth_call RPC path.",
			"Both paths use the same ABI input and require a true verification result.",
		},
	}
	r.Comparison = comparison{EvoCryptEthCallOverPrecompile: evo.EthCallTiming.MeanMillis / pre.EthCallTiming.MeanMillis, EvoCryptNodeOverPrecompile: evo.ExecutionTiming.MeanMillis / pre.EthCallTiming.MeanMillis}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "result.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	summary := renderSummary(r)
	if err := os.WriteFile(filepath.Join(outputDir, "result.txt"), []byte(summary), 0o644); err != nil {
		return err
	}
	fmt.Print(summary)
	return nil
}

func selectedAlgorithms(value string) ([]algorithmSpec, error) {
	if value == "" {
		return []algorithmSpec{algorithms[aigis.Name], algorithms[dilithium3.Name], algorithms[mldsa65.Name], algorithms[slhdsa.Name], algorithms[groth16bls12381.Name], algorithms[groth16bn254.Name]}, nil
	}
	var out []algorithmSpec
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		spec, ok := algorithms[name]
		if !ok {
			return nil, fmt.Errorf("unknown algorithm %q", name)
		}
		out = append(out, spec)
	}
	return out, nil
}

func loadInput(root string, spec algorithmSpec, inputDir string) (inputData, string, []vectorFile, error) {
	if spec.Kind == "pqc" {
		pqcRoot := filepath.Join(root, "experiments", "cryptoupgrade", "algorithm", "pqcgo")
		if err := pqcbench.CheckArtifacts(pqcRoot, spec.Name); err != nil {
			return inputData{}, "", nil, err
		}
		v, err := pqcbench.LoadVectors(pqcRoot, spec.Name)
		if err != nil {
			return inputData{}, "", nil, err
		}
		files := map[string][]byte{"pk.bin": v.PublicKey, "message.bin": v.Message, "signature.bin": v.Signature}
		for name, value := range files {
			if err := os.WriteFile(filepath.Join(inputDir, name), value, 0o644); err != nil {
				return inputData{}, "", nil, err
			}
		}
		wasmPath := pqcbench.WASMPath(pqcRoot, spec.Name)
		wasm, err := os.ReadFile(wasmPath)
		if err != nil {
			return inputData{}, "", nil, err
		}
		if err := os.WriteFile(filepath.Join(inputDir, spec.Name+".wasm"), wasm, 0o644); err != nil {
			return inputData{}, "", nil, err
		}
		return inputData{First: v.PublicKey, Second: v.Message, Third: v.Signature}, wasmPath, []vectorFile{vectorInfo(filepath.Join(inputDir, "pk.bin"), v.PublicKey), vectorInfo(filepath.Join(inputDir, "message.bin"), v.Message), vectorInfo(filepath.Join(inputDir, "signature.bin"), v.Signature)}, nil
	}
	zkRoot := filepath.Join(root, "experiments", "cryptoupgrade", "algorithm", "zkgo")
	if err := zkbench.CheckArtifacts(zkRoot, spec.Name); err != nil {
		return inputData{}, "", nil, err
	}
	v, err := zkbench.LoadVectors(zkRoot, spec.Name)
	if err != nil {
		return inputData{}, "", nil, err
	}
	files := map[string][]byte{"vk.bin": v.VK, "public.bin": v.Public, "proof.bin": v.Proof}
	for name, value := range files {
		if err := os.WriteFile(filepath.Join(inputDir, name), value, 0o644); err != nil {
			return inputData{}, "", nil, err
		}
	}
	wasmPath := zkbench.WASMPath(zkRoot, spec.Name)
	wasm, err := os.ReadFile(wasmPath)
	if err != nil {
		return inputData{}, "", nil, err
	}
	if err := os.WriteFile(filepath.Join(inputDir, spec.Name+".wasm"), wasm, 0o644); err != nil {
		return inputData{}, "", nil, err
	}
	return inputData{First: v.VK, Second: v.Public, Third: v.Proof}, wasmPath, []vectorFile{vectorInfo(filepath.Join(inputDir, "vk.bin"), v.VK), vectorInfo(filepath.Join(inputDir, "public.bin"), v.Public), vectorInfo(filepath.Join(inputDir, "proof.bin"), v.Proof)}, nil
}

func uploadWASM(ctx context.Context, client *ethclient.Client, codeABI abi.ABI, key *ecdsa.PrivateKey, from common.Address, chainID *big.Int, name, wasmPath string) (uploadResult, error) {
	encoded, err := cryptoupgrade.EncodeWasmFile(wasmPath)
	if err != nil {
		return uploadResult{}, err
	}
	data, err := codeABI.Pack("uploadCode", name, encoded, uint64(7), "bytes,bytes,bytes", "bool")
	if err != nil {
		return uploadResult{}, err
	}
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return uploadResult{}, err
	}
	to := common.CodeStorageAddress
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data})
	if err != nil {
		gasLimit = 25_000_000
	} else {
		gasLimit = gasLimit * 12 / 10
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return uploadResult{}, err
	}
	tx := types.NewTx(&types.LegacyTx{Nonce: nonce, To: &to, Gas: gasLimit, GasPrice: gasPrice, Data: data})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return uploadResult{}, err
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return uploadResult{}, err
	}
	receipt, err := waitReceipt(ctx, client, signed.Hash())
	if err != nil {
		return uploadResult{}, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return uploadResult{}, fmt.Errorf("upload reverted, gasUsed=%d", receipt.GasUsed)
	}
	return uploadResult{TxHash: signed.Hash().Hex(), BlockNumber: receipt.BlockNumber.Uint64(), GasUsed: receipt.GasUsed, Status: receipt.Status}, nil
}

func measureEvoCrypt(ctx context.Context, client *rpc.Client, from common.Address, data []byte, stagePath, name string, warmup, n int) (pathResult, error) {
	before, err := fileSize(stagePath)
	if err != nil {
		return pathResult{}, err
	}
	rpcTiming := timeRPCPath(ctx, client, from, common.CodeStorageAddress, data, warmup, n)
	if rpcTiming.err != nil {
		return pathResult{Error: rpcTiming.err.Error()}, rpcTiming.err
	}
	durations, err := readStageDurations(stagePath, before, name)
	if err != nil {
		return pathResult{}, err
	}
	if len(durations) < warmup+n {
		return pathResult{}, fmt.Errorf("EvoCrypt stage samples=%d, want=%d; check stage log", len(durations), warmup+n)
	}
	stage := durations[len(durations)-(warmup+n):][warmup:]
	return pathResult{ExecutionTiming: toTiming(pqcbench.Summarize(stage)), EthCallTiming: toTiming(pqcbench.Summarize(rpcTiming.samples)), Success: true, OutputMatch: true}, nil
}

func measurePrecompile(ctx context.Context, client *rpc.Client, from, to common.Address, data []byte, warmup, n int) (pathResult, error) {
	timed := timeRPCPath(ctx, client, from, to, data, warmup, n)
	if timed.err != nil {
		return pathResult{Error: timed.err.Error()}, timed.err
	}
	return pathResult{EthCallTiming: toTiming(pqcbench.Summarize(timed.samples)), Success: true, OutputMatch: true}, nil
}

func timeRPCPath(ctx context.Context, client *rpc.Client, from, to common.Address, data []byte, warmup, n int) timed {
	call := func() error {
		var output hexutil.Bytes
		args := map[string]interface{}{"from": from, "to": to, "gas": hexutil.Uint64(defaultGasLimit), "data": hexutil.Bytes(data)}
		if err := client.CallContext(ctx, &output, "eth_call", args, "latest"); err != nil {
			return err
		}
		ok, err := unpackBool(output)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("verification returned false")
		}
		return nil
	}
	for i := 0; i < warmup; i++ {
		if err := call(); err != nil {
			return timed{err: fmt.Errorf("warmup %d: %w", i, err)}
		}
	}
	samples := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		if err := call(); err != nil {
			return timed{err: fmt.Errorf("sample %d: %w", i, err)}
		}
		samples = append(samples, float64(time.Since(start).Nanoseconds())/1e6)
	}
	return timed{samples: samples}
}

func waitCallable(ctx context.Context, client *rpc.Client, from common.Address, data []byte) error {
	deadline := time.Now().Add(2 * time.Minute)
	var last error
	for time.Now().Before(deadline) {
		last = checkBoolCall(ctx, client, from, common.CodeStorageAddress, data)
		if last == nil {
			return nil
		}
		if !strings.Contains(last.Error(), "required-version-missing") {
			return last
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for EvoCrypt runtime: %w", last)
}

func checkBoolCall(ctx context.Context, client *rpc.Client, from, to common.Address, data []byte) error {
	var output hexutil.Bytes
	args := map[string]interface{}{"from": from, "to": to, "gas": hexutil.Uint64(defaultGasLimit), "data": hexutil.Bytes(data)}
	if err := client.CallContext(ctx, &output, "eth_call", args, "latest"); err != nil {
		return err
	}
	ok, err := unpackBool(output)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("verification returned false")
	}
	return nil
}

func readStageDurations(path string, offset int64, name string) ([]float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if offset > int64(len(raw)) {
		offset = 0
	}
	var values []float64
	for _, line := range strings.Split(string(raw[offset:]), "\n") {
		var event map[string]interface{}
		if strings.TrimSpace(line) == "" || json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		algorithmName, _ := event["algorithm"].(string)
		if event["stage"] != "coprocessor_exit" || !strings.EqualFold(algorithmName, name) {
			continue
		}
		if success, ok := event["success"].(bool); ok && !success {
			continue
		}
		if ns, ok := event["durationNs"].(float64); ok {
			values = append(values, ns/1e6)
		}
	}
	return values, nil
}

func vectorManifest(inputDir string, files []vectorFile) []vectorFile { return files }

func vectorInfo(path string, data []byte) vectorFile {
	sum := sha256.Sum256(data)
	return vectorFile{Name: filepath.Base(path), Bytes: len(data), SHA256: hex.EncodeToString(sum[:])}
}

func toTiming(t pqcbench.Timing) *timing {
	return &timing{MeanMillis: t.MeanMillis, P50Millis: t.P50Millis, P95Millis: t.P95Millis, MinMillis: t.MinMillis, MaxMillis: t.MaxMillis, SampleMillis: t.SampleMillis}
}

func renderSummary(r result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s EvoCrypt vs precompile (warmup=%d n=%d)\n", r.Description, r.Warmup, r.Samples)
	fmt.Fprintf(&b, "  algorithm: %s precompile=%s\n", r.Algorithm, r.Precompile)
	fmt.Fprintf(&b, "  evocrypt node execution: mean=%.3fms p50=%.3fms p95=%.3fms\n", r.EvoCrypt.ExecutionTiming.MeanMillis, r.EvoCrypt.ExecutionTiming.P50Millis, r.EvoCrypt.ExecutionTiming.P95Millis)
	fmt.Fprintf(&b, "  evocrypt eth_call: mean=%.3fms p50=%.3fms p95=%.3fms\n", r.EvoCrypt.EthCallTiming.MeanMillis, r.EvoCrypt.EthCallTiming.P50Millis, r.EvoCrypt.EthCallTiming.P95Millis)
	fmt.Fprintf(&b, "  precompile eth_call: mean=%.3fms p50=%.3fms p95=%.3fms\n", r.PrecompileResult.EthCallTiming.MeanMillis, r.PrecompileResult.EthCallTiming.P50Millis, r.PrecompileResult.EthCallTiming.P95Millis)
	fmt.Fprintf(&b, "  evocrypt/precompile mean ratio: eth_call=%.2fx nodeExecution=%.2fx\n", r.Comparison.EvoCryptEthCallOverPrecompile, r.Comparison.EvoCryptNodeOverPrecompile)
	fmt.Fprintf(&b, "  upload tx: %s block=%d gasUsed=%d\n", r.Upload.TxHash, r.Upload.BlockNumber, r.Upload.GasUsed)
	return b.String()
}

func packBytes(a, b, c []byte) ([]byte, error) {
	t, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	return abi.Arguments{{Type: t}, {Type: t}, {Type: t}}.Pack(a, b, c)
}

func unpackBool(encoded []byte) (bool, error) {
	t, err := abi.NewType("bool", "", nil)
	if err != nil {
		return false, err
	}
	values, err := abi.Arguments{{Type: t}}.Unpack(encoded)
	if err != nil {
		return false, err
	}
	ok, valid := values[0].(bool)
	if !valid {
		return false, fmt.Errorf("expected bool output, got %T", values[0])
	}
	return ok, nil
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	deadline, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		receipt, err := client.TransactionReceipt(deadline, hash)
		if err == nil && receipt != nil {
			return receipt, nil
		}
		select {
		case <-deadline.Done():
			return nil, deadline.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func readKey(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(strings.TrimPrefix(string(raw), "0x"))
	if len(key) != 64 {
		return "", fmt.Errorf("invalid private key length in %s", path)
	}
	return key, nil
}

func rpcString(ctx context.Context, client *rpc.Client, method string) (string, error) {
	var value string
	if err := client.CallContext(ctx, &value, method); err != nil {
		return "", err
	}
	return value, nil
}

func findRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot locate go-ethereum root")
		}
	}
}
