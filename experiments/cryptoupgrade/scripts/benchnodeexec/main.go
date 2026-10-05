// 单节点上对比同一验签输入的原生 CGO 与 EvoCrypt 执行时间。
// 原生时间只包含 CGO verify。EvoCrypt 时间取节点 stage log 里 coprocessor_exit 的 durationNs，
// 因此不含 RPC 往返。上传、激活和预热不计入均值。
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
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

const experimentName = "node-exec-efficiency"

type kind int

const (
	kindPQC kind = iota
	kindGroth
)

type timingJSON struct {
	MeanMillis   float64   `json:"meanMillis"`
	P50Millis    float64   `json:"p50Millis"`
	P95Millis    float64   `json:"p95Millis"`
	MinMillis    float64   `json:"minMillis"`
	MaxMillis    float64   `json:"maxMillis"`
	SampleMillis []float64 `json:"sampleMillis"`
}

type algorithmResult struct {
	Algorithm      string      `json:"algorithm"`
	Native         *timingJSON `json:"native,omitempty"`
	EvoCrypt       *timingJSON `json:"evocrypt,omitempty"`
	CallMeanMillis float64     `json:"ethCallMeanMillis,omitempty"`
	OutputMatched  bool        `json:"outputMatched"`
	Error          string      `json:"error,omitempty"`
}

type runResult struct {
	Experiment string            `json:"experiment"`
	RunID      string            `json:"runId"`
	RPC        string            `json:"rpc"`
	Warmup     int               `json:"warmup"`
	Samples    int               `json:"samples"`
	Algorithms []algorithmResult `json:"algorithms"`
	Notes      []string          `json:"notes"`
}

type spec struct {
	name string
	kind kind
	root string
}

type timed struct {
	samples []float64
	err     error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchnodeexec: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	rootFlag := flag.String("root", "", "go-ethereum 仓库根目录")
	rpcURL := flag.String("rpc", "http://127.0.0.1:8761", "单节点 HTTP RPC")
	keyFile := flag.String("key-file", "", "签名私钥文件")
	stageLog := flag.String("stage-log", "", "节点 stage_timing.jsonl")
	warmup := flag.Int("warmup", 10, "预热次数")
	samples := flag.Int("n", 100, "采样次数")
	algorithms := flag.String("algorithms", "", "逗号分隔算法名，默认六个全部测量")
	outputDir := flag.String("output", "", "结果目录")
	chainIDFlag := flag.Uint64("chain-id", 11223344, "链 ID")
	flag.Parse()

	repoRoot, err := findRepoRoot(*rootFlag)
	if err != nil {
		return err
	}
	keyPath := *keyFile
	if keyPath == "" {
		keyPath = filepath.Join(repoRoot, "experiments", "cryptoupgrade", "config", "signer.key")
	}
	logPath := *stageLog
	if logPath == "" {
		logPath = filepath.Join(repoRoot, "experiments", "cryptoupgrade", "output", "network", "local-1node", "node1", "plugin", "stage_timing.jsonl")
	}
	runID := time.Now().Format("20060102-150405")
	outDir := *outputDir
	if outDir == "" {
		outDir = filepath.Join(repoRoot, "experiments", "cryptoupgrade", "output", "results", experimentName, runID)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
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

	codeStorageABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		return fmt.Errorf("parse CodeStorage ABI: %w", err)
	}

	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, *rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()
	rpcClient, err := rpc.DialContext(ctx, *rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer rpcClient.Close()

	pqcRoot := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "pqcgo")
	zkRoot := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "zkgo")
	specs := selectSpecs(splitCSV(*algorithms), []spec{
		{name: aigis.Name, kind: kindPQC, root: pqcRoot},
		{name: dilithium3.Name, kind: kindPQC, root: pqcRoot},
		{name: mldsa65.Name, kind: kindPQC, root: pqcRoot},
		{name: slhdsa.Name, kind: kindPQC, root: pqcRoot},
		{name: groth16bls12381.Name, kind: kindGroth, root: zkRoot},
		{name: groth16bn254.Name, kind: kindGroth, root: zkRoot},
	})
	if len(specs) == 0 {
		return fmt.Errorf("no algorithms selected")
	}

	result := runResult{
		Experiment: experimentName,
		RunID:      runID,
		RPC:        *rpcURL,
		Warmup:     *warmup,
		Samples:    *samples,
		Notes: []string{
			"native 是与 WASM 同源的 CGO verify，只计函数调用",
			"evocrypt 是单节点 coprocessor_exit.durationNs，不含 RPC",
			"ethCallMeanMillis 是 eth_call 往返，仅作对照，不作为执行时间",
		},
	}

	failed := false
	chainID := new(big.Int).SetUint64(*chainIDFlag)
	for _, item := range specs {
		fmt.Printf("measure %s\n", item.name)
		one := measure(ctx, client, rpcClient, codeStorageABI, key, from, chainID, logPath, item, *warmup, *samples)
		result.Algorithms = append(result.Algorithms, one)
		if one.Error != "" {
			failed = true
			fmt.Printf("fail %s: %s\n", item.name, one.Error)
			continue
		}
		fmt.Printf("%s  native=%.3fms  evocrypt=%.3fms\n", item.name, one.Native.MeanMillis, one.EvoCrypt.MeanMillis)
	}

	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(outDir, "result.json")
	if err := os.WriteFile(jsonPath, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	summary := renderSummary(result, jsonPath)
	if err := os.WriteFile(filepath.Join(outDir, "result.txt"), []byte(summary), 0o644); err != nil {
		return err
	}
	fmt.Print(summary)
	if failed {
		return fmt.Errorf("one or more algorithms failed")
	}
	return nil
}

