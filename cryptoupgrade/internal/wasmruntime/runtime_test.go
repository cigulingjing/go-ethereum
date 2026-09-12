package wasmruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activationtrace"
)

func TestActivateAndExecute(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "Add.wasm")
	cachePath := filepath.Join(dir, "compiled")
	wasm := buildConstantModule([]byte("ok"), false, false)
	if err := os.WriteFile(wasmPath, wasm, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	loader := NewLoader(Config{MemoryLimitPages: 8, ExecutionBudget: time.Second})
	if err := loader.Activate(context.Background(), wasmPath, cachePath); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	output, err := loader.Execute(context.Background(), wasmPath, cachePath, []byte{1, 2, 3})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !bytes.Equal(output, []byte("ok")) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestActivateWritesCompileTrace(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "activation_trace.jsonl")
	t.Setenv(activationtrace.TraceFileEnvVar, tracePath)
	wasmPath := filepath.Join(dir, "Add.wasm")
	cachePath := filepath.Join(dir, "compiled")
	if err := os.WriteFile(wasmPath, buildConstantModule([]byte("ok"), false, false), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	ctx := activationtrace.ContextWithEvent(context.Background(), activationtrace.Event{
		Name:        "Add",
		Version:     1,
		TxHash:      "0xabc",
		BlockNumber: 7,
	})
	loader := NewLoader(Config{MemoryLimitPages: 8, ExecutionBudget: time.Second})
	if err := loader.Activate(ctx, wasmPath, cachePath); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected trace line count %d: %s", len(lines), raw)
	}
	var compiled, instantiated activationtrace.Event
	if err := json.Unmarshal([]byte(lines[0]), &compiled); err != nil {
		t.Fatalf("unmarshal compiled trace: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &instantiated); err != nil {
		t.Fatalf("unmarshal instantiated trace: %v", err)
	}
	if compiled.Stage != "wasm_compiled" || instantiated.Stage != "wasm_instantiated" {
		t.Fatalf("unexpected trace stages: %#v %#v", compiled, instantiated)
	}
	if compiled.Name != "Add" || compiled.TxHash != "0xabc" || compiled.WasmPath != wasmPath || compiled.CompiledPath != cachePath {
		t.Fatalf("unexpected compiled trace: %#v", compiled)
	}
}

func TestRejectIllegalImport(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "Import.wasm")
	cachePath := filepath.Join(dir, "compiled")
	wasm := buildConstantModule([]byte("ok"), true, false)
	if err := os.WriteFile(wasmPath, wasm, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	loader := NewLoader(Config{MemoryLimitPages: 8, ExecutionBudget: time.Second})
	if err := loader.Activate(context.Background(), wasmPath, cachePath); err == nil {
		t.Fatal("expected illegal import to fail")
	}
}

func TestExecutionBudgetStopsLoopingModule(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "Loop.wasm")
	cachePath := filepath.Join(dir, "compiled")
	wasm := buildLoopModule()
	if err := os.WriteFile(wasmPath, wasm, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	loader := NewLoader(Config{MemoryLimitPages: 8, ExecutionBudget: 20 * time.Millisecond})
	if err := loader.Activate(context.Background(), wasmPath, cachePath); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	_, err := loader.Execute(context.Background(), wasmPath, cachePath, nil)
	if err == nil {
		t.Fatal("expected looping module to exceed execution budget")
	}
}

func TestMissingExecuteExport(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "NoExecute.wasm")
	cachePath := filepath.Join(dir, "compiled")
	wasm := buildNoExecuteModule()
	if err := os.WriteFile(wasmPath, wasm, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	loader := NewLoader(Config{MemoryLimitPages: 8, ExecutionBudget: time.Second})
	if err := loader.Activate(context.Background(), wasmPath, cachePath); err == nil || !errors.Is(err, ErrExecuteExportMissing) {
		t.Fatalf("expected missing execute export error, got %v", err)
	}
}

func TestTinyGoFixturesExecute(t *testing.T) {
	dir := t.TempDir()
	loader := NewLoader(Config{MemoryLimitPages: 8, ExecutionBudget: time.Second})

	addInput := make([]byte, 64)
	binary.BigEndian.PutUint64(addInput[24:32], 1)
	binary.BigEndian.PutUint64(addInput[56:64], 2)
	addOut, err := loader.Execute(context.Background(), fixturePath(t, "add.wasm"), filepath.Join(dir, "add-cache"), addInput)
	if err != nil {
		t.Fatalf("Add fixture Execute failed: %v", err)
	}
	wantAdd := make([]byte, 32)
	binary.BigEndian.PutUint64(wantAdd[24:32], 3)
	if !bytes.Equal(addOut, wantAdd) {
		t.Fatalf("unexpected Add output: %x", addOut)
	}

	shaInput := abiBytesInput([]byte("abc"))
	shaOut, err := loader.Execute(context.Background(), fixturePath(t, "sha256.wasm"), filepath.Join(dir, "sha-cache"), shaInput)
	if err != nil {
		t.Fatalf("Sha256 fixture Execute failed: %v", err)
	}
	sum := sha256.Sum256([]byte("abc"))
	wantSha := abiBytesOutput(sum[:])
	if !bytes.Equal(shaOut, wantSha) {
		t.Fatalf("unexpected Sha256 output: %x", shaOut)
	}
}

func buildConstantModule(output []byte, withImport bool, infiniteLoop bool) []byte {
	var mod []byte
	mod = append(mod, 0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00)

	typeSection := []byte{0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f}
	mod = appendSection(mod, 1, typeSection)

	if withImport {
		importSection := []byte{0x01}
		importSection = appendString(importSection, "env")
		importSection = appendString(importSection, "abort")
		importSection = append(importSection, 0x00, 0x00)
		mod = appendSection(mod, 2, importSection)
	}

	functionSection := []byte{0x01, 0x00}
	mod = appendSection(mod, 3, functionSection)

	memorySection := []byte{0x01, 0x00, 0x01}
	mod = appendSection(mod, 5, memorySection)

	exportSection := []byte{0x02}
	exportSection = appendString(exportSection, "execute")
	exportSection = append(exportSection, 0x00)
	executeIndex := byte(0)
	if withImport {
		executeIndex = 1
	}
	exportSection = append(exportSection, executeIndex)
	exportSection = appendString(exportSection, "memory")
	exportSection = append(exportSection, 0x02, 0x00)
	mod = appendSection(mod, 7, exportSection)

	body := []byte{0x00}
	if infiniteLoop {
		body = append(body, 0x03, 0x40, 0x0c, 0x00, 0x0b)
	} else {
		body = append(body, 0x41, 0x80, 0x08, 0x0b)
	}
	codeSection := []byte{0x01}
	codeSection = append(codeSection, encodeU32(uint32(len(body)))...)
	codeSection = append(codeSection, body...)
	mod = appendSection(mod, 10, codeSection)

	data := make([]byte, 4+len(output))
	binary.LittleEndian.PutUint32(data[:4], uint32(len(output)))
	copy(data[4:], output)
	dataSection := []byte{0x01, 0x00, 0x41, 0x80, 0x08, 0x0b}
	dataSection = append(dataSection, encodeU32(uint32(len(data)))...)
	dataSection = append(dataSection, data...)
	mod = appendSection(mod, 11, dataSection)

	return mod
}

func buildNoExecuteModule() []byte {
	var mod []byte
	mod = append(mod, 0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00)
	typeSection := []byte{0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f}
	mod = appendSection(mod, 1, typeSection)
	functionSection := []byte{0x01, 0x00}
	mod = appendSection(mod, 3, functionSection)
	memorySection := []byte{0x01, 0x00, 0x01}
	mod = appendSection(mod, 5, memorySection)
	exportSection := []byte{0x01}
	exportSection = appendString(exportSection, "memory")
	exportSection = append(exportSection, 0x02, 0x00)
	mod = appendSection(mod, 7, exportSection)
	body := []byte{0x00, 0x41, 0x80, 0x08, 0x0b}
	codeSection := []byte{0x01}
	codeSection = append(codeSection, encodeU32(uint32(len(body)))...)
	codeSection = append(codeSection, body...)
	mod = appendSection(mod, 10, codeSection)
	return mod
}

func buildLoopModule() []byte {
	var mod []byte
	mod = append(mod, 0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00)
	typeSection := []byte{0x01, 0x60, 0x02, 0x7f, 0x7f, 0x00}
	mod = appendSection(mod, 1, typeSection)
	functionSection := []byte{0x01, 0x00}
	mod = appendSection(mod, 3, functionSection)
	memorySection := []byte{0x01, 0x00, 0x01}
	mod = appendSection(mod, 5, memorySection)
	exportSection := []byte{0x01}
	exportSection = appendString(exportSection, "execute")
	exportSection = append(exportSection, 0x00, 0x00)
	mod = appendSection(mod, 7, exportSection)
	body := []byte{0x00, 0x03, 0x40, 0x0c, 0x00, 0x0b, 0x0b}
	codeSection := []byte{0x01}
	codeSection = append(codeSection, encodeU32(uint32(len(body)))...)
	codeSection = append(codeSection, body...)
	mod = appendSection(mod, 10, codeSection)
	return mod
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "algorithm", "wasm", "archive", name)
}

