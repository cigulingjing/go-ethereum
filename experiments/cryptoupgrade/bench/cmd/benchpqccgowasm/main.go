// CGO 原生 verify 与 EvoCrypt WASM Execute 的进程内对照。
// 只读 experiments/cryptoupgrade/algorithm/pqcgo/<algorithm>/，不注册生产预编译。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench/aigis"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench/dilithium3"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench/mldsa65"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/pqcbench/slhdsa"
)

const experimentName = "pqc-cgo-wasm"

type verifier struct {
	name    string
	enabled func() bool
	verify  func(pk, message, signature []byte) (bool, error)
}

type timingJSON struct {
	MeanMillis   float64   `json:"meanMillis"`
	P50Millis    float64   `json:"p50Millis"`
	P95Millis    float64   `json:"p95Millis"`
	MinMillis    float64   `json:"minMillis"`
	MaxMillis    float64   `json:"maxMillis"`
	SampleMillis []float64 `json:"sampleMillis"`
}

type algorithmResult struct {
	Algorithm          string      `json:"algorithm"`
	Skipped            bool        `json:"skipped"`
	SkipReason         string      `json:"skipReason,omitempty"`
	Error              string      `json:"error,omitempty"`
	Warmup             int         `json:"warmup,omitempty"`
	Samples            int         `json:"samples,omitempty"`
	CGO                *timingJSON `json:"cgo,omitempty"`
	WASM               *timingJSON `json:"wasm,omitempty"`
	WASMOverCGOMean    float64     `json:"wasmOverCgoMean,omitempty"`
	NativeBytes        int64       `json:"nativeBytes,omitempty"`
	WASMBytes          int64       `json:"wasmBytes,omitempty"`
	NativeArchiveBytes int64       `json:"nativeArchiveBytes,omitempty"`
	OutputMatched      bool        `json:"outputMatched,omitempty"`
}

