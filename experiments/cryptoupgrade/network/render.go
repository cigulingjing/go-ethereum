package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RenderOptions 控制多节点网络制品输出。
type RenderOptions struct {
	OutputDir string
	Geth      string
}

// Artifacts 记录一次 render 产生的关键路径。
type Artifacts struct {
	OutputDir   string              `json:"outputDir"`
	ConfigPath  string              `json:"configPath"`
	GenesisPath string              `json:"genesisPath"`
	ComposePath string              `json:"composePath"`
	Nodes       []NodeArtifact      `json:"nodes"`
	StaticPeers map[string][]string `json:"staticPeers"`
}

// NodeArtifact 记录单个节点的运行目录和启动脚本。
type NodeArtifact struct {
	ID              string `json:"id"`
	Datadir         string `json:"datadir"`
	PluginDir       string `json:"pluginDir"`
	GethConfigPath  string `json:"gethConfigPath"`
	StaticPeersPath string `json:"staticPeersPath"`
	StartScript     string `json:"startScript"`
}

// Render 生成 genesis、static peers、节点目录、启动脚本和 compose 文件。
// @file 生成文件放置在 build/cryptoupgrade-networks/ 目录下。
func Render(cfg *Config, opts RenderOptions) (*Artifacts, error) {
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join("build", "cryptoupgrade-networks", cfg.Network.Name)
	}
	if opts.Geth == "" {
		opts.Geth = "geth"
	}
	outDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	genesis, err := BuildGenesis(cfg)
	if err != nil {
		return nil, err
	}
	genesisJSON, err := MarshalGenesisJSON(genesis)
	if err != nil {
		return nil, err
	}
	genesisPath := filepath.Join(outDir, "genesis.json")
	if err := os.WriteFile(genesisPath, append(genesisJSON, '\n'), 0644); err != nil {
		return nil, err
	}
	configPath := filepath.Join(outDir, "network.yaml")
	if cfg.ConfigPath() != "" {
		if err := writeConfigSnapshot(cfg, configPath); err != nil {
			return nil, err
		}
	}
	staticPeers, err := StaticPeers(cfg)
	if err != nil {
		return nil, err
	}
	artifacts := &Artifacts{
		OutputDir:   outDir,
		ConfigPath:  configPath,
		GenesisPath: genesisPath,
		ComposePath: filepath.Join(outDir, "docker-compose.yml"),
		StaticPeers: staticPeers,
	}
	for _, node := range cfg.Nodes {
		nodeRoot := filepath.Join(outDir, "nodes", node.ID)
		datadir := filepath.Join(nodeRoot, "datadir")
		pluginDir := filepath.Join(nodeRoot, "plugin")
		if err := os.MkdirAll(datadir, 0755); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return nil, err
		}
		staticPath := filepath.Join(nodeRoot, "static-nodes.json")
		peerJSON, err := json.MarshalIndent(staticPeers[node.ID], "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(staticPath, append(peerJSON, '\n'), 0644); err != nil {
			return nil, err
		}
		gethConfigPath := filepath.Join(nodeRoot, "config.toml")
		if err := os.WriteFile(gethConfigPath, []byte(gethConfigTOML(staticPeers[node.ID])), 0644); err != nil {
			return nil, err
		}
		if err := copyOptionalFile(node.NodeKey, filepath.Join(nodeRoot, "nodekey"), 0600); err != nil {
			return nil, err
		}
		if node.Keystore != "" {
			if err := copyKeystore(node.Keystore, filepath.Join(datadir, "keystore")); err != nil {
				return nil, err
			}
		}
		if node.Password != "" {
			if err := copyOptionalFile(node.Password, filepath.Join(nodeRoot, "password.txt"), 0600); err != nil {
				return nil, err
			}
		}
		if _, err := os.Stat(filepath.Join(nodeRoot, "password.txt")); os.IsNotExist(err) {
			if err := os.WriteFile(filepath.Join(nodeRoot, "password.txt"), nil, 0600); err != nil {
				return nil, err
			}
		}
		startScript := filepath.Join(nodeRoot, "start.sh")
		if err := os.WriteFile(startScript, []byte(startScriptContent(cfg, node, opts.Geth, genesisPath, datadir, pluginDir)), 0755); err != nil {
			return nil, err
		}
		artifacts.Nodes = append(artifacts.Nodes, NodeArtifact{
			ID:              node.ID,
			Datadir:         datadir,
			PluginDir:       pluginDir,
			GethConfigPath:  gethConfigPath,
			StaticPeersPath: staticPath,
			StartScript:     startScript,
		})
	}
	compose, err := ComposeYAML(cfg)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(artifacts.ComposePath, []byte(compose), 0644); err != nil {
		return nil, err
	}
	artifactJSON, err := json.MarshalIndent(artifacts, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "artifacts.json"), append(artifactJSON, '\n'), 0644); err != nil {
		return nil, err
	}
	return artifacts, nil
}

