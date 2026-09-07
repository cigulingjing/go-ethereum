// Package activation coordinates the node-local activation of uploaded algorithms.
package activation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmruntime"
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
func (s *Service) ActivateVersion(ctx context.Context, name string, info model.AlgorithmVersionInfo) error {
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
	if err := s.repository.EnsureDirs(); err != nil {
		return fmt.Errorf("prepare activation repository: %w", err)
	}
	wasmPath := s.versionSourcePath(name, info.Version)
	if err := s.codec.DecodeToFile(info.Code, wasmPath); err != nil {
		return fmt.Errorf("persist algorithm %s version %d wasm: %w", name, info.Version, err)
	}
	rawWasm, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("read persisted wasm %s: %w", wasmPath, err)
	}
	info.WasmHash = wasmHash(rawWasm)
	info.RuntimeName = wasmRuntimeName
	info.RuntimeVersion = wasmRuntimeVersion

	previous, hadPrevious := s.repository.ActiveVersion(name)
	s.repository.SetActiveVersion(name, info)
	if err := s.repository.Save(); err != nil {
		s.restore(name, info.Version, previous, hadPrevious)
		return fmt.Errorf("persist algorithm %s version %d metadata: %w", name, info.Version, err)
	}
	compiledPath := s.versionCompiledPath(name, info.Version)
	if err := s.runtime.Activate(ctx, wasmPath, compiledPath); err != nil {
		s.restore(name, info.Version, previous, hadPrevious)
		// load 失败时恢复旧 metadata，避免 receipt 或持久化状态被误认为激活完成。
		if rollbackErr := s.repository.Save(); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("activate algorithm %s version %d wasm: %w", name, info.Version, err),
				fmt.Errorf("rollback algorithm %s version %d metadata: %w", name, info.Version, rollbackErr),
			)
		}
		return fmt.Errorf("activate algorithm %s version %d wasm: %w", name, info.Version, err)
	}
	return nil
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
