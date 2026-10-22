package wasmruntime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/pqcnative"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func TestAigisSig2DirectExecutionTiming(t *testing.T) {
	errInvalidAigisOutput := errors.New("Aigis-sig2 verification returned false")
	if !pqcnative.Enabled() {
		t.Skip("AigisSig2Verify native comparison requires CGO")
	}
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	input, err := (abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}).Pack(
		model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "aigis_sig2_verify.wasm")
	cachePath := filepath.Join(dir, "compiled")
	if err := os.WriteFile(wasmPath, model.AigisSig2VerifyWASM, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	loader := NewLoader(Config{})
	module, err := loader.prepare(ctx, wasmPath, cachePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = module.Close(ctx) })
	if err := module.ensureInputCapacity(inputMemoryOffset, uint32(len(input))); err != nil {
		t.Fatal(err)
	}
	if !module.memory.Write(inputMemoryOffset, input) {
		t.Fatal("write WASM input")
	}
	expectedOutput := make([]byte, 32)
	expectedOutput[31] = 1
	baselinePath := filepath.Join(dir, "baseline.wasm")
	if err := os.WriteFile(baselinePath, buildConstantModule([]byte("ok"), false, false), 0o600); err != nil {
		t.Fatal(err)
	}
	baseline, err := loader.prepare(ctx, baselinePath, filepath.Join(dir, "baseline-cache"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = baseline.Close(ctx) })
	baselineCall := func() error {
		_, err := baseline.execute.Call(ctx, api.EncodeU32(inputMemoryOffset), api.EncodeU32(uint32(len(input))))
		return err
	}
	wasmCall := func() error {
		results, err := module.execute.Call(ctx, api.EncodeU32(inputMemoryOffset), api.EncodeU32(uint32(len(input))))
		if err != nil {
			return err
		}
		output, err := module.readLengthPrefixed(api.DecodeU32(results[0]))
		if err != nil {
			return err
		}
		if !bytes.Equal(output, expectedOutput) {
			return errInvalidAigisOutput
		}
		return nil
	}
	nativeCall := func() error {
		valid, err := pqcnative.Verify(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
		if err != nil {
			return err
		}
		if !valid {
			return errInvalidAigisOutput
		}
		return nil
	}
	for _, path := range []struct {
		name string
		call func() error
	}{
		{"native-verify", nativeCall},
		{"wasm-boundary-baseline", baselineCall},
		{"wasm-execute-export", wasmCall},
		{"wasmruntime.Execute", func() error {
			_, err := loader.Execute(ctx, wasmPath, cachePath, input)
			return err
		}},
	} {
		for i := 0; i < 10; i++ {
			if err := path.call(); err != nil {
				t.Fatalf("%s warmup: %v", path.name, err)
			}
		}
		var total time.Duration
		for i := 0; i < 100; i++ {
			start := time.Now()
			if err := path.call(); err != nil {
				t.Fatalf("%s sample: %v", path.name, err)
			}
			total += time.Since(start)
		}
		t.Logf("%-22s mean=%8.3fms (%d ns)", path.name, float64(total.Nanoseconds())/100/1e6, total.Nanoseconds()/100)
	}
}

type aigisFunctionProfile struct {
	name  string
	calls int
	total time.Duration
	start []time.Time
}

func (profile *aigisFunctionProfile) Before(context.Context, api.Module, api.FunctionDefinition, []uint64, experimental.StackIterator) {
	profile.start = append(profile.start, time.Now())
}

func (profile *aigisFunctionProfile) After(context.Context, api.Module, api.FunctionDefinition, []uint64) {
	last := len(profile.start) - 1
	profile.total += time.Since(profile.start[last])
	profile.start = profile.start[:last]
	profile.calls++
}

func (profile *aigisFunctionProfile) Abort(context.Context, api.Module, api.FunctionDefinition, error) {
	profile.start = profile.start[:len(profile.start)-1]
}

