package stagelog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stages.jsonl")
	t.Setenv(FileEnv, path)
	t.Setenv(DisableEnv, "false")
	t.Cleanup(Close)
	base := With(context.Background(), Fields{"txHash": "0x123"})
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := With(base, Fields{"executionId": NewID()})
			Record(ctx, "coprocessor_enter", nil)
			Record(ctx, "coprocessor_exit", Fields{"success": true})
		}()
	}
	wg.Wait()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 64 {
		t.Fatal(len(lines))
	}
	counts := map[string]int{}
	for _, line := range lines {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event["txHash"] != "0x123" || event["timestampUnixNano"] == nil || event["sessionId"] == nil {
			t.Fatal(event)
		}
		counts[event["executionId"].(string)]++
	}
	if len(counts) != 32 {
		t.Fatal(counts)
	}
	for _, count := range counts {
		if count != 2 {
			t.Fatal(count)
		}
	}
	if Metadata(base)["executionId"] != nil {
		t.Fatal("context metadata was mutated")
	}
}

func TestTimestampAndDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stages.jsonl")
	t.Setenv(FileEnv, path)
	t.Setenv(DisableEnv, "true")
	t.Cleanup(Close)
	Record(nil, "ignored", nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	t.Setenv(DisableEnv, "false")
	at := time.Unix(123, 456).UTC()
	RecordAt(nil, "boundary", at, nil)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var event map[string]any
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatal(err)
	}
	if event["timestampUnixNano"] != "123000000456" || event["time"] != at.Format(time.RFC3339Nano) {
		t.Fatal(event)
	}
}

func TestOutputFailureIsNonfatal(t *testing.T) {
	t.Setenv(FileEnv, t.TempDir())
	t.Setenv(DisableEnv, "false")
	t.Cleanup(Close)
	Record(nil, "boundary", nil)
	t.Setenv(FileEnv, filepath.Join(t.TempDir(), "recovered.jsonl"))
	Record(nil, "recovered", nil)
	if _, err := os.Stat(Path()); err != nil {
		t.Fatal(err)
	}
}
