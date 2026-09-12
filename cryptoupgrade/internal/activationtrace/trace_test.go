package activationtrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activation_trace.jsonl")
	when := time.Date(2026, 9, 7, 12, 0, 0, 123, time.UTC)
	if err := Write(path, Event{
		Stage:          "wasm_compiled",
		Name:           "Add",
		Version:        1,
		TxHash:         "0xabc",
		BlockNumber:    7,
		WasmHash:       "0x123",
		DurationMillis: 1.25,
		Timestamp:      when,
	}); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	var event Event
	if err := json.Unmarshal(raw[:len(raw)-1], &event); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	if event.Stage != "wasm_compiled" || event.Name != "Add" || event.Version != 1 || event.TxHash != "0xabc" || event.BlockNumber != 7 {
		t.Fatalf("unexpected event: %#v", event)
	}
	if event.WasmHash != "0x123" || event.DurationMillis != 1.25 || !event.Timestamp.Equal(when) {
		t.Fatalf("unexpected trace details: %#v", event)
	}
}

func TestRecordDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activation_trace.jsonl")
	t.Setenv(TraceFileEnvVar, path)
	t.Setenv(DisableEnvVar, "true")
	if err := Record(Event{Stage: "activation_started", Name: "Add"}); err != nil {
		t.Fatalf("Record failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("disabled trace wrote file or unexpected stat error: %v", err)
	}
}

func TestRecordWithoutConfiguredPathIsNoop(t *testing.T) {
	t.Setenv(TraceFileEnvVar, "")
	t.Setenv(DisableEnvVar, "")
	t.Setenv("GETH_CRYPTOUPGRADE_PLUGIN_DIR", "")
	if err := Record(Event{Stage: "activation_started", Name: "Add"}); err != nil {
		t.Fatalf("Record failed: %v", err)
	}
}
