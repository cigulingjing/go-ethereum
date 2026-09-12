package rpc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
)

type stageTimingService struct{}

func (*stageTimingService) Call(ctx context.Context) string {
	id, _ := stagelog.Metadata(ctx)["requestId"].(string)
	return id
}

func TestStageTimingBatchCorrelation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stage.jsonl")
	t.Setenv(stagelog.FileEnv, path)
	t.Cleanup(stagelog.Close)
	server := NewServer()
	defer server.Stop()
	if err := server.RegisterName("eth", new(stageTimingService)); err != nil {
		t.Fatal(err)
	}
	client := DialInProc(server)
	defer client.Close()
	var first, second, single string
	batch := []BatchElem{{Method: "eth_call", Result: &first}, {Method: "eth_call", Result: &second}}
	if err := client.BatchCallContext(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	for _, elem := range batch {
		if elem.Error != nil {
			t.Fatal(elem.Error)
		}
	}
	if err := client.CallContext(context.Background(), &single, "eth_call"); err != nil {
		t.Fatal(err)
	}
	if first == "" || second == "" || single == "" || first == second || single == first || single == second {
		t.Fatal(first, second, single)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event["stage"] != "rpc_received" {
			t.Fatal(event)
		}
		seen[event["requestId"].(string)] = true
	}
	if len(seen) != 3 || !seen[first] || !seen[second] || !seen[single] {
		t.Fatal(seen)
	}
}
