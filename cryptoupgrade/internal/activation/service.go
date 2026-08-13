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
	Active(name string) (model.AlgorithmInfo, bool)
	SetActive(name string, info model.AlgorithmInfo)
	DeleteActive(name string)
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
	if s == nil || s.codec == nil || s.compiler == nil || s.repository == nil || s.loader == nil {
		return errors.New("activation service dependencies are incomplete")
	}
	name = model.NormalizeAlgorithmName(strings.TrimSpace(name))
	if name == "" {
		return errors.New("algorithm name is empty")
	}
	if current, ok := s.repository.Active(name); ok && current == info {
		return nil
	}
	if err := s.repository.EnsureDirs(); err != nil {
		return fmt.Errorf("prepare activation repository: %w", err)
	}
	sourcePath := s.repository.SourcePath(name)
	if err := s.codec.DecodeToFile(info.Code, sourcePath); err != nil {
		return fmt.Errorf("decode algorithm %s source: %w", name, err)
	}
	pluginPath := s.repository.PluginPath(name)
	if err := s.compiler.Compile(ctx, sourcePath, pluginPath); err != nil {
		return fmt.Errorf("compile algorithm %s: %w", name, err)
	}

	previous, hadPrevious := s.repository.Active(name)
	s.repository.SetActive(name, info)
	if err := s.repository.Save(); err != nil {
		s.restore(name, previous, hadPrevious)
		return fmt.Errorf("persist algorithm %s metadata: %w", name, err)
	}
	if err := s.loader.Load(pluginPath, name); err != nil {
		s.restore(name, previous, hadPrevious)
		// load 失败时恢复旧 metadata，避免 receipt 或持久化状态被误认为激活完成。
		if rollbackErr := s.repository.Save(); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("load algorithm %s plugin: %w", name, err),
				fmt.Errorf("rollback algorithm %s metadata: %w", name, rollbackErr),
			)
		}
		return fmt.Errorf("load algorithm %s plugin: %w", name, err)
	}
	return nil
}

func (s *Service) restore(name string, previous model.AlgorithmInfo, hadPrevious bool) {
	if hadPrevious {
		s.repository.SetActive(name, previous)
		return
	}
	s.repository.DeleteActive(name)
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
