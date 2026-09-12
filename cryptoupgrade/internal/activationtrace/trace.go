// Package activationtrace writes machine-readable node-local activation events.
package activationtrace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/repository"
)

const (
	TraceFileEnvVar = "GETH_CRYPTOUPGRADE_TRACE_FILE"
	DisableEnvVar   = "GETH_CRYPTOUPGRADE_TRACE_DISABLE"

	defaultTraceFile = "activation_trace.jsonl"
)

var writeMu sync.Mutex

type contextKey struct{}

// Event describes one node-local WASM upgrade activation stage.
type Event struct {
	Stage           string    `json:"stage"`
	Name            string    `json:"name,omitempty"`
	Version         uint64    `json:"version,omitempty"`
	ActivationBlock uint64    `json:"activationBlock,omitempty"`
	TxHash          string    `json:"txHash,omitempty"`
	BlockNumber     uint64    `json:"blockNumber,omitempty"`
	WasmHash        string    `json:"wasmHash,omitempty"`
	WasmPath        string    `json:"wasmPath,omitempty"`
	CompiledPath    string    `json:"compiledPath,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	DurationMillis  float64   `json:"durationMillis,omitempty"`
	Error           string    `json:"error,omitempty"`
}

// Enabled reports whether trace writing is enabled for the current process.
func Enabled() bool {
	raw := strings.TrimSpace(os.Getenv(DisableEnvVar))
	if raw == "" {
		return true
	}
	disabled, err := strconv.ParseBool(raw)
	return err != nil || !disabled
}

// Path returns the JSONL file used by the current process.
func Path() (string, bool) {
	if !Enabled() {
		return "", false
	}
	if path := strings.TrimSpace(os.Getenv(TraceFileEnvVar)); path != "" {
		return path, true
	}
	if strings.TrimSpace(os.Getenv(repository.PluginDirEnvVar)) == "" {
		return "", false
	}
	workspace, err := repository.ResolveWorkspaceFromEnvironment()
	if err != nil || strings.TrimSpace(workspace.BaseDir) == "" {
		return "", false
	}
	return filepath.Join(workspace.BaseDir, defaultTraceFile), true
}

// ContextWithEvent stores common event metadata for lower activation stages.
func ContextWithEvent(ctx context.Context, event Event) context.Context {
	return context.WithValue(ctx, contextKey{}, event)
}

// EventFromContext returns common event metadata recorded by the event adapter.
func EventFromContext(ctx context.Context) (Event, bool) {
	if ctx == nil {
		return Event{}, false
	}
	event, ok := ctx.Value(contextKey{}).(Event)
	return event, ok
}

// ForStage builds a trace event for stage, preserving context metadata.
func ForStage(ctx context.Context, stage string) Event {
	event, _ := EventFromContext(ctx)
	event.Stage = stage
	event.Timestamp = time.Time{}
	event.DurationMillis = 0
	event.Error = ""
	return event
}

// Record appends an event to the process trace file. It is intentionally
// best-effort at call sites so instrumentation cannot change activation semantics.
func Record(event Event) error {
	path, ok := Path()
	if !ok {
		return nil
	}
	return Write(path, event)
}

// Write appends one event as JSONL to path.
func Write(path string, event Event) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if event.Stage == "" {
		return fmt.Errorf("activation trace stage is empty")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	writeMu.Lock()
	defer writeMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(raw)
	return err
}
