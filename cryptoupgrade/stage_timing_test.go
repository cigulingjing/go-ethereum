package cryptoupgrade

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
)

func TestStageTimingWASMCall(t *testing.T) {
	dir := t.TempDir()
	withRuntimePluginPaths(t, newPluginPaths(filepath.Join(dir, "plugin")), nil)
	path := filepath.Join(dir, "stage.jsonl")
	t.Setenv(stagelog.FileEnv, path)
	t.Cleanup(stagelog.Close)
	code, err := EncodeWasmFile("algorithm/wasm/archive/add.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if err := ActivateAlgorithm("TimingAdd", algoInfo{Code: code, Gas: 7, IType: "int256,int256", OType: "int256"}); err != nil {
		t.Fatal(err)
	}
	input := make([]byte, 64)
	input[31] = 3
	input[63] = 4
	call, err := CodeStorageABI.Pack("callFunc", "TimingAdd", input)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"request-a", "request-b"} {
		ctx := stagelog.With(context.Background(), stagelog.Fields{"requestId": id, "phase": "simulation"})
		result, err := RunCodeStorageCallAtContext(ctx, call, 1, true, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 32 || result[31] != 7 {
			t.Fatalf("unexpected addition output %x", result)
		}
	}
	ctx := stagelog.With(nil, stagelog.Fields{"requestId": "failed"})
	if _, _, err := callUpgradeAlgoWithInfoContext(ctx, "Missing", filepath.Join(dir, "missing.wasm"), filepath.Join(dir, "missing-cache"), 7, nil, algoInfo{Gas: 7}); err == nil {
		t.Fatal("missing WASM succeeded")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	executions := map[string]string{}
	exits := 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if e["stage"] == "coprocessor_enter" {
			executions[e["requestId"].(string)] = e["executionId"].(string)
		}
		if e["stage"] == "coprocessor_exit" {
			id := e["requestId"].(string)
			if e["executionId"] != executions[id] || e["durationNs"].(float64) < 0 {
				t.Fatal(e)
			}
			if e["success"] != (id != "failed") {
				t.Fatal(e)
			}
			exits++
		}
	}
	if len(executions) != 3 || exits != 3 || executions["request-a"] == executions["request-b"] {
		t.Fatal(executions, exits)
	}
}
