package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type fixture struct {
	name      string
	file      string
	unit      string
	levels    []int
	input     func(int) ([]byte, error)
	reference int
}

type measurement struct {
	Algorithm    string  `json:"algorithm"`
	Metric       string  `json:"metric"`
	Value        int     `json:"value"`
	InputBytes   int     `json:"inputBytes"`
	OutputDigest string  `json:"outputSha256"`
	MedianNs     int64   `json:"medianNs"`
	P95Ns        int64   `json:"p95Ns"`
	MinimumNs    int64   `json:"minimumNs"`
	SamplesNs    []int64 `json:"samplesNs"`
	CandidateGas uint64  `json:"candidateGas"`
}

type report struct {
	Timestamp       time.Time         `json:"timestamp"`
	GoVersion       string            `json:"goVersion"`
	OS              string            `json:"os"`
	Architecture    string            `json:"architecture"`
	CPUCount        int               `json:"cpuCount"`
	Runtime         string            `json:"runtime"`
	WasmHashes      map[string]string `json:"wasmSha256"`
	Warmup          int               `json:"warmup"`
	Samples         int               `json:"samples"`
	AnchorAlgorithm string            `json:"anchorAlgorithm"`
	AnchorGas       uint64            `json:"anchorGas"`
	AnchorMedianNs  int64             `json:"anchorMedianNs"`
	Measurements    []measurement     `json:"measurements"`
}

type engine struct {
	runtime wazero.Runtime
	module  api.Module
	entry   api.Function
	alloc   api.Function
	memory  api.Memory
	offset  uint32
}

func newEngine(ctx context.Context, path string) (*engine, error) {
	wasm, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(256))
	if _, err = wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	compiled, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	module, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(filepath.Base(path)).WithStartFunctions())
	if err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	if init := module.ExportedFunction("_initialize"); init != nil {
		if _, err = init.Call(ctx); err != nil {
			_ = rt.Close(ctx)
			return nil, err
		}
	}
	result := &engine{runtime: rt, module: module, entry: module.ExportedFunction("execute"), alloc: module.ExportedFunction("allocate"), memory: module.Memory()}
	if result.entry == nil || result.memory == nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("missing WASM execute or memory export: %s", path)
	}
	result.offset = result.memory.Size()
	if result.offset < 1<<16 {
		result.offset = 1 << 16
	}
	return result, nil
}

func (engine *engine) call(ctx context.Context, input []byte) ([]byte, error) {
	offset := engine.offset
	if engine.alloc != nil {
		result, err := engine.alloc.Call(ctx, api.EncodeU32(uint32(len(input))))
		if err != nil || len(result) != 1 {
			return nil, fmt.Errorf("allocate input: %v", err)
		}
		offset = api.DecodeU32(result[0])
	} else if uint64(offset)+uint64(len(input)) > uint64(engine.memory.Size()) {
		missing := uint64(offset) + uint64(len(input)) - uint64(engine.memory.Size())
		if _, ok := engine.memory.Grow(uint32((missing + 65535) / 65536)); !ok {
			return nil, fmt.Errorf("grow WASM memory")
		}
	}
	if !engine.memory.Write(offset, input) {
		return nil, fmt.Errorf("write WASM input")
	}
	result, err := engine.entry.Call(ctx, api.EncodeU32(offset), api.EncodeU32(uint32(len(input))))
	if err != nil || len(result) != 1 {
		return nil, fmt.Errorf("execute: %v", err)
	}
	pointer := api.DecodeU32(result[0])
	length, ok := engine.memory.ReadUint32Le(pointer)
	if !ok {
		return nil, fmt.Errorf("read WASM output length")
	}
	output, ok := engine.memory.Read(pointer+4, length)
	if !ok {
		return nil, fmt.Errorf("read WASM output")
	}
	return append([]byte(nil), output...), nil
}

