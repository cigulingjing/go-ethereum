// Package activation coordinates the node-local activation of uploaded algorithms.
package activation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

// Codec decodes an uploaded source artifact to a repository-owned path.
type Codec interface {
	DecodeToFile(encoded, outputPath string) error
}

// Compiler builds a plugin from explicit source and output paths.
type Compiler interface {
	Compile(ctx context.Context, sourcePath, outputPath string) error
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

// Loader validates and activates a compiled plugin symbol.
type Loader interface {
	Load(pluginPath, symbolName string) error
}

// Service coordinates one complete local activation.
type Service struct {
	codec      Codec
	compiler   Compiler
	repository Repository
	loader     Loader
}

// NewService creates an activation service from injectable stage boundaries.
func NewService(codec Codec, compiler Compiler, repository Repository, loader Loader) *Service {
	return &Service{
		codec:      codec,
		compiler:   compiler,
		repository: repository,
		loader:     loader,
	}
}

// Activate decodes, compiles, persists and loads one uploaded algorithm.
func (s *Service) Activate(ctx context.Context, name string, info model.AlgorithmInfo) error {
	return s.ActivateVersion(ctx, name, model.LegacyVersion(info))
}

// ActivateVersion decodes, compiles, persists and loads one uploaded algorithm version.
func (s *Service) ActivateVersion(ctx context.Context, name string, info model.AlgorithmVersionInfo) error {
	if s == nil || s.codec == nil || s.compiler == nil || s.repository == nil || s.loader == nil {
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
	sourcePath := s.versionSourcePath(name, info.Version)
	if err := s.codec.DecodeToFile(info.Code, sourcePath); err != nil {
		return fmt.Errorf("decode algorithm %s version %d source: %w", name, info.Version, err)
	}
	pluginPath := s.versionPluginPath(name, info.Version)
	if err := s.compiler.Compile(ctx, sourcePath, pluginPath); err != nil {
		return fmt.Errorf("compile algorithm %s version %d: %w", name, info.Version, err)
	}

	previous, hadPrevious := s.repository.ActiveVersion(name)
	s.repository.SetActiveVersion(name, info)
	if err := s.repository.Save(); err != nil {
		s.restore(name, info.Version, previous, hadPrevious)
		return fmt.Errorf("persist algorithm %s version %d metadata: %w", name, info.Version, err)
	}
	if err := s.loader.Load(pluginPath, name); err != nil {
		s.restore(name, info.Version, previous, hadPrevious)
		// load 失败时恢复旧 metadata，避免 receipt 或持久化状态被误认为激活完成。
		if rollbackErr := s.repository.Save(); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("load algorithm %s version %d plugin: %w", name, info.Version, err),
				fmt.Errorf("rollback algorithm %s version %d metadata: %w", name, info.Version, rollbackErr),
			)
		}
		return fmt.Errorf("load algorithm %s version %d plugin: %w", name, info.Version, err)
	}
	return nil
}

func (s *Service) versionSourcePath(name string, version uint64) string {
	if version == 1 {
		return s.repository.SourcePath(name)
	}
	return s.repository.VersionSourcePath(name, version)
}

func (s *Service) versionPluginPath(name string, version uint64) string {
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

// CompilerFunc adapts a function to Compiler.
type CompilerFunc func(ctx context.Context, sourcePath, outputPath string) error

// Compile calls f.
func (f CompilerFunc) Compile(ctx context.Context, sourcePath, outputPath string) error {
	return f(ctx, sourcePath, outputPath)
}

// LoaderFunc adapts a function to Loader.
type LoaderFunc func(pluginPath, symbolName string) error

// Load calls f.
func (f LoaderFunc) Load(pluginPath, symbolName string) error {
	return f(pluginPath, symbolName)
}
