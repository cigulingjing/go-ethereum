// Package cgowasm 提供单算法 CGO vs WASM 对照。每条算法入口只 import 自己的 CGO 包。
package cgowasm

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
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench"
)

type Kind int

const (
	KindPQC Kind = iota
	KindGroth16
)

// Config 描述一个可独立运行的算法对照。
type Config struct {
	Name       string
	Kind       Kind
	Enabled    func() bool
	Verify     func(a, b, c []byte) (bool, error)
	Experiment string
}

type TimingJSON struct {
	MeanMillis   float64   `json:"meanMillis"`
	P50Millis    float64   `json:"p50Millis"`
	P95Millis    float64   `json:"p95Millis"`
	MinMillis    float64   `json:"minMillis"`
	MaxMillis    float64   `json:"maxMillis"`
	SampleMillis []float64 `json:"sampleMillis"`
}

type AlgorithmResult struct {
	Algorithm          string      `json:"algorithm"`
	Skipped            bool        `json:"skipped"`
	SkipReason         string      `json:"skipReason,omitempty"`
	Error              string      `json:"error,omitempty"`
	Warmup             int         `json:"warmup,omitempty"`
	Samples            int         `json:"samples,omitempty"`
	CGO                *TimingJSON `json:"cgo,omitempty"`
	WASM               *TimingJSON `json:"wasm,omitempty"`
	WASMOverCGOMean    float64     `json:"wasmOverCgoMean,omitempty"`
	NativeBytes        int64       `json:"nativeBytes,omitempty"`
	WASMBytes          int64       `json:"wasmBytes,omitempty"`
	NativeArchiveBytes int64       `json:"nativeArchiveBytes,omitempty"`
	OutputMatched      bool        `json:"outputMatched,omitempty"`
}

type RunResult struct {
	Experiment string            `json:"experiment"`
	RunID      string            `json:"runId"`
	Warmup     int               `json:"warmup"`
	Samples    int               `json:"samples"`
	Algorithms []AlgorithmResult `json:"algorithms"`
}

// Main 是单算法可执行入口。
func Main(cfg Config) {
	if err := RunCLI(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", cfg.Name, err)
		os.Exit(1)
	}
}

