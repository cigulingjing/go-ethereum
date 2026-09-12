// Package activation coordinates the node-local activation of uploaded algorithms.
package activation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activationtrace"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmruntime"
	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
)

// Codec decodes an uploaded WASM artifact and can persist it to a repository-owned path.
type Codec interface {
	DecodeToFile(encoded, outputPath string) error
}

// Runtime compiles and instantiates a prepared WASM module.
type Runtime interface {
	Activate(ctx context.Context, wasmPath, compiledPath string) error
}

// Repository owns activation paths and active algorithm metadata.
type Repository interface {
	EnsureDirs() error
	SourcePath(name string) string
	PluginPath(name string) string
	VersionSourcePath(name string, version uint64) string
	VersionPluginPath(name string, version uint64) string
	Active(name string) (model.AlgorithmInfo, bool)
	SetActive(name string, info model.AlgorithmInfo)
	DeleteActive(name string)
	ActiveVersion(name string) (model.AlgorithmVersionInfo, bool)
	PreparedVersion(name string, version uint64) (model.AlgorithmVersionInfo, bool)
	SetActiveVersion(name string, info model.AlgorithmVersionInfo)
	DeleteActiveVersion(name string)
	DeletePreparedVersion(name string, version uint64)
	Save() error
}

// Service coordinates one complete local activation.
type Service struct {
	codec      Codec
	runtime    Runtime
	repository Repository
}

// NewService creates an activation service from injectable stage boundaries.
func NewService(codec Codec, runtime Runtime, repository Repository) *Service {
	return &Service{
		codec:      codec,
		runtime:    runtime,
		repository: repository,
	}
}

// Activate decodes, compiles, persists and loads one uploaded algorithm.
func (s *Service) Activate(ctx context.Context, name string, info model.AlgorithmInfo) error {
	return s.ActivateVersion(ctx, name, model.LegacyVersion(info))
}

// ActivateVersion decodes, compiles, persists and loads one uploaded algorithm version.
func (s *Service) ActivateVersion(ctx context.Context, name string, info model.AlgorithmVersionInfo) (activationErr error) {
	if s == nil || s.codec == nil || s.runtime == nil || s.repository == nil {
		return errors.New("activation service dependencies are incomplete")
	}
	name = model.NormalizeAlgorithmName(strings.TrimSpace(name))
	if name == "" {
		return errors.New("algorithm name is empty")
	}
	if info.Version == 0 {
		return errors.New("algorithm version is zero")
	}
	if current, ok := s.repository.PreparedVersion(name, info.Version); ok && current == info {
		return nil
	}
	activationStart := time.Now()
	var loadedAt time.Time
	if stagelog.Enabled() {
		source := activationtrace.ForStage(ctx, "")
		ctx = stagelog.With(ctx, stagelog.Fields{"flow": "upgrade", "algorithm": name, "version": info.Version,
			"activationBlock": info.ActivationBlock, "txHash": source.TxHash, "blockNumber": source.BlockNumber, "upgradeId": stagelog.NewID()})
		stagelog.RecordAt(ctx, "wasm_upgrade_started", activationStart, nil)
		defer func() {
			end := loadedAt
			if end.IsZero() {
				end = time.Now()
			}
			fields := stagelog.Fields{"durationNs": end.Sub(activationStart).Nanoseconds(), "success": activationErr == nil}
			stage := "wasm_loaded"
			if activationErr != nil {
				stage = "wasm_upgrade_failed"
				fields["error"] = activationErr.Error()
			}
			stagelog.RecordAt(ctx, stage, end, fields)
		}()
	}
	traceEvent := activationtrace.ForStage(ctx, "")
	traceEvent.Name = name
	traceEvent.Version = info.Version
	traceEvent.ActivationBlock = info.ActivationBlock
	ctx = activationtrace.ContextWithEvent(ctx, traceEvent)
	if err := s.repository.EnsureDirs(); err != nil {
		recordTrace(ctx, "activation_failed", 0, func(event *activationtrace.Event) {
			event.Error = err.Error()
		})
		return fmt.Errorf("prepare activation repository: %w", err)
	}
	wasmPath := s.versionSourcePath(name, info.Version)
	persistStart := time.Now()
	if err := s.codec.DecodeToFile(info.Code, wasmPath); err != nil {
		recordTrace(ctx, "activation_failed", 0, func(event *activationtrace.Event) {
			event.WasmPath = wasmPath
			event.Error = err.Error()
		})
		return fmt.Errorf("persist algorithm %s version %d wasm: %w", name, info.Version, err)
	}
	rawWasm, err := os.ReadFile(wasmPath)
	if err != nil {
		recordTrace(ctx, "activation_failed", 0, func(event *activationtrace.Event) {
			event.WasmPath = wasmPath
			event.Error = err.Error()
		})
		return fmt.Errorf("read persisted wasm %s: %w", wasmPath, err)
	}
	info.WasmHash = wasmHash(rawWasm)
	info.RuntimeName = wasmRuntimeName
	info.RuntimeVersion = wasmRuntimeVersion
	traceEvent.WasmHash = info.WasmHash
	traceEvent.WasmPath = wasmPath
	ctx = activationtrace.ContextWithEvent(ctx, traceEvent)
	recordTrace(ctx, "wasm_persisted", time.Since(persistStart), nil)

	previous, hadPrevious := s.repository.ActiveVersion(name)
	s.repository.SetActiveVersion(name, info)
	if err := s.repository.Save(); err != nil {
		s.restore(name, info.Version, previous, hadPrevious)
		recordTrace(ctx, "activation_failed", 0, func(event *activationtrace.Event) {
			event.Error = err.Error()
		})
		return fmt.Errorf("persist algorithm %s version %d metadata: %w", name, info.Version, err)
	}
	compiledPath := s.versionCompiledPath(name, info.Version)
	traceEvent.CompiledPath = compiledPath
	ctx = activationtrace.ContextWithEvent(ctx, traceEvent)
	if err := s.runtime.Activate(ctx, wasmPath, compiledPath); err != nil {
		s.restore(name, info.Version, previous, hadPrevious)
		recordTrace(ctx, "activation_failed", time.Since(activationStart), func(event *activationtrace.Event) {
			event.Error = err.Error()
		})
		// load 失败时恢复旧 metadata，避免 receipt 或持久化状态被误认为激活完成。
		if rollbackErr := s.repository.Save(); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("activate algorithm %s version %d wasm: %w", name, info.Version, err),
				fmt.Errorf("rollback algorithm %s version %d metadata: %w", name, info.Version, rollbackErr),
			)
		}
		return fmt.Errorf("activate algorithm %s version %d wasm: %w", name, info.Version, err)
	}
	loadedAt = time.Now()
	recordTrace(ctx, "activation_completed", time.Since(activationStart), nil)
	return nil
}