func measure(ctx context.Context, client *ethclient.Client, rpcClient *rpc.Client, codeStorageABI abi.ABI, key *ecdsa.PrivateKey, from common.Address, chainID *big.Int, stageLog string, item spec, warmup, n int) algorithmResult {
	out := algorithmResult{Algorithm: item.name}
	nativeFn, input, err := loadNative(item)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	nativeOK, err := nativeFn()
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if !nativeOK {
		out.Error = "native verify returned false"
		return out
	}
	nativeTimed := timePath(warmup, n, func() error {
		ok, err := nativeFn()
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("native verify returned false")
		}
		return nil
	})
	if nativeTimed.err != nil {
		out.Error = nativeTimed.err.Error()
		return out
	}

	wasmPath := filepath.Join(item.root, item.name, item.name+".wasm")
	if err := uploadWASM(ctx, client, codeStorageABI, key, from, chainID, item.name, wasmPath); err != nil {
		out.Error = err.Error()
		return out
	}
	callData, err := codeStorageABI.Pack("callFunc", item.name, input)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	// 链上版本号会先于本地 WASM 编译完成。大模块需要等到运行时加载后再采样。
	if err := waitCallable(ctx, rpcClient, codeStorageABI, from, callData); err != nil {
		out.Error = err.Error()
		return out
	}

	offset, err := fileSize(stageLog)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	callSamples := make([]float64, 0, n)
	for i := 0; i < warmup+n; i++ {
		start := time.Now()
		if err := checkCall(ctx, rpcClient, codeStorageABI, from, callData); err != nil {
			out.Error = fmt.Errorf("eth_call %d: %w", i, err).Error()
			return out
		}
		if i >= warmup {
			callSamples = append(callSamples, float64(time.Since(start).Nanoseconds())/1e6)
		}
	}
	durations, err := readCoprocessorDurations(stageLog, offset, capitalName(item.name))
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if len(durations) < warmup+n {
		out.Error = fmt.Sprintf("stage log samples %d, want at least %d", len(durations), warmup+n)
		return out
	}
	measured := durations[len(durations)-(warmup+n):]
	evocryptSamples := measured[warmup:]

	nativeJSON := toTiming(pqcbench.Summarize(nativeTimed.samples))
	evocryptJSON := toTiming(pqcbench.Summarize(evocryptSamples))
	callJSON := pqcbench.Summarize(callSamples)
	out.Native = &nativeJSON
	out.EvoCrypt = &evocryptJSON
	out.CallMeanMillis = callJSON.MeanMillis
	out.OutputMatched = true
	return out
}

func loadNative(item spec) (func() (bool, error), []byte, error) {
	switch item.kind {
	case kindPQC:
		if err := pqcbench.CheckArtifacts(item.root, item.name); err != nil {
			return nil, nil, err
		}
		vectors, err := pqcbench.LoadVectors(item.root, item.name)
		if err != nil {
			return nil, nil, err
		}
		input, err := packBytes(vectors.PublicKey, vectors.Message, vectors.Signature)
		if err != nil {
			return nil, nil, err
		}
		verify, ok := pqcVerifier(item.name)
		if !ok {
			return nil, nil, fmt.Errorf("unknown pqc algorithm %s", item.name)
		}
		pk, message, signature := vectors.PublicKey, vectors.Message, vectors.Signature
		return func() (bool, error) { return verify(pk, message, signature) }, input, nil
	case kindGroth:
		if err := zkbench.CheckArtifacts(item.root, item.name); err != nil {
			return nil, nil, err
		}
		vectors, err := zkbench.LoadVectors(item.root, item.name)
		if err != nil {
			return nil, nil, err
		}
		input, err := packBytes(vectors.VK, vectors.Public, vectors.Proof)
		if err != nil {
			return nil, nil, err
		}
		verify, ok := grothVerifier(item.name)
		if !ok {
			return nil, nil, fmt.Errorf("unknown groth algorithm %s", item.name)
		}
		vk, public, proof := vectors.VK, vectors.Public, vectors.Proof
		return func() (bool, error) { return verify(vk, public, proof) }, input, nil
	default:
		return nil, nil, fmt.Errorf("unknown kind")
	}
}