type runResult struct {
	Experiment string             `json:"experiment"`
	RunID      string             `json:"runId"`
	Warmup     int                `json:"warmup"`
	Samples    int                `json:"samples"`
	Algorithms []algorithmResult  `json:"algorithms"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "benchpqccgowasm: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root := flag.String("root", "", "go-ethereum 仓库根目录")
	algos := flag.String("algorithms", "aigis_sig2_verify,dilithium3_verify,ml_dsa_65_verify,slh_dsa_shake_192f_verify", "逗号分隔算法名")
	warmup := flag.Int("warmup", 10, "预热次数")
	samples := flag.Int("n", 100, "采样次数")
	outputDir := flag.String("output", "", "结果目录，默认 results/pqc-cgo-wasm/<run-id>")
	flag.Parse()

	repoRoot, err := findRepoRoot(*root)
	if err != nil {
		return err
	}
	pqcgoRoot := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "pqcgo")
	runID := time.Now().Format("20060102-150405")
	outDir := *outputDir
	if outDir == "" {
		outDir = filepath.Join(repoRoot, "experiments", "cryptoupgrade", "results", experimentName, runID)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	names := splitCSV(*algos)
	result := runResult{
		Experiment: experimentName,
		RunID:      runID,
		Warmup:     *warmup,
		Samples:    *samples,
	}
	wasmRT := pqcbench.NewWASMRuntime()

	failed := false
	for _, name := range names {
		item := measureAlgorithm(pqcgoRoot, outDir, name, *warmup, *samples, wasmRT)
		result.Algorithms = append(result.Algorithms, item)
		if item.Skipped {
			fmt.Printf("skip %s: %s\n", name, item.SkipReason)
			continue
		}
		if item.Error != "" || item.CGO == nil || item.WASM == nil {
			failed = true
			fmt.Printf("fail %s: %s\n", name, item.Error)
			continue
		}
		fmt.Printf("%s  cgo=%.3fms  wasm=%.3fms  nativeBytes=%d  wasmBytes=%d  overhead=%.1f%%\n",
			name, item.CGO.MeanMillis, item.WASM.MeanMillis, item.NativeBytes, item.WASMBytes, item.WASMOverCGOMean*100)
	}

	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(outDir, "result.json")
	if err := os.WriteFile(jsonPath, raw, 0o644); err != nil {
		return err
	}
	summary := renderSummary(result, jsonPath)
	if err := os.WriteFile(filepath.Join(outDir, "result.txt"), []byte(summary), 0o644); err != nil {
		return err
	}
	fmt.Print(summary)
	if failed {
		return errors.New("one or more algorithms failed contrast")
	}
	return nil
}

func measureAlgorithm(pqcgoRoot, outDir, name string, warmup, n int, wasmRT *pqcbench.WASMRuntime) algorithmResult {
	item := algorithmResult{Algorithm: name, Warmup: warmup, Samples: n}
	verify, ok := lookup(name)
	if !ok {
		item.Skipped = true
		item.SkipReason = "unknown algorithm"
		return item
	}
	if !verify.enabled() {
		item.Skipped = true
		item.SkipReason = pqcbench.ErrCGODisabled.Error()
		return item
	}
	if err := pqcbench.CheckArtifacts(pqcgoRoot, name); err != nil {
		item.Skipped = true
		item.SkipReason = err.Error()
		return item
	}
	vectors, err := pqcbench.LoadVectors(pqcgoRoot, name)
	if err != nil {
		item.Skipped = true
		item.SkipReason = err.Error()
		return item
	}
	sizes, err := pqcbench.MeasureSizes(pqcgoRoot, name)
	if err != nil {
		item.Skipped = true
		item.SkipReason = err.Error()
		return item
	}

	cgoOK, err := verify.verify(vectors.PublicKey, vectors.Message, vectors.Signature)
	if err != nil {
		if errors.Is(err, pqcbench.ErrCGODisabled) {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		return failResult(item, err)
	}

	input, err := packVerifyInput(vectors.PublicKey, vectors.Message, vectors.Signature)
	if err != nil {
		return failResult(item, err)
	}
	wasmDir := filepath.Join(outDir, "wasm-cache", name)
	if err := os.MkdirAll(wasmDir, 0o755); err != nil {
		return failResult(item, err)
	}
	wasmPath := pqcbench.WASMPath(pqcgoRoot, name)
	compiledPath := filepath.Join(wasmDir, "compiled")
	wasmOut, err := wasmRT.Execute(context.Background(), wasmPath, compiledPath, input)
	if err != nil {
		return failResult(item, err)
	}
	wasmOK, err := unpackBool(wasmOut)
	if err != nil {
		return failResult(item, err)
	}
	if cgoOK != wasmOK || !cgoOK {
		return failResult(item, fmt.Errorf("%w: cgo=%v wasm=%v", pqcbench.ErrOutputMismatch, cgoOK, wasmOK))
	}

	cgoTiming := timePath(warmup, n, func() error {
		ok, err := verify.verify(vectors.PublicKey, vectors.Message, vectors.Signature)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("cgo verify returned false")
		}
		return nil
	})
	if cgoTiming.err != nil {
		return failResult(item, cgoTiming.err)
	}
	wasmTiming := timePath(warmup, n, func() error {
		out, err := wasmRT.Execute(context.Background(), wasmPath, compiledPath, input)
		if err != nil {
			return err
		}
		ok, err := unpackBool(out)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("wasm verify returned false")
		}
		return nil
	})
	if wasmTiming.err != nil {
		return failResult(item, wasmTiming.err)
	}

	cgoJSON := toTimingJSON(pqcbench.Summarize(cgoTiming.samples))
	wasmJSON := toTimingJSON(pqcbench.Summarize(wasmTiming.samples))
	item.CGO = &cgoJSON
	item.WASM = &wasmJSON
	item.WASMOverCGOMean = pqcbench.Overhead(wasmJSON.MeanMillis, cgoJSON.MeanMillis)
	item.NativeBytes = sizes.NativeBytes
	item.WASMBytes = sizes.WASMBytes
	item.NativeArchiveBytes = sizes.NativeArchiveBytes
	item.OutputMatched = true
	return item
}

func failResult(item algorithmResult, err error) algorithmResult {
	item.Error = err.Error()
	item.CGO = nil
	item.WASM = nil
	item.OutputMatched = false
	return item
}

func lookup(name string) (verifier, bool) {
	switch name {
	case aigis.Name:
		return verifier{name: aigis.Name, enabled: aigis.Enabled, verify: aigis.Verify}, true
	case dilithium3.Name:
		return verifier{name: dilithium3.Name, enabled: dilithium3.Enabled, verify: dilithium3.Verify}, true
	case mldsa65.Name:
		return verifier{name: mldsa65.Name, enabled: mldsa65.Enabled, verify: mldsa65.Verify}, true
	case slhdsa.Name:
		return verifier{name: slhdsa.Name, enabled: slhdsa.Enabled, verify: slhdsa.Verify}, true
	default:
		return verifier{}, false
	}
}

type timed struct {
	samples []float64
	err     error
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

func packVerifyInput(pk, message, signature []byte) ([]byte, error) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}
	return args.Pack(pk, message, signature)
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

func toTimingJSON(t pqcbench.Timing) timingJSON {
	return timingJSON{
		MeanMillis:   t.MeanMillis,
		P50Millis:    t.P50Millis,
		P95Millis:    t.P95Millis,
		MinMillis:    t.MinMillis,
		MaxMillis:    t.MaxMillis,
		SampleMillis: t.SampleMillis,
	}
}

func renderSummary(result runResult, jsonPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PQC CGO vs WASM (warmup=%d n=%d)\n", result.Warmup, result.Samples)
	for _, item := range result.Algorithms {
		if item.Skipped {
			fmt.Fprintf(&b, "  %-32s skipped  %s\n", item.Algorithm, item.SkipReason)
			continue
		}
		if item.Error != "" {
			fmt.Fprintf(&b, "  %-32s failed   %s\n", item.Algorithm, item.Error)
			continue
		}
		fmt.Fprintf(&b, "  %-32s cgo=%.3fms wasm=%.3fms nativeBytes=%d wasmBytes=%d overhead=%.1f%%\n",
			item.Algorithm, item.CGO.MeanMillis, item.WASM.MeanMillis, item.NativeBytes, item.WASMBytes, item.WASMOverCGOMean*100)
	}
	fmt.Fprintf(&b, "wrote %s\n", jsonPath)
	return b.String()
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func findRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "experiments", "cryptoupgrade", "algorithm", "pqcgo")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot locate go-ethereum root; pass -root")
		}
		dir = parent
	}
}