func TestAigisSig2GuestFunctionProfile(t *testing.T) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	input, err := (abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}).Pack(
		model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	var profiles []*aigisFunctionProfile
	factory := experimental.FunctionListenerFactoryFunc(func(definition api.FunctionDefinition) experimental.FunctionListener {
		name := definition.DebugName()
		for _, term := range []string{"verify_internal", "expand_mat", "polyvecl_ntt", "polyveck_ntt", "polyvecl_pointwise_acc_invmontgomery", "polyveck_invntt_montgomery", "polyveck_use_hint", "challenge", "shake256", "poly_ntt", "poly_pointwise_invmontgomery"} {
			if strings.HasSuffix(name, "."+term) || strings.HasSuffix(name, "_"+term) {
				profile := &aigisFunctionProfile{name: name}
				profiles = append(profiles, profile)
				return profile
			}
		}
		return nil
	})
	ctx := experimental.WithFunctionListenerFactory(context.Background(), factory)
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(DefaultMemoryLimitPages))
	t.Cleanup(func() { _ = runtime.Close(ctx) })
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		t.Fatal(err)
	}
	compiled, err := runtime.CompileModule(ctx, model.AigisSig2VerifyWASM)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		t.Fatal(err)
	}
	if initialize := instance.ExportedFunction(wasmInitExport); initialize != nil {
		if _, err := initialize.Call(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !instance.Memory().Write(inputMemoryOffset, input) {
		t.Fatal("write WASM input")
	}
	execute := instance.ExportedFunction("execute")
	expectedOutput := make([]byte, 32)
	expectedOutput[31] = 1
	for sample := 0; sample < 10; sample++ {
		results, err := execute.Call(ctx, api.EncodeU32(inputMemoryOffset), api.EncodeU32(uint32(len(input))))
		if err != nil {
			t.Fatal(err)
		}
		pointer := api.DecodeU32(results[0])
		length, ok := instance.Memory().ReadUint32Le(pointer)
		if !ok || length != 32 {
			t.Fatal("invalid WASM output")
		}
		output, ok := instance.Memory().Read(pointer+4, length)
		if !ok || !bytes.Equal(output, expectedOutput) {
			t.Fatal("WASM verification did not return true")
		}
	}
	if len(profiles) == 0 {
		t.Fatal("WASM module has no function names to profile")
	}
	sort.Slice(profiles, func(left, right int) bool { return profiles[left].total > profiles[right].total })
	for index, profile := range profiles {
		if profile.calls == 0 {
			continue
		}
		t.Logf("%-65s calls=%5d inclusive=%8.3fms", profile.name, profile.calls, float64(profile.total.Nanoseconds())/1e6)
		if index >= 20 {
			break
		}
	}
}

func TestAigisSig2TerminationCheckProfile(t *testing.T) {
	errInvalidAigisOutput := errors.New("Aigis-sig2 verification returned false")
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	input, err := (abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}).Pack(
		model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	expectedOutput := make([]byte, 32)
	expectedOutput[31] = 1
	for _, terminationCheck := range []bool{true, false} {
		config := wazero.NewRuntimeConfig().WithMemoryLimitPages(DefaultMemoryLimitPages).WithCloseOnContextDone(terminationCheck)
		runtime := wazero.NewRuntimeWithConfig(ctx, config)
		t.Cleanup(func() { _ = runtime.Close(ctx) })
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
			t.Fatal(err)
		}
		compiled, err := runtime.CompileModule(ctx, model.AigisSig2VerifyWASM)
		if err != nil {
			t.Fatal(err)
		}
		instance, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStartFunctions())
		if err != nil {
			t.Fatal(err)
		}
		if initialize := instance.ExportedFunction(wasmInitExport); initialize != nil {
			if _, err := initialize.Call(ctx); err != nil {
				t.Fatal(err)
			}
		}
		memory := instance.Memory()
		if !memory.Write(inputMemoryOffset, input) {
			t.Fatal("write WASM input")
		}
		execute := instance.ExportedFunction("execute")
		call := func() error {
			results, err := execute.Call(ctx, api.EncodeU32(inputMemoryOffset), api.EncodeU32(uint32(len(input))))
			if err != nil {
				return err
			}
			pointer := api.DecodeU32(results[0])
			length, ok := memory.ReadUint32Le(pointer)
			if !ok || length != 32 {
				return errors.New("invalid WASM output length")
			}
			output, ok := memory.Read(pointer+4, length)
			if !ok || !bytes.Equal(output, expectedOutput) {
				return errInvalidAigisOutput
			}
			return nil
		}
		for sample := 0; sample < 10; sample++ {
			if err := call(); err != nil {
				t.Fatal(err)
			}
		}
		var total time.Duration
		for sample := 0; sample < 100; sample++ {
			start := time.Now()
			if err := call(); err != nil {
				t.Fatal(err)
			}
			total += time.Since(start)
		}
		t.Logf("closeOnContextDone=%t mean=%d ns", terminationCheck, total.Nanoseconds()/100)
	}
}