func abiBytesInput(data []byte) []byte {
	outLen := 64 + align32(len(data))
	out := make([]byte, outLen)
	binary.BigEndian.PutUint64(out[24:32], 32)
	binary.BigEndian.PutUint64(out[56:64], uint64(len(data)))
	copy(out[64:], data)
	return out
}

func abiBytesOutput(data []byte) []byte {
	outLen := 64 + align32(len(data))
	out := make([]byte, outLen)
	binary.BigEndian.PutUint64(out[24:32], 32)
	binary.BigEndian.PutUint64(out[56:64], uint64(len(data)))
	copy(out[64:], data)
	return out
}

func align32(length int) int {
	if length == 0 {
		return 0
	}
	return ((length + 31) / 32) * 32
}

func appendSection(dst []byte, id byte, payload []byte) []byte {
	dst = append(dst, id)
	dst = append(dst, encodeU32(uint32(len(payload)))...)
	dst = append(dst, payload...)
	return dst
}

func appendString(dst []byte, value string) []byte {
	dst = append(dst, encodeU32(uint32(len(value)))...)
	dst = append(dst, value...)
	return dst
}

func encodeU32(value uint32) []byte {
	var out []byte
	for {
		b := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		out = append(out, b)
		if value == 0 {
			break
		}
	}
	return out
}