func pqcVerifier(name string) (func(pk, message, signature []byte) (bool, error), bool) {
	switch name {
	case aigis.Name:
		return aigis.Verify, aigis.Enabled()
	case dilithium3.Name:
		return dilithium3.Verify, dilithium3.Enabled()
	case mldsa65.Name:
		return mldsa65.Verify, mldsa65.Enabled()
	case slhdsa.Name:
		return slhdsa.Verify, slhdsa.Enabled()
	default:
		return nil, false
	}
}

func grothVerifier(name string) (func(vk, public, proof []byte) (bool, error), bool) {
	switch name {
	case groth16bls12381.Name:
		return groth16bls12381.Verify, groth16bls12381.Enabled()
	case groth16bn254.Name:
		return groth16bn254.Verify, groth16bn254.Enabled()
	default:
		return nil, false
	}
}

func uploadWASM(ctx context.Context, client *ethclient.Client, codeStorageABI abi.ABI, key *ecdsa.PrivateKey, from common.Address, chainID *big.Int, name, wasmPath string) error {
	encoded, err := encodeWasmFile(wasmPath)
	if err != nil {
		return fmt.Errorf("encode wasm: %w", err)
	}
	data, err := codeStorageABI.Pack("uploadCode", name, encoded, uint64(7), "bytes,bytes,bytes", "bool")
	if err != nil {
		return fmt.Errorf("pack uploadCode: %w", err)
	}
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return fmt.Errorf("pending nonce: %w", err)
	}
	to := common.CodeStorageAddress
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to, Data: data})
	if err != nil {
		gasLimit = 25_000_000
	} else {
		gasLimit = gasLimit * 12 / 10
		if gasLimit < 8_000_000 {
			gasLimit = 8_000_000
		}
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("suggest gas price: %w", err)
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return fmt.Errorf("sign tx: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return fmt.Errorf("send upload: %w", err)
	}
	receipt, err := waitReceipt(ctx, client, signed.Hash())
	if err != nil {
		return fmt.Errorf("wait upload receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("upload reverted, gasUsed=%d", receipt.GasUsed)
	}
	if err := waitActive(ctx, client, codeStorageABI, name); err != nil {
		return err
	}
	fmt.Printf("uploaded %s block=%d gasUsed=%d\n", name, receipt.BlockNumber.Uint64(), receipt.GasUsed)
	return nil
}

func waitCallable(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, data []byte) error {
	deadline := time.Now().Add(2 * time.Minute)
	var last error
	for time.Now().Before(deadline) {
		last = checkCall(ctx, client, codeStorageABI, from, data)
		if last == nil {
			return nil
		}
		if !strings.Contains(last.Error(), "required-version-missing") {
			return last
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for runtime load: %w", last)
}

func selectSpecs(names []string, all []spec) []spec {
	if len(names) == 0 {
		return all
	}
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	var selected []spec
	for _, item := range all {
		if wanted[item.name] {
			selected = append(selected, item)
		}
	}
	return selected
}

func splitCSV(raw string) []string {
	var names []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			names = append(names, part)
		}
	}
	return names
}

func checkCall(ctx context.Context, client *rpc.Client, codeStorageABI abi.ABI, from common.Address, data []byte) error {
	var output hexutil.Bytes
	gas := hexutil.Uint64(30_000_000)
	args := map[string]interface{}{
		"from": from,
		"to":   common.CodeStorageAddress,
		"gas":  gas,
		"data": hexutil.Bytes(data),
	}
	if err := client.CallContext(ctx, &output, "eth_call", args, "latest"); err != nil {
		return err
	}
	// CodeStorage.callFunc 直接返回 WASM 输出，不再包一层 ABI bytes。
	okValue, err := unpackBool(output)
	if err != nil {
		return err
	}
	if !okValue {
		return fmt.Errorf("evocrypt verify returned false")
	}
	return nil
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		receipt, err := client.TransactionReceipt(waitCtx, hash)
		if err == nil && receipt != nil {
			return receipt, nil
		}
		lastErr = err
		select {
		case <-waitCtx.Done():
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, waitCtx.Err()
		case <-ticker.C:
		}
	}
}

