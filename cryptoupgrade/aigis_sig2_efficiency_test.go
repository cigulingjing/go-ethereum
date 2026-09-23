package cryptoupgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/pqcnative"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmruntime"
)

const (
	aigisEfficiencyWarmup  = 10
	aigisEfficiencySamples = 100
)

type aigisSchemeMetrics struct {
	Scheme       string    `json:"scheme"`
	MeanMillis   float64   `json:"meanMillis"`
	P50Millis    float64   `json:"p50Millis"`
	P95Millis    float64   `json:"p95Millis"`
	MinMillis    float64   `json:"minMillis"`
	MaxMillis    float64   `json:"maxMillis"`
	SampleMillis []float64 `json:"sampleMillis"`
	OutputHex    string    `json:"outputHex"`
}

type aigisEfficiencyResult struct {
	Experiment             string             `json:"experiment"`
	Algorithm              string             `json:"algorithm"`
	Warmup                 int                `json:"warmup"`
	Samples                int                `json:"samples"`
	WASM                   aigisSchemeMetrics `json:"wasm"`
	Precompile             aigisSchemeMetrics `json:"precompile"`
	WASMOverPrecompileMean float64            `json:"wasmOverPrecompileMean"`
	OutputMatched          bool               `json:"outputMatched"`
	Notes                  []string           `json:"notes"`
}

func TestAigisSig2PrecompileVerify(t *testing.T) {
	if !pqcnative.Enabled() {
		t.Skip("AigisSig2Verify precompile requires CGO")
	}
	entry, ok := PrecompileByName("AigisSig2Verify")
	if !ok {
		t.Fatal("missing AigisSig2Verify precompile")
	}
	if entry.Address() != common.CryptoUpgradeAigisSig2VerifyAddress {
		t.Fatalf("address %s, want %s", entry.Address(), common.CryptoUpgradeAigisSig2VerifyAddress)
	}
	input, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
	if err != nil {
		t.Fatal(err)
	}
	out, err := entry.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if !unpackAigisBool(t, out) {
		t.Fatalf("valid signature: got %x want true", out)
	}
	tampered := append([]byte(nil), model.AigisSig2Signature...)
	tampered[0] ^= 0x01
	badInput, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, tampered)
	if err != nil {
		t.Fatal(err)
	}
	badOut, err := entry.Run(badInput)
	if err != nil {
		t.Fatal(err)
	}
	if unpackAigisBool(t, badOut) {
		t.Fatal("tampered signature verified")
	}
}