func writeConfigSnapshot(cfg *Config, path string) error {
	// Render 输出目录与源 YAML 不同，直接复制相对路径会让快照再次加载时指向错误位置。
	// 写入已经归一化的配置，保证 addSource、keystore 和 nodekey 等输入仍可复现。
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal normalized network config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return fmt.Errorf("write normalized network config: %w", err)
	}
	return nil
}

// GethArgs 生成单个节点的 geth 启动参数。
func GethArgs(cfg *Config, node NodeConfig, datadir, pluginDir, nodeKeyPath, passwordPath string) []string {
	args := []string{
		"--config", filepath.Join(filepath.Dir(datadir), "config.toml"),
		"--datadir", datadir,
		"--networkid", fmt.Sprintf("%d", cfg.Network.NetworkID),
		"--port", fmt.Sprintf("%d", node.P2PPort),
		"--nodekey", nodeKeyPath,
		"--syncmode", "full",
		"--http",
		"--http.addr", "0.0.0.0",
		"--http.port", fmt.Sprintf("%d", node.HTTPPort),
		"--http.api", strings.Join(node.HTTPAPIs, ","),
		"--http.corsdomain", "*",
		"--http.vhosts", "*",
		"--ipcdisable",
		"--allow-insecure-unlock",
	}
	if node.Role == "signer" {
		args = append(args,
			"--mine",
			"--unlock", node.Account,
			"--password", passwordPath,
			"--miner.pending.feeRecipient", node.Account,
		)
	}
	args = append(args, node.ExtraArgs...)
	return args
}

func gethConfigTOML(staticPeers []string) string {
	var b strings.Builder
	b.WriteString("[Node.P2P]\n")
	b.WriteString("StaticNodes = [\n")
	for _, peer := range staticPeers {
		b.WriteString("  ")
		b.WriteString(tomlQuote(peer))
		b.WriteString(",\n")
	}
	b.WriteString("]\n")
	return b.String()
}

func tomlQuote(value string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}

func startScriptContent(cfg *Config, node NodeConfig, geth, genesisPath, datadir, pluginDir string) string {
	nodeRoot := filepath.Dir(datadir)
	nodeKeyPath := filepath.Join(nodeRoot, "nodekey")
	passwordPath := filepath.Join(nodeRoot, "password.txt")
	args := append([]string{geth}, GethArgs(cfg, node, datadir, pluginDir, nodeKeyPath, passwordPath)...)
	return fmt.Sprintf(`#!/bin/sh
set -eu
export GETH_CRYPTOUPGRADE_PLUGIN_DIR=%q
export CRYPTOUPGRADE_MODULE="${CRYPTOUPGRADE_MODULE:-$(pwd)}"
%q init --datadir %q %q >/dev/null
exec %s
`, pluginDir, geth, datadir, genesisPath, shellJoin(args))
}

func copyOptionalFile(src, dst string, perm os.FileMode) error {
	if src == "" {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return copyFile(src, dst, perm)
}

func copyKeystore(src, dstDir string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copy keystore %s: %w", src, err)
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("keystore path %s is not a regular file or directory", src)
		}
		return copyFile(src, filepath.Join(dstDir, filepath.Base(src)), 0600)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	// Geth keystore 目录是扁平文件集合，只复制 regular file，避免把临时目录嵌入 datadir。
	copied := false
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if !entryInfo.Mode().IsRegular() {
			continue
		}
		if err := copyFile(filepath.Join(src, entry.Name()), filepath.Join(dstDir, entry.Name()), 0600); err != nil {
			return err
		}
		copied = true
	}
	if !copied {
		return fmt.Errorf("keystore directory %s contains no key files", src)
	}
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, perm)
}

func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return r == '\'' || r == '"' || r == '\\' || r == '$' || r == ' ' || r == '*' || r == '\n' || r == '\t'
	}) == -1 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
