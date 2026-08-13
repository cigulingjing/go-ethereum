//go:build !windows

package compiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompilePublishesArtifactWithCompatibleBuildContext(t *testing.T) {
	dir := t.TempDir()
	goBin := writeFakeGo(t, dir)
	src := filepath.Join(dir, "Add.go")
	output := filepath.Join(dir, "Add.so")
	logPath := filepath.Join(dir, "build.log")
	if err := os.WriteFile(src, []byte("package main"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	err := Compile(context.Background(), src, output, BuildContext{
		GoBinary:   goBin,
		WorkingDir: dir,
		Env:        []string{"FAKE_GO_LOG=" + logPath},
		BuildTags:  []string{"urfave_cli_no_docs", "ckzg"},
		Trimpath:   true,
		CGOEnabled: true,
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "fake-plugin\n" {
		t.Fatalf("unexpected artifact: %q", data)
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read fake go log: %v", err)
	}
	logText := string(logData)
	for _, want := range []string{
		"build -buildmode=plugin",
		"-tags=urfave_cli_no_docs,ckzg",
		"-trimpath",
		"CGO_ENABLED=1",
		"PWD=" + dir,
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("build context missing %q in %q", want, logText)
		}
	}
}

func TestCompileFailureKeepsOldArtifact(t *testing.T) {
	dir := t.TempDir()
	goBin := writeFakeGo(t, dir)
	src := filepath.Join(dir, "Add.go")
	output := filepath.Join(dir, "Add.so")
	if err := os.WriteFile(src, []byte("package main"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(output, []byte("old-plugin"), 0644); err != nil {
		t.Fatalf("write old artifact: %v", err)
	}

	err := Compile(context.Background(), src, output, BuildContext{
		GoBinary: goBin,
		Env:      []string{"FAKE_GO_FAIL=1"},
	})
	if err == nil {
		t.Fatal("expected Compile to fail")
	}
	data, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatalf("read old artifact: %v", readErr)
	}
	if string(data) != "old-plugin" {
		t.Fatalf("failed build replaced old artifact: %q", data)
	}
}

func TestCompileRejectsMissingBuildOutput(t *testing.T) {
	dir := t.TempDir()
	goBin := writeFakeGo(t, dir)
	src := filepath.Join(dir, "Add.go")
	output := filepath.Join(dir, "Add.so")
	if err := os.WriteFile(src, []byte("package main"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	err := Compile(context.Background(), src, output, BuildContext{
		GoBinary: goBin,
		Env:      []string{"FAKE_GO_SKIP_OUTPUT=1"},
	})
	if err == nil {
		t.Fatal("expected missing output error")
	}
}

func writeFakeGo(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "fake-go")
	script := `#!/bin/sh
set -eu
if [ "${FAKE_GO_LOG:-}" != "" ]; then
  printf '%s\n' "$*" "CGO_ENABLED=${CGO_ENABLED:-}" "PWD=$PWD" > "$FAKE_GO_LOG"
fi
if [ "${FAKE_GO_FAIL:-}" = "1" ]; then
  exit 1
fi
output=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    output="$1"
    break
  fi
  shift
done
if [ "${FAKE_GO_SKIP_OUTPUT:-}" != "1" ]; then
  printf 'fake-plugin\n' > "$output"
fi
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}
	return path
}