func RunCLI(cfg Config) error {
	root := flag.String("root", "", "go-ethereum 仓库根目录")
	warmup := flag.Int("warmup", 10, "预热次数")
	samples := flag.Int("n", 100, "采样次数")
	outputDir := flag.String("output", "", "结果目录，默认 output/results/<algorithm>/<run-id>")
	flag.Parse()

	repoRoot, err := FindRepoRoot(*root)
	if err != nil {
		return err
	}
	experiment := cfg.Experiment
	if experiment == "" {
		experiment = cfg.Name
	}
	runID := time.Now().Format("20060102-150405")
	outDir := *outputDir
	if outDir == "" {
		outDir = filepath.Join(repoRoot, "experiments", "cryptoupgrade", "output", "results", experiment, runID)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	result := RunResult{
		Experiment: experiment,
		RunID:      runID,
		Warmup:     *warmup,
		Samples:    *samples,
	}
	item := MeasureOne(cfg, repoRoot, outDir, *warmup, *samples, pqcbench.NewWASMRuntime())
	result.Algorithms = append(result.Algorithms, item)
	if err := writeResult(result, outDir); err != nil {
		return err
	}
	if item.Skipped {
		return fmt.Errorf("skipped: %s", item.SkipReason)
	}
	if item.Error != "" || item.CGO == nil || item.WASM == nil {
		return errors.New(item.Error)
	}
	return nil
}

func MeasureOne(cfg Config, repoRoot, outDir string, warmup, n int, wasmRT *pqcbench.WASMRuntime) AlgorithmResult {
	item := AlgorithmResult{Algorithm: cfg.Name, Warmup: warmup, Samples: n}
	if cfg.Enabled == nil || cfg.Verify == nil {
		item.Skipped = true
		item.SkipReason = "missing verify implementation"
		return item
	}
	if !cfg.Enabled() {
		item.Skipped = true
		item.SkipReason = pqcbench.ErrCGODisabled.Error()
		return item
	}

	var (
		argA, argB, argC []byte
		sizesNative      int64
		sizesWASM        int64
		sizesArchive     int64
		wasmPath         string
	)
	switch cfg.Kind {
	case KindPQC:
		root := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "pqcgo")
		if err := pqcbench.CheckArtifacts(root, cfg.Name); err != nil {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		vectors, err := pqcbench.LoadVectors(root, cfg.Name)
		if err != nil {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		sizes, err := pqcbench.MeasureSizes(root, cfg.Name)
		if err != nil {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		argA, argB, argC = vectors.PublicKey, vectors.Message, vectors.Signature
		sizesNative, sizesWASM, sizesArchive = sizes.NativeBytes, sizes.WASMBytes, sizes.NativeArchiveBytes
		wasmPath = pqcbench.WASMPath(root, cfg.Name)
	case KindGroth16:
		root := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "zkgo")
		if err := zkbench.CheckArtifacts(root, cfg.Name); err != nil {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		vectors, err := zkbench.LoadVectors(root, cfg.Name)
		if err != nil {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		sizes, err := zkbench.MeasureSizes(root, cfg.Name)
		if err != nil {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		argA, argB, argC = vectors.VK, vectors.Public, vectors.Proof
		sizesNative, sizesWASM = sizes.NativeBytes, sizes.WASMBytes
		wasmPath = zkbench.WASMPath(root, cfg.Name)
	default:
		item.Skipped = true
		item.SkipReason = "unknown algorithm kind"
		return item
	}

	cgoOK, err := cfg.Verify(argA, argB, argC)
	if err != nil {
		if errors.Is(err, pqcbench.ErrCGODisabled) {
			item.Skipped = true
			item.SkipReason = err.Error()
			return item
		}
		return failResult(item, err)
	}

	input, err := packBytes3(argA, argB, argC)
	if err != nil {
		return failResult(item, err)
	}
	wasmDir := filepath.Join(outDir, "wasm-cache", cfg.Name)
	if err := os.MkdirAll(wasmDir, 0o755); err != nil {
		return failResult(item, err)
	}
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
		ok, err := cfg.Verify(argA, argB, argC)
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
	item.NativeBytes = sizesNative
	item.WASMBytes = sizesWASM
	item.NativeArchiveBytes = sizesArchive
	item.OutputMatched = true
	return item
}

func WriteResult(result RunResult, outDir string) error {
	return writeResult(result, outDir)
}

func writeResult(result RunResult, outDir string) error {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(outDir, "result.json")
	if err := os.WriteFile(jsonPath, raw, 0o644); err != nil {
		return err
	}
	summary := RenderSummary(result, jsonPath)
	if err := os.WriteFile(filepath.Join(outDir, "result.txt"), []byte(summary), 0o644); err != nil {
		return err
	}
	fmt.Print(summary)
	return nil
}

func RenderSummary(result RunResult, jsonPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CGO vs WASM (warmup=%d n=%d)\n", result.Warmup, result.Samples)
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

func FindRepoRoot(explicit string) (string, error) {
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
			if _, err := os.Stat(filepath.Join(dir, "experiments", "cryptoupgrade")); err == nil {
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

func failResult(item AlgorithmResult, err error) AlgorithmResult {
	item.Error = err.Error()
	item.CGO = nil
	item.WASM = nil
	item.OutputMatched = false
	return item
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

func packBytes3(a, b, c []byte) ([]byte, error) {
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

func toTimingJSON(t pqcbench.Timing) TimingJSON {
	return TimingJSON{
		MeanMillis:   t.MeanMillis,
		P50Millis:    t.P50Millis,
		P95Millis:    t.P95Millis,
		MinMillis:    t.MinMillis,
		MaxMillis:    t.MaxMillis,
		SampleMillis: t.SampleMillis,
	}
}