func TestAigisSig2WASMVsPrecompileEfficiency(t *testing.T) {
	if !pqcnative.Enabled() {
		t.Skip("AigisSig2Verify precompile requires CGO")
	}
	if len(model.AigisSig2VerifyWASM) == 0 {
		t.Fatal("missing embedded Aigis WASM")
	}

	dir := t.TempDir()
	withRuntimePluginPaths(t, newPluginPaths(filepath.Join(dir, "plugin")), nil)
	resetAlgorithmInfoForTest(t)

	encoded, err := EncodeWasm(model.AigisSig2VerifyWASM)
	if err != nil {
		t.Fatal(err)
	}
	info := model.AigisSig2VerifyInfo(encoded)
	if err := ActivateAlgorithm(model.AigisSig2VerifyName, info); err != nil {
		t.Fatal(err)
	}

	input, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
	if err != nil {
		t.Fatal(err)
	}
	call, err := CodeStorageABI.Pack("callFunc", model.AigisSig2VerifyName, input)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := PrecompileByName("AigisSig2Verify")
	if !ok {
		t.Fatal("missing AigisSig2Verify precompile")
	}

	wasmMetrics := measureAigisScheme(t, "upgrade", func() ([]byte, error) {
		return RunCodeStorageCallAt(call, 1, true, nil)
	})
	precompileMetrics := measureAigisScheme(t, "precompile", func() ([]byte, error) {
		return entry.Run(input)
	})
	if wasmMetrics.OutputHex != precompileMetrics.OutputHex {
		t.Fatalf("output mismatch wasm=%s precompile=%s", wasmMetrics.OutputHex, precompileMetrics.OutputHex)
	}
	if !unpackAigisBool(t, mustDecodeHex(t, wasmMetrics.OutputHex)) {
		t.Fatal("expected valid signature to verify on both paths")
	}

	result := aigisEfficiencyResult{
		Experiment:             "aigis-exec-efficiency",
		Algorithm:              model.AigisSig2VerifyName,
		Warmup:                 aigisEfficiencyWarmup,
		Samples:                aigisEfficiencySamples,
		WASM:                   wasmMetrics,
		Precompile:             precompileMetrics,
		WASMOverPrecompileMean: ratioOver(wasmMetrics.MeanMillis, precompileMetrics.MeanMillis),
		OutputMatched:          true,
		Notes: []string{
			"in-process callFunc vs Precompile.Run; excludes RPC and upgrade/upload time",
			"both paths use Aigis-sig2 verify with empty ctx and the same embedded vectors",
		},
	}
	outDir := filepath.Join("..", "experiments", "cryptoupgrade", "results", "aigis-exec-efficiency", time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(outDir, "result.json")
	if err := os.WriteFile(jsonPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	summary := fmt.Sprintf(
		"Aigis-sig2 verify efficiency (warmup=%d n=%d)\n  WASM callFunc     mean=%.3fms p50=%.3fms p95=%.3fms\n  precompile Run    mean=%.3fms p50=%.3fms p95=%.3fms\n  WASM/precompile   mean ratio=%.2fx  overhead=%.1f%%\n  wrote %s\n",
		result.Warmup, result.Samples,
		result.WASM.MeanMillis, result.WASM.P50Millis, result.WASM.P95Millis,
		result.Precompile.MeanMillis, result.Precompile.P50Millis, result.Precompile.P95Millis,
		result.WASMOverPrecompileMean+1, result.WASMOverPrecompileMean*100,
		jsonPath,
	)
	if err := os.WriteFile(filepath.Join(outDir, "result.txt"), []byte(summary), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + summary)
}

func TestAigisSig2ExecutionBreakdown(t *testing.T) {
	if !pqcnative.Enabled() {
		t.Skip("AigisSig2Verify precompile requires CGO")
	}

	dir := t.TempDir()
	withRuntimePluginPaths(t, newPluginPaths(filepath.Join(dir, "plugin")), nil)
	resetAlgorithmInfoForTest(t)

	encoded, err := EncodeWasm(model.AigisSig2VerifyWASM)
	if err != nil {
		t.Fatal(err)
	}
	if err := ActivateAlgorithm(model.AigisSig2VerifyName, model.AigisSig2VerifyInfo(encoded)); err != nil {
		t.Fatal(err)
	}
	input, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
	if err != nil {
		t.Fatal(err)
	}
	call, err := CodeStorageABI.Pack("callFunc", model.AigisSig2VerifyName, input)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := PrecompileByName("AigisSig2Verify")
	if !ok {
		t.Fatal("missing AigisSig2Verify precompile")
	}
	wasmPath, compiledPath := runtimeArtifactPaths(model.AigisSig2VerifyName, 1)

	layers := []struct {
		name string
		fn   func() error
	}{
		{"cgo-verify", func() error {
			ok, err := pqcnative.Verify(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("cgo verify returned false")
			}
			return nil
		}},
		{"precompile-Run", func() error {
			out, err := entry.Run(input)
			if err != nil {
				return err
			}
			if !unpackAigisBool(t, out) {
				return fmt.Errorf("precompile returned false")
			}
			return nil
		}},
		{"wasmruntime.Execute", func() error {
			out, err := wasmruntime.Default.Execute(context.Background(), wasmPath, compiledPath, input)
			if err != nil {
				return err
			}
			if !unpackAigisBool(t, out) {
				return fmt.Errorf("wasm execute returned false")
			}
			return nil
		}},
		{"callFunc", func() error {
			out, err := RunCodeStorageCallAt(call, 1, true, nil)
			if err != nil {
				return err
			}
			if !unpackAigisBool(t, out) {
				return fmt.Errorf("callFunc returned false")
			}
			return nil
		}},
	}

	for _, layer := range layers {
		for i := 0; i < aigisEfficiencyWarmup; i++ {
			if err := layer.fn(); err != nil {
				t.Fatalf("%s warmup: %v", layer.name, err)
			}
		}
		var total time.Duration
		for i := 0; i < aigisEfficiencySamples; i++ {
			start := time.Now()
			if err := layer.fn(); err != nil {
				t.Fatalf("%s sample: %v", layer.name, err)
			}
			total += time.Since(start)
		}
		mean := total / time.Duration(aigisEfficiencySamples)
		t.Logf("%-22s mean=%8.3fms  (%d ns)", layer.name, float64(mean.Nanoseconds())/1e6, mean.Nanoseconds())
	}
}

func BenchmarkAigisSig2Verify(b *testing.B) {
	if !pqcnative.Enabled() {
		b.Skip("AigisSig2Verify precompile requires CGO")
	}
	input, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("precompile", func(b *testing.B) {
		entry, ok := PrecompileByName("AigisSig2Verify")
		if !ok {
			b.Fatal("missing AigisSig2Verify precompile")
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := entry.Run(input); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("wasm", func(b *testing.B) {
		dir := b.TempDir()
		withRuntimePluginPaths(b, newPluginPaths(filepath.Join(dir, "plugin")), nil)
		resetAlgorithmInfoForTest(b)
		encoded, err := EncodeWasm(model.AigisSig2VerifyWASM)
		if err != nil {
			b.Fatal(err)
		}
		if err := ActivateAlgorithm(model.AigisSig2VerifyName, model.AigisSig2VerifyInfo(encoded)); err != nil {
			b.Fatal(err)
		}
		call, err := CodeStorageABI.Pack("callFunc", model.AigisSig2VerifyName, input)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := RunCodeStorageCallAt(call, 1, true, nil); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := RunCodeStorageCallAt(call, 1, true, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func measureAigisScheme(t *testing.T, scheme string, fn func() ([]byte, error)) aigisSchemeMetrics {
	t.Helper()
	for i := 0; i < aigisEfficiencyWarmup; i++ {
		if _, err := fn(); err != nil {
			t.Fatalf("%s warmup: %v", scheme, err)
		}
	}
	samples := make([]float64, 0, aigisEfficiencySamples)
	var last []byte
	for i := 0; i < aigisEfficiencySamples; i++ {
		start := time.Now()
		out, err := fn()
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("%s sample %d: %v", scheme, i, err)
		}
		last = out
		samples = append(samples, float64(elapsed.Nanoseconds())/1e6)
	}
	return aigisSchemeMetrics{
		Scheme:       scheme,
		MeanMillis:   meanFloats(samples),
		P50Millis:    percentileFloats(samples, 0.50),
		P95Millis:    percentileFloats(samples, 0.95),
		MinMillis:    minFloats(samples),
		MaxMillis:    maxFloats(samples),
		SampleMillis: samples,
		OutputHex:    fmt.Sprintf("%x", last),
	}
}

func ratioOver(num, den float64) float64 {
	if den == 0 {
		return math.NaN()
	}
	return (num - den) / den
}

func meanFloats(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentileFloats(values []float64, p float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func minFloats(values []float64) float64 {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloats(values []float64) float64 {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func mustDecodeHex(t *testing.T, encoded string) []byte {
	t.Helper()
	return common.Hex2Bytes(encoded)
}