func pack(types []string, values ...interface{}) ([]byte, error) {
	arguments := make(abi.Arguments, 0, len(types))
	for _, name := range types {
		typ, err := abi.NewType(name, "", nil)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, abi.Argument{Type: typ})
	}
	return arguments.Pack(values...)
}

func filled(length int) []byte {
	return bytes.Repeat([]byte{0x73}, length)
}

func schnorrProof(message []byte) []byte {
	proof := make([]byte, 3*256)
	proof[255], proof[511] = 4, 4
	digest := sha256.Sum256(append(append([]byte("cryptoupgrade-schnorr"), proof[:512]...), message...))
	response := new(big.Int).Add(new(big.Int).SetBytes(digest[:]), big.NewInt(1)).Bytes()
	copy(proof[len(proof)-len(response):], response)
	return proof
}

func fixtures() []fixture {
	variableBytes := func(length int) ([]byte, error) { return pack([]string{"bytes"}, filled(length)) }
	return []fixture{
		{"Add", "add.wasm", "operandBits", []int{8, 64, 128, 254}, func(bits int) ([]byte, error) {
			value := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), big.NewInt(1))
			return pack([]string{"int256", "int256"}, value, value)
		}, 64},
		{"Sha256", "sha256.wasm", "messageBytes", []int{32, 256, 1024, 4096}, variableBytes, 256},
		{"Blake2bSum256", "blake2b.wasm", "messageBytes", []int{32, 256, 1024, 4096}, variableBytes, 256},
		{"Pbkdf2Sha256", "pbkdf2_sha256.wasm", "iterations", []int{100, 500, 1000, 2000}, func(iterations int) ([]byte, error) {
			return pack([]string{"bytes", "bytes", "uint256", "uint256"}, []byte("password"), []byte("salt"), big.NewInt(int64(iterations)), big.NewInt(32))
		}, 500},
		{"Dh2048Secret", "dh2048.wasm", "exponentBits", []int{16, 64, 128, 256}, func(bits int) ([]byte, error) {
			secret := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), big.NewInt(1))
			return pack([]string{"bytes", "bytes"}, secret.Bytes(), []byte{4})
		}, 128},
		{"PedersenCommit", "pedersen_commit.wasm", "messageBits", []int{16, 64, 128, 256}, func(bits int) ([]byte, error) {
			message := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), big.NewInt(1))
			return pack([]string{"bytes", "bytes"}, message.Bytes(), []byte{11})
		}, 128},
		{"SchnorrVerify", "schnorr_proof.wasm", "messageBytes", []int{16, 64, 256, 1024}, func(length int) ([]byte, error) {
			message := filled(length)
			return pack([]string{"bytes", "bytes"}, message, schnorrProof(message))
		}, 256},
		{"PolynomialMul", "polynomial_mul.wasm", "polynomialLength", []int{2, 4, 8, 16}, func(length int) ([]byte, error) {
			left, right := make([]*big.Int, length), make([]*big.Int, length)
			for index := range left {
				left[index], right[index] = big.NewInt(int64(index+1)), big.NewInt(int64(index+1))
			}
			return pack([]string{"uint256[]", "uint256[]", "uint256"}, left, right, big.NewInt(12289))
		}, 4},
	}
}