func recordTrace(ctx context.Context, stage string, duration time.Duration, mutate func(*activationtrace.Event)) {
	event := activationtrace.ForStage(ctx, stage)
	if duration > 0 {
		event.DurationMillis = float64(duration.Nanoseconds()) / float64(time.Millisecond)
	}
	if mutate != nil {
		mutate(&event)
	}
	_ = activationtrace.Record(event)
}

func (s *Service) versionSourcePath(name string, version uint64) string {
	if version == 1 {
		return s.repository.SourcePath(name)
	}
	return s.repository.VersionSourcePath(name, version)
}

func (s *Service) versionCompiledPath(name string, version uint64) string {
	if version == 1 {
		return s.repository.PluginPath(name)
	}
	return s.repository.VersionPluginPath(name, version)
}

func (s *Service) restore(name string, failedVersion uint64, previous model.AlgorithmVersionInfo, hadPrevious bool) {
	s.repository.DeletePreparedVersion(name, failedVersion)
	if hadPrevious {
		s.repository.SetActiveVersion(name, previous)
		return
	}
	s.repository.DeleteActiveVersion(name)
}

// CodecFunc adapts a function to Codec.
type CodecFunc func(encoded, outputPath string) error

// DecodeToFile calls f.
func (f CodecFunc) DecodeToFile(encoded, outputPath string) error {
	return f(encoded, outputPath)
}

// RuntimeFunc adapts a function to Runtime.
type RuntimeFunc func(ctx context.Context, wasmPath, compiledPath string) error

// Activate calls f.
func (f RuntimeFunc) Activate(ctx context.Context, wasmPath, compiledPath string) error {
	return f(ctx, wasmPath, compiledPath)
}

func wasmHash(rawWasm []byte) string {
	return crypto.Keccak256Hash(rawWasm).Hex()
}

const (
	wasmRuntimeName    = wasmruntime.RuntimeName
	wasmRuntimeVersion = wasmruntime.RuntimeVersion
)
