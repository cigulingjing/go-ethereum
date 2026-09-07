package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

func TestResolveWorkspaceDefault(t *testing.T) {
	cwd := t.TempDir()
	got, err := ResolveWorkspace("", cwd)
	if err != nil {
		t.Fatalf("ResolveWorkspace failed: %v", err)
	}
	want := NewWorkspace(filepath.Join(cwd, "plugin"))
	if got != want {
		t.Fatalf("unexpected workspace\nwant: %#v\n got: %#v", want, got)
	}
}

func TestResolveWorkspaceEnvironmentOverride(t *testing.T) {
	base := filepath.Join(t.TempDir(), "node-plugin")
	t.Setenv(PluginDirEnvVar, base)
	got, err := ResolveWorkspaceFromEnvironment()
	if err != nil {
		t.Fatalf("ResolveWorkspaceFromEnvironment failed: %v", err)
	}
	if want := NewWorkspace(base); got != want {
		t.Fatalf("unexpected workspace\nwant: %#v\n got: %#v", want, got)
	}
}

func TestRepositoryMetadataRoundTripPreservesSchema(t *testing.T) {
	workspace := NewWorkspace(filepath.Join(t.TempDir(), "plugin"))
	repo := New(workspace)
	want := model.AlgorithmInfo{
		Code:           "encoded-source",
		Gas:            12345,
		IType:          "bytes,uint256",
		OType:          "bool",
		WasmHash:       "0xabc",
		RuntimeName:    "wazero",
		RuntimeVersion: "v1.12.0",
	}
	repo.SetActive("Verify", want)
	if err := repo.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(workspace.AlgorithmInfoPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var schema map[string]map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("Unmarshal schema failed: %v", err)
	}
	keys := make(map[string]struct{})
	for key := range schema["Verify"] {
		keys[key] = struct{}{}
	}
	wantKeys := map[string]struct{}{"code": {}, "gas": {}, "itype": {}, "otype": {}, "wasmHash": {}, "runtimeName": {}, "runtimeVersion": {}}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("metadata schema changed: want %v got %v", wantKeys, keys)
	}

	loaded := New(workspace)
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got, ok := loaded.Active("Verify"); !ok || got != want {
		t.Fatalf("unexpected loaded metadata: %#v, %t", got, ok)
	}
}

func TestRepositoryVersionMetadataAndPathsAreIsolated(t *testing.T) {
	workspace := NewWorkspace(filepath.Join(t.TempDir(), "plugin"))
	repo := New(workspace)
	v1 := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "v1", Gas: 1, IType: "bytes", OType: "bytes", WasmHash: "0x1", RuntimeName: "wazero", RuntimeVersion: "v1.12.0"},
		Version:         1,
		ActivationBlock: 0,
	}
	v2 := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "v2", Gas: 2, IType: "bytes", OType: "bytes", WasmHash: "0x2", RuntimeName: "wazero", RuntimeVersion: "v1.12.0"},
		Version:         2,
		ActivationBlock: 42,
	}
	repo.SetUploadedVersion("Add", v1)
	repo.SetUploadedVersion("Add", v2)
	repo.SetActiveVersion("Add", v1)
	repo.SetActiveVersion("Add", v2)

	if repo.VersionSourcePath("Add", 2) == repo.SourcePath("Add") {
		t.Fatal("versioned source path reused legacy path")
	}
	if repo.VersionPluginPath("Add", 2) == repo.PluginPath("Add") {
		t.Fatal("versioned plugin path reused legacy path")
	}
	if got, ok := repo.UploadedVersion("Add", 2); !ok || got != v2 {
		t.Fatalf("unexpected uploaded version: %#v %t", got, ok)
	}
	if got, ok := repo.ActiveVersionAt("Add", 41); !ok || got.Version != 1 {
		t.Fatalf("unexpected active version before activation: %#v %t", got, ok)
	}
	if got, ok := repo.ActiveVersionAt("Add", 42); !ok || got.Version != 2 {
		t.Fatalf("unexpected active version after activation: %#v %t", got, ok)
	}
	if got, ok := repo.PreparedVersion("Add", 2); !ok || got != v2 {
		t.Fatalf("unexpected prepared version: %#v %t", got, ok)
	}
}

func TestRepositoryVersionMetadataRoundTrip(t *testing.T) {
	workspace := NewWorkspace(filepath.Join(t.TempDir(), "plugin"))
	repo := New(workspace)
	want := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "encoded-source", Gas: 12345, IType: "bytes", OType: "bytes", WasmHash: "0xabc", RuntimeName: "wazero", RuntimeVersion: "v1.12.0"},
		Version:         2,
		ActivationBlock: 42,
	}
	repo.SetActiveVersion("Verify", want)
	if err := repo.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded := New(workspace)
	if err := loaded.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got, ok := loaded.PreparedVersion("Verify", 2); !ok || got != want {
		t.Fatalf("unexpected loaded version metadata: %#v, %t", got, ok)
	}
	if got, ok := loaded.Active("Verify"); !ok || got != want.Base() {
		t.Fatalf("unexpected legacy active metadata: %#v, %t", got, ok)
	}
}

func TestRepositoryLoadInvalidJSONKeepsState(t *testing.T) {
	workspace := NewWorkspace(filepath.Join(t.TempDir(), "plugin"))
	if err := workspace.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}
	repo := New(workspace)
	want := model.AlgorithmInfo{Code: "existing", Gas: 7}
	repo.SetActive("Add", want)
	if err := os.WriteFile(workspace.AlgorithmInfoPath, []byte("{"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := repo.Load(); err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if got, ok := repo.Active("Add"); !ok || got != want {
		t.Fatalf("invalid load changed state: %#v, %t", got, ok)
	}
}

func TestRepositorySaveReplacesExistingFile(t *testing.T) {
	workspace := NewWorkspace(filepath.Join(t.TempDir(), "plugin"))
	if err := workspace.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}
	if err := os.WriteFile(workspace.AlgorithmInfoPath, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	repo := New(workspace)
	repo.SetActive("Add", model.AlgorithmInfo{Gas: 9})
	if err := repo.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if _, err := os.Stat(workspace.AlgorithmInfoPath); err != nil {
		t.Fatalf("metadata file missing after Save: %v", err)
	}
}