func run(root string, warmup, samples int, anchorGas uint64) (report, error) {
	if warmup < 0 || samples < 3 || anchorGas == 0 {
		return report{}, fmt.Errorf("warmup must be nonnegative, samples >= 3, anchor gas positive")
	}
	ctx := context.Background()
	result := report{Timestamp: time.Now().UTC(), GoVersion: runtime.Version(), OS: runtime.GOOS, Architecture: runtime.GOARCH, CPUCount: runtime.NumCPU(), Runtime: "wazero", WasmHashes: make(map[string]string), Warmup: warmup, Samples: samples, AnchorAlgorithm: "Add", AnchorGas: anchorGas}
	for _, fixture := range fixtures() {
		path := filepath.Join(root, fixture.file)
		wasm, err := os.ReadFile(path)
		if err != nil {
			return report{}, err
		}
		result.WasmHashes[fixture.name] = fmt.Sprintf("%x", sha256.Sum256(wasm))
		engine, err := newEngine(ctx, path)
		if err != nil {
			return report{}, fmt.Errorf("%s: %w", fixture.name, err)
		}
		for _, level := range fixture.levels {
			input, err := fixture.input(level)
			if err != nil {
				return report{}, err
			}
			for count := 0; count < warmup; count++ {
				if _, err := engine.call(ctx, input); err != nil {
					return report{}, fmt.Errorf("%s %d warmup: %w", fixture.name, level, err)
				}
			}
			measure := measurement{Algorithm: fixture.name, Metric: fixture.unit, Value: level, InputBytes: len(input), SamplesNs: make([]int64, 0, samples)}
			for count := 0; count < samples; count++ {
				start := time.Now()
				output, err := engine.call(ctx, input)
				duration := time.Since(start).Nanoseconds()
				if err != nil || len(output) == 0 {
					return report{}, fmt.Errorf("%s %d sample %d: output=%d err=%v", fixture.name, level, count, len(output), err)
				}
				if fixture.name == "SchnorrVerify" && (len(output) != 32 || output[31] != 1) {
					return report{}, fmt.Errorf("invalid Schnorr proof result for input length %d", level)
				}
				digest := fmt.Sprintf("%x", sha256.Sum256(output))
				if count > 0 && digest != measure.OutputDigest {
					return report{}, fmt.Errorf("non-deterministic output for %s %d sample %d: first=%s current=%s", fixture.name, level, count, measure.OutputDigest, digest)
				}
				measure.OutputDigest = digest
				measure.SamplesNs = append(measure.SamplesNs, duration)
			}
			sorted := append([]int64(nil), measure.SamplesNs...)
			sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
			measure.MinimumNs, measure.MedianNs = sorted[0], sorted[len(sorted)/2]
			measure.P95Ns = sorted[(95*len(sorted)+99)/100-1]
			if fixture.name == "Add" && level == fixture.reference {
				result.AnchorMedianNs = measure.MedianNs
			}
			result.Measurements = append(result.Measurements, measure)
		}
		if err := engine.runtime.Close(ctx); err != nil {
			return report{}, err
		}
	}
	if result.AnchorMedianNs <= 0 {
		return report{}, fmt.Errorf("missing Add anchor measurement")
	}
	for index := range result.Measurements {
		measure := &result.Measurements[index]
		numerator := new(big.Int).Mul(new(big.Int).SetUint64(anchorGas), new(big.Int).SetInt64(measure.MedianNs))
		cost, remainder := new(big.Int).QuoRem(numerator, big.NewInt(result.AnchorMedianNs), new(big.Int))
		if remainder.Sign() != 0 {
			cost.Add(cost, big.NewInt(1))
		}
		if !cost.IsUint64() || cost.Sign() == 0 {
			return report{}, fmt.Errorf("invalid candidate gas for %s", measure.Algorithm)
		}
		measure.CandidateGas = cost.Uint64()
	}
	return result, nil
}

func main() {
	root := flag.String("wasm-dir", "experiments/cryptoupgrade/algorithm/go/wasm", "WASM fixture directory")
	warmup := flag.Int("warmup", 5, "warmup calls for each input")
	samples := flag.Int("samples", 31, "samples for each input")
	anchorGas := flag.Uint64("anchor-gas", 100, "candidate gas for the Add 64-bit anchor")
	output := flag.String("out", "", "JSON result path (default stdout)")
	flag.Parse()
	result, err := run(*root, *warmup, *samples, *anchorGas)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded = append(encoded, '\n')
	if *output == "" {
		_, err = os.Stdout.Write(encoded)
	} else {
		err = os.WriteFile(*output, encoded, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
