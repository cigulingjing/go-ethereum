package activation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activationtrace"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
)

func TestStageTimingUpgradeSuccessAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "stage.jsonl")
			t.Setenv(stagelog.FileEnv, path)
			t.Cleanup(stagelog.Close)
			repo := &fakeRepository{active: map[string]model.AlgorithmInfo{}, sourcePath: filepath.Join(dir, "module.wasm"), pluginPath: filepath.Join(dir, "compiled")}
			svc := NewService(CodecFunc(func(_ string, path string) error { return os.WriteFile(path, []byte("wasm"), 0600) }), RuntimeFunc(func(context.Context, string, string) error {
				if fail {
					return errors.New("load failed")
				}
				return nil
			}), repo)
			ctx := activationtrace.ContextWithEvent(context.Background(), activationtrace.Event{TxHash: "0x123", BlockNumber: 8})
			err := svc.ActivateVersion(ctx, "Add", model.AlgorithmVersionInfo{AlgorithmInfo: model.AlgorithmInfo{Code: "encoded"}, Version: 2, ActivationBlock: 50})
			if (err != nil) != fail {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
			if len(lines) != 2 {
				t.Fatal(string(raw))
			}
			var first, last map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(lines[1]), &last); err != nil {
				t.Fatal(err)
			}
			want := "wasm_loaded"
			if fail {
				want = "wasm_upgrade_failed"
			}
			if first["stage"] != "wasm_upgrade_started" || last["stage"] != want || first["upgradeId"] != last["upgradeId"] {
				t.Fatal(first, last)
			}
			if last["txHash"] != "0x123" || last["version"] != float64(2) || last["activationBlock"] != float64(50) {
				t.Fatal(last)
			}
		})
	}
}
