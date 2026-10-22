package cgowasm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/pqcbench/mldsa65"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench/groth16bls12381"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/core/zkbench/groth16bn254"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	terminationProfileWarmup  = 3
	terminationProfileSamples = 20
	terminationInputOffset    = uint32(1 << 16)
)

func TestWASMTerminationCheckLatency(t *testing.T) {
	repoRoot, err := FindRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		root    string
		load    func(string, string) ([][]byte, error)
		enabled func() bool
		verify  func([]byte, []byte, []byte) (bool, error)
	}{
		{
			name: "ml_dsa_65_verify", root: "pqcgo", enabled: mldsa65.Enabled, verify: mldsa65.Verify,
			load: loadPQCInputs,
		},
		{
			name: "groth16_bn254_verify", root: "zkgo", enabled: groth16bn254.Enabled, verify: groth16bn254.Verify,
			load: loadGroth16Inputs,
		},
		{
			name: "groth16_bls12381_verify", root: "zkgo", enabled: groth16bls12381.Enabled, verify: groth16bls12381.Verify,
			load: loadGroth16Inputs,
		},
	}
	for _, candidate := range cases {
		t.Run(candidate.name, func(t *testing.T) {
			if !candidate.enabled() {
				t.Skip("native verification requires CGO")
			}
			root := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", candidate.root)
			vectors, err := candidate.load(root, candidate.name)
			if err != nil {
				t.Fatal(err)
			}
			input, err := packBytes3(vectors[0], vectors[1], vectors[2])
			if err != nil {
				t.Fatal(err)
			}
			wasm, err := os.ReadFile(filepath.Join(root, candidate.name, candidate.name+".wasm"))
			if err != nil {
				t.Fatal(err)
			}
			native := func() error {
				valid, err := candidate.verify(vectors[0], vectors[1], vectors[2])
				if err != nil {
					return err
				}
				if !valid {
					return fmt.Errorf("%s native verification returned false", candidate.name)
				}
				return nil
			}
			t.Logf("native mean=%s", measureVerification(t, native))
			for _, closeOnContextDone := range []bool{true, false, false, true} {
				t.Logf("closeOnContextDone=%t mean=%s", closeOnContextDone,
					measureWASMVerification(t, wasm, input, closeOnContextDone))
			}
		})
	}
}

func loadPQCInputs(root, name string) ([][]byte, error) {
	vectors, err := pqcbench.LoadVectors(root, name)
	if err != nil {
		return nil, err
	}
	return [][]byte{vectors.PublicKey, vectors.Message, vectors.Signature}, nil
}

func loadGroth16Inputs(root, name string) ([][]byte, error) {
	vectors, err := zkbench.LoadVectors(root, name)
	if err != nil {
		return nil, err
	}
	return [][]byte{vectors.VK, vectors.Public, vectors.Proof}, nil
}

func measureWASMVerification(t *testing.T, wasm, input []byte, closeOnContextDone bool) time.Duration {
	t.Helper()
	verify, closeRuntime := prepareWASMVerification(t, wasm, input, closeOnContextDone)
	defer closeRuntime()
	return measureVerification(t, verify)
}

func prepareWASMVerification(t *testing.T, wasm, input []byte, closeOnContextDone bool) (func() error, func()) {
	return prepareWASMVerificationWithContext(t, context.Background(), wasm, input, closeOnContextDone)
}

func prepareWASMVerificationWithContext(t *testing.T, ctx context.Context, wasm, input []byte, closeOnContextDone bool) (func() error, func()) {
	call, check, closeRuntime := prepareWASMCall(t, ctx, wasm, input, closeOnContextDone)
	return func() error {
		results, err := call()
		if err != nil {
			return err
		}
		return check(results)
	}, closeRuntime
}

func prepareWASMCall(t *testing.T, ctx context.Context, wasm, input []byte, closeOnContextDone bool) (func() ([]uint64, error), func([]uint64) error, func()) {
	t.Helper()
	config := wazero.NewRuntimeConfig().WithMemoryLimitPages(256).WithCloseOnContextDone(closeOnContextDone)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	closeRuntime := func() { _ = runtime.Close(ctx) }
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		t.Fatal(err)
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		t.Fatal(err)
	}
	if initialize := instance.ExportedFunction("_initialize"); initialize != nil {
		if _, err := initialize.Call(ctx); err != nil {
			t.Fatal(err)
		}
	}
	memory := instance.Memory()
	required := terminationInputOffset + uint32(len(input))
	if size := memory.Size(); required > size {
		if _, ok := memory.Grow((required - size + 65535) / 65536); !ok {
			t.Fatal("grow WASM input memory")
		}
	}
	if !memory.Write(terminationInputOffset, input) {
		t.Fatal("write WASM input")
	}
	execute := instance.ExportedFunction("execute")
	if execute == nil {
		t.Fatal("missing WASM execute export")
	}
	expected := make([]byte, 32)
	expected[31] = 1
	return func() ([]uint64, error) {
			return execute.Call(ctx, api.EncodeU32(terminationInputOffset), api.EncodeU32(uint32(len(input))))
		}, func(results []uint64) error {
			if len(results) != 1 {
				return fmt.Errorf("execute returned %d results", len(results))
			}
			pointer := api.DecodeU32(results[0])
			length, ok := memory.ReadUint32Le(pointer)
			if !ok {
				return fmt.Errorf("invalid WASM output pointer %d", pointer)
			}
			output, ok := memory.Read(pointer+4, length)
			if !ok || !bytes.Equal(output, expected) {
				return fmt.Errorf("WASM verification did not return ABI true")
			}
			return nil
		}, closeRuntime
}