func waitActive(ctx context.Context, client *ethclient.Client, codeStorageABI abi.ABI, name string) error {
	data, err := codeStorageABI.Pack("getActiveVersion", name)
	if err != nil {
		return err
	}
	to := common.CodeStorageAddress
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		out, err := client.CallContract(ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
		if err == nil && len(out) > 0 {
			values, unpackErr := codeStorageABI.Unpack("getActiveVersion", out)
			if unpackErr == nil && len(values) > 0 {
				if active, ok := values[0].(uint64); ok && active >= 1 {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for %s activation", name)
}

func readCoprocessorDurations(path string, offset int64, algorithm string) ([]float64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read stage log: %w", err)
	}
	if int64(len(raw)) < offset {
		offset = 0
	}
	var samples []float64
	for _, line := range strings.Split(string(raw[offset:]), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event["stage"] != "coprocessor_exit" {
			continue
		}
		if event["algorithm"] != algorithm {
			continue
		}
		if success, ok := event["success"].(bool); ok && !success {
			continue
		}
		ns, ok := asFloat(event["durationNs"])
		if !ok {
			continue
		}
		samples = append(samples, ns/1e6)
	}
	return samples, nil
}

func asFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func capitalName(name string) string {
	if name == "" {
		return name
	}
	bytes := []byte(name)
	if bytes[0] >= 'a' && bytes[0] <= 'z' {
		bytes[0] -= 'a' - 'A'
	}
	return string(bytes)
}

func encodeWasmFile(path string) (string, error) {
	wasm, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read wasm file %s: %w", path, err)
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(wasm); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("compress wasm bytecode: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finish wasm compression: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func packBytes(a, b, c []byte) ([]byte, error) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}
	return args.Pack(a, b, c)
}

func unpackBool(encoded []byte) (bool, error) {
	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return false, err
	}
	values, err := abi.Arguments{{Type: boolType}}.Unpack(encoded)
	if err != nil {
		return false, err
	}
	ok, okType := values[0].(bool)
	if !okType {
		return false, fmt.Errorf("expected bool output, got %T", values[0])
	}
	return ok, nil
}

func timePath(warmup, n int, fn func() error) timed {
	for i := 0; i < warmup; i++ {
		if err := fn(); err != nil {
			return timed{err: fmt.Errorf("warmup: %w", err)}
		}
	}
	samples := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		if err := fn(); err != nil {
			return timed{err: fmt.Errorf("sample %d: %w", i, err)}
		}
		samples = append(samples, float64(time.Since(start).Nanoseconds())/1e6)
	}
	return timed{samples: samples}
}

func toTiming(t pqcbench.Timing) timingJSON {
	return timingJSON{
		MeanMillis:   t.MeanMillis,
		P50Millis:    t.P50Millis,
		P95Millis:    t.P95Millis,
		MinMillis:    t.MinMillis,
		MaxMillis:    t.MaxMillis,
		SampleMillis: t.SampleMillis,
	}
}

func fileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

func readKey(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read key file: %w", err)
	}
	hexKey := strings.TrimSpace(strings.TrimPrefix(string(raw), "0x"))
	if len(hexKey) != 64 {
		return "", fmt.Errorf("key file %s is not a 32-byte hex secret", path)
	}
	return hexKey, nil
}

func findRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find go.mod from %s", dir)
		}
		dir = parent
	}
}

func renderSummary(result runResult, jsonPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "experiment=%s run=%s rpc=%s warmup=%d samples=%d\n", result.Experiment, result.RunID, result.RPC, result.Warmup, result.Samples)
	fmt.Fprintf(&b, "%-32s %14s %16s\n", "algorithm", "native", "evocrypt")
	for _, item := range result.Algorithms {
		if item.Error != "" || item.Native == nil || item.EvoCrypt == nil {
			fmt.Fprintf(&b, "%-32s %s\n", item.Algorithm, item.Error)
			continue
		}
		fmt.Fprintf(&b, "%-32s %14s %16s\n", item.Algorithm, formatMillis(item.Native.MeanMillis), formatMillis(item.EvoCrypt.MeanMillis))
	}
	fmt.Fprintf(&b, "json=%s\n", jsonPath)
	return b.String()
}

func formatMillis(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.1fµs", ms*1000)
	}
	return fmt.Sprintf("%.3fms", ms)
}
