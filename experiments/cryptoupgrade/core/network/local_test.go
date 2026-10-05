package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compactYAML(t *testing.T, dir string, count int) string {
	t.Helper()
	raw := strings.Split(testConfigYAML(dir), "nodes:\n")[0]
	return raw + fmt.Sprintf("local:\n  nodeCount: %d\n  outputDir: ./runtime\n  httpPortBase: 8761\n  expose:\n    - node: node1\n      httpHostPort: 8761\n", count)
}

func TestCompactNetworkCounts(t *testing.T) {
	for _, count := range []int{1, 20, 40} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			dir := t.TempDir()
			path := writeConfig(t, dir, compactYAML(t, dir, count))
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(cfg.Nodes) != count {
				t.Fatalf("nodes = %d", len(cfg.Nodes))
			}
			if _, err := os.Stat(filepath.Join(dir, "runtime")); !os.IsNotExist(err) {
				t.Fatalf("loading wrote runtime files: %v", err)
			}
			peers, err := StaticPeers(cfg)
			if err != nil {
				t.Fatal(err)
			}
			for i, node := range cfg.Nodes {
				wantHTTPHost := 0
				if i == 0 {
					wantHTTPHost = 8761
				}
				if node.HTTPHostPort != wantHTTPHost {
					t.Fatalf("unexpected expose ports: %+v", node)
				}
				if len(peers[node.ID]) != count-1 {
					t.Fatalf("wrong peers for %s", node.ID)
				}
				if (node.Role == "signer") != (i == 0) {
					t.Fatalf("wrong role for %s", node.ID)
				}
			}
			again, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			first, _ := EnodeForNode(cfg.Nodes[0])
			second, _ := EnodeForNode(again.Nodes[0])
			if first != second {
				t.Fatal("unstable node identity")
			}
			if _, err := BuildGenesis(cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCompactConfigRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(string) string
	}{
		{"zero", func(s string) string { return strings.Replace(s, "nodeCount: 20", "nodeCount: 0", 1) }},
		{"negative", func(s string) string { return strings.Replace(s, "nodeCount: 20", "nodeCount: -1", 1) }},
		{"overflow", func(s string) string {
			return strings.Replace(s, "httpHostPort: 8761", "httpHostPort: 8761\n    - node: node2\n      httpHostPort: 8761", 1)
		}},
		{"negative port", func(s string) string { return strings.Replace(s, "httpHostPort: 8761", "httpHostPort: -1", 1) }},
		{"mixed", func(s string) string { return s + "nodes:\n  - id: node1\n" }},
		{"missing account", func(s string) string { return s + "  signerAccount: missing\n" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if _, err := LoadConfig(writeConfig(t, dir, tc.edit(compactYAML(t, dir, 20)))); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCompactRenderSnapshot(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(writeConfig(t, dir, compactYAML(t, dir, 20)))
	if err != nil {
		t.Fatal(err)
	}
	wantPeers, err := StaticPeers(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "relocated")
	artifacts, err := Render(cfg, RenderOptions{OutputDir: out})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadConfig(artifacts.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Local != nil || len(snapshot.Nodes) != 20 {
		t.Fatal("snapshot must preserve expanded deployment")
	}
	for _, node := range snapshot.Nodes {
		if node.Datadir != filepath.Join(out, node.ID, "datadir") {
			t.Fatalf("incorrect datadir: %s", node.Datadir)
		}
		if _, err := os.Stat(node.NodeKey); err != nil {
			t.Fatal(err)
		}
	}
	gotPeers, err := StaticPeers(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(gotPeers) != fmt.Sprint(wantPeers) {
		t.Fatal("render changed peers")
	}
	if _, err := os.Stat(filepath.Join(snapshot.Nodes[0].Datadir, "keystore")); err != nil {
		t.Fatal(err)
	}
	// 默认输出路径来自 YAML，重复渲染不会另建规模目录。
	artifacts, err = Render(cfg, RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.OutputDir != filepath.Join(dir, "runtime") {
		t.Fatal(artifacts.OutputDir)
	}
}
