package compiler

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var artifactMu sync.Mutex

// BuildContext 描述 Go Plugin 编译所需的明确构建环境。
type BuildContext struct {
	GoBinary   string
	WorkingDir string
	Env        []string
	BuildTags  []string
	Trimpath   bool
	CGOEnabled bool
	Stdout     io.Writer
	Stderr     io.Writer
}

// Compile 将明确的 Go 源码编译为 plugin，并原子发布到 outputPath。
func Compile(ctx context.Context, srcPath, outputPath string, build BuildContext) error {
	if strings.TrimSpace(srcPath) == "" {
		return fmt.Errorf("plugin source path is empty")
	}
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("plugin output path is empty")
	}
	if strings.TrimSpace(build.GoBinary) == "" {
		return fmt.Errorf("go binary is empty")
	}

	srcPath, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("resolve plugin source path: %w", err)
	}
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve plugin output path: %w", err)
	}

	artifactMu.Lock()
	defer artifactMu.Unlock()

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create plugin output directory %s: %w", outputDir, err)
	}
	tmp, err := os.CreateTemp(outputDir, "."+filepath.Base(outputPath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary plugin artifact: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temporary plugin artifact: %w", err)
	}
	if err := os.Remove(tmpPath); err != nil {
		return fmt.Errorf("prepare temporary plugin artifact: %w", err)
	}
	defer os.Remove(tmpPath)

	args := []string{"build", "-buildmode=plugin"}
	if len(build.BuildTags) > 0 {
		args = append(args, "-tags="+strings.Join(build.BuildTags, ","))
	}
	if build.Trimpath {
		args = append(args, "-trimpath")
	}
	args = append(args, "-o", tmpPath, srcPath)

	cmd := exec.CommandContext(ctx, build.GoBinary, args...)
	cmd.Dir = build.WorkingDir
	cmd.Env = append(os.Environ(), build.Env...)
	if build.CGOEnabled {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
	}
	cmd.Stdout = build.Stdout
	cmd.Stderr = build.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build plugin %s: %w", srcPath, err)
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("inspect built plugin artifact: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("built plugin artifact %s is not a non-empty regular file", tmpPath)
	}
	// 构建阶段不触碰旧制品，只有完整制品通过校验后才原子替换。
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("publish plugin artifact: %w", err)
	}
	return nil
}