func TestWASMExecuteOnlyLatency(t *testing.T) {
	repoRoot, err := FindRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		root string
		load func(string, string) ([][]byte, error)
	}{
		{name: "aigis_sig2_verify", root: "pqcgo", load: loadPQCInputs},
		{name: "dilithium3_verify", root: "pqcgo", load: loadPQCInputs},
		{name: "ml_dsa_65_verify", root: "pqcgo", load: loadPQCInputs},
		{name: "slh_dsa_shake_192f_verify", root: "pqcgo", load: loadPQCInputs},
		{name: "groth16_bls12381_verify", root: "zkgo", load: loadGroth16Inputs},
		{name: "groth16_bn254_verify", root: "zkgo", load: loadGroth16Inputs},
	}
	for _, candidate := range cases {
		t.Run(candidate.name, func(t *testing.T) {
			root := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", candidate.root)
			vectors, err := candidate.load(root, candidate.name)
			if err != nil {
				t.Fatal(err)
			}
			input, err := packBytes3(vectors[0], vectors[1], vectors[2])
			if err != nil {
				t.Fatal(err)
			}
			wasm, err := os.ReadFile(filepath.Join(root, candidate.name, candidate.name+".wasm"))
			if err != nil {
				t.Fatal(err)
			}
			call, check, closeRuntime := prepareWASMCall(t, context.Background(), wasm, input, false)
			defer closeRuntime()
			for sample := 0; sample < terminationProfileWarmup; sample++ {
				results, err := call()
				if err != nil {
					t.Fatal(err)
				}
				if err := check(results); err != nil {
					t.Fatal(err)
				}
			}
			var total time.Duration
			for sample := 0; sample < terminationProfileSamples; sample++ {
				start := time.Now()
				results, err := call()
				elapsed := time.Since(start)
				if err != nil {
					t.Fatal(err)
				}
				if err := check(results); err != nil {
					t.Fatal(err)
				}
				total += elapsed
			}
			t.Logf("WithCloseOnContextDone=false execute.Call mean=%s samples=%d", total/terminationProfileSamples, terminationProfileSamples)
		})
	}
}

func measureVerification(t *testing.T, verify func() error) time.Duration {
	return measureVerificationSamples(t, verify, terminationProfileSamples)
}

func measureVerificationSamples(t *testing.T, verify func() error, samples int) time.Duration {
	t.Helper()
	for sample := 0; sample < terminationProfileWarmup; sample++ {
		if err := verify(); err != nil {
			t.Fatal(err)
		}
	}
	var total time.Duration
	for sample := 0; sample < samples; sample++ {
		start := time.Now()
		if err := verify(); err != nil {
			t.Fatal(err)
		}
		total += time.Since(start)
	}
	return total / time.Duration(samples)
}

func TestGroth16PerfHotLoop(t *testing.T) {
	curve := os.Getenv("CRYPTOUPGRADE_PERF_CURVE")
	mode := os.Getenv("CRYPTOUPGRADE_PERF_MODE")
	if curve == "" && mode == "" {
		t.Skip("set CRYPTOUPGRADE_PERF_CURVE and CRYPTOUPGRADE_PERF_MODE to profile hot calls")
	}
	var verify func([]byte, []byte, []byte) (bool, error)
	switch curve {
	case "bn254":
		verify = groth16bn254.Verify
		curve = "groth16_bn254_verify"
	case "bls12381":
		verify = groth16bls12381.Verify
		curve = "groth16_bls12381_verify"
	default:
		t.Fatalf("unknown curve %q", curve)
	}
	repoRoot, err := FindRepoRoot("")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(repoRoot, "experiments", "cryptoupgrade", "algorithm", "zkgo")
	vectors, err := loadGroth16Inputs(root, curve)
	if err != nil {
		t.Fatal(err)
	}
	var call func() error
	switch mode {
	case "native":
		call = func() error {
			valid, err := verify(vectors[0], vectors[1], vectors[2])
			if err != nil {
				return err
			}
			if !valid {
				return fmt.Errorf("%s native verification returned false", curve)
			}
			return nil
		}
	case "wasm":
		input, err := packBytes3(vectors[0], vectors[1], vectors[2])
		if err != nil {
			t.Fatal(err)
		}
		wasm, err := os.ReadFile(filepath.Join(root, curve, curve+".wasm"))
		if err != nil {
			t.Fatal(err)
		}
		var closeRuntime func()
		call, closeRuntime = prepareWASMVerification(t, wasm, input, false)
		defer closeRuntime()
	default:
		t.Fatalf("unknown mode %q", mode)
	}
	t.Logf("curve=%s mode=%s samples=400 mean=%s", curve, mode, measureVerificationSamples(t, call, 400))
}
