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
		Code:  "encoded-source",
		Gas:   12345,
		IType: "bytes,uint256",
		OType: "bool",
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
	wantKeys := map[string]struct{}{"code": {}, "gas": {}, "itype": {}, "otype": {}}
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
