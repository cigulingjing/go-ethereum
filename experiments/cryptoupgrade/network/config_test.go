package network

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testNodeKey1 = "4c0883a69102937d6231471b5dbb6204fe5129617082794e0ddb4f2b8e07cf4d"
	testNodeKey2 = "6c8759342fb1d3c387f6d580bf95313e540c5b8f53c87e1b16e0e5f9f5a6c3a2"
	testSigner   = "0x90F8bf6A479f320eAD074411a4B0e7944Ea8c9C1"
)

func TestLoadConfigNormalizesNetwork(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, testConfigYAML(dir))
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Network.ChainID != 11223344 {
		t.Fatalf("unexpected chain id: %d", cfg.Network.ChainID)
	}
	if len(cfg.Nodes) != 2 {
		t.Fatalf("unexpected node count: %d", len(cfg.Nodes))
	}
	if cfg.Nodes[0].Datadir != filepath.Join(dir, "data/node1") {
		t.Fatalf("datadir was not resolved relative to config: %s", cfg.Nodes[0].Datadir)
	}
	if cfg.Nodes[1].Role != "observer" {
		t.Fatalf("unexpected observer role: %s", cfg.Nodes[1].Role)
	}
}

func TestLoadConfigRejectsDuplicateNodeID(t *testing.T) {
	dir := t.TempDir()
	raw := strings.Replace(testConfigYAML(dir), "id: node2", "id: node1", 1)
	_, err := LoadConfig(writeConfig(t, dir, raw))
	if err == nil || !strings.Contains(err.Error(), "duplicate node id") {
		t.Fatalf("expected duplicate node id error, got %v", err)
	}
}

func TestLoadConfigRejectsMissingSigner(t *testing.T) {
	dir := t.TempDir()
	raw := strings.Replace(testConfigYAML(dir), "role: signer", "role: observer", 1)
	_, err := LoadConfig(writeConfig(t, dir, raw))
	if err == nil || !strings.Contains(err.Error(), "has no signer node") {
		t.Fatalf("expected missing signer node error, got %v", err)
	}
}

func TestLoadConfigRejectsPrivateKeyField(t *testing.T) {
	dir := t.TempDir()
	raw := testConfigYAML(dir) + "\nprivateKey: 0xabc\n"
	_, err := LoadConfig(writeConfig(t, dir, raw))
	if err == nil || !strings.Contains(err.Error(), "privateKey") {
		t.Fatalf("expected privateKey rejection, got %v", err)
	}
}

func TestBuildGenesisAndStaticPeers(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(writeConfig(t, dir, testConfigYAML(dir)))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	genesis, err := BuildGenesis(cfg)
	if err != nil {
		t.Fatalf("BuildGenesis failed: %v", err)
	}
	wantExtra := 32 + len(cfg.Consensus.Signers)*20 + 65
	if len(genesis.ExtraData) != wantExtra {
		t.Fatalf("unexpected clique extradata length: want %d got %d", wantExtra, len(genesis.ExtraData))
	}
	if genesis.Config.TerminalTotalDifficulty == nil || genesis.Config.TerminalTotalDifficulty.Sign() <= 0 {
		t.Fatalf("terminal total difficulty must keep Clique private chains out of merge-at-genesis mode: %v", genesis.Config.TerminalTotalDifficulty)
	}
	peers, err := StaticPeers(cfg)
	if err != nil {
		t.Fatalf("StaticPeers failed: %v", err)
	}
	if len(peers["node1"]) != 1 || !strings.HasPrefix(peers["node1"][0], "enode://") {
		t.Fatalf("unexpected node1 peers: %#v", peers["node1"])
	}
	if strings.Contains(peers["node1"][0], "@node1:") {
		t.Fatalf("node1 peer list contains itself: %#v", peers["node1"])
	}
}

func TestRenderArtifacts(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(writeConfig(t, dir, testConfigYAML(dir)))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	out := filepath.Join(dir, "out")
	artifacts, err := Render(cfg, RenderOptions{OutputDir: out})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if len(artifacts.Nodes) != 2 {
		t.Fatalf("unexpected rendered node count: %d", len(artifacts.Nodes))
	}
	snapshot, err := LoadConfig(artifacts.ConfigPath)
	if err != nil {
		t.Fatalf("reload rendered config snapshot: %v", err)
	}
	if snapshot.CryptoUpgrade.AddSource != cfg.CryptoUpgrade.AddSource {
		t.Fatalf("rendered addSource changed: want %s got %s", cfg.CryptoUpgrade.AddSource, snapshot.CryptoUpgrade.AddSource)
	}
	if _, err := os.Stat(snapshot.CryptoUpgrade.AddSource); err != nil {
		t.Fatalf("rendered addSource is not reusable: %v", err)
	}
	for _, path := range []string{artifacts.GenesisPath, artifacts.ComposePath, filepath.Join(out, "artifacts.json"), artifacts.Nodes[0].GethConfigPath, artifacts.Nodes[0].StaticPeersPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	compose, err := os.ReadFile(artifacts.ComposePath)
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}
	if !strings.Contains(string(compose), "entrypoint: []") {
		t.Fatalf("compose must clear image entrypoint before running shell command: %s", compose)
	}
	if !strings.Contains(string(compose), "./nodes/node1/config.toml:/config.toml:ro") {
		t.Fatalf("compose must mount geth config.toml: %s", compose)
	}
	gethConfig, err := os.ReadFile(artifacts.Nodes[0].GethConfigPath)
	if err != nil {
		t.Fatalf("read geth config: %v", err)
	}
	if !strings.Contains(string(gethConfig), "[Node.P2P]") || !strings.Contains(string(gethConfig), "StaticNodes") {
		t.Fatalf("geth config must contain static peers: %s", gethConfig)
	}
	if _, err := os.Stat(filepath.Join(artifacts.Nodes[0].Datadir, "geth", "static-nodes.json")); err == nil {
		t.Fatalf("deprecated datadir static-nodes.json should not be rendered")
	}
	startScript, err := os.ReadFile(artifacts.Nodes[0].StartScript)
	if err != nil {
		t.Fatalf("read start script: %v", err)
	}
	if !strings.Contains(string(startScript), filepath.Join(out, "nodes/node1/nodekey")) {
		t.Fatalf("start script does not use node-local nodekey: %s", startScript)
	}
	keystorePath := filepath.Join(artifacts.Nodes[0].Datadir, "keystore", testKeystoreFileName())
	keystore, err := os.ReadFile(keystorePath)
	if err != nil {
		t.Fatalf("expected rendered signer keystore %s: %v", keystorePath, err)
	}
	if !strings.Contains(string(keystore), strings.TrimPrefix(strings.ToLower(testSigner), "0x")) {
		t.Fatalf("rendered signer keystore has unexpected content: %s", keystore)
	}
}

func TestGethArgsSignerAndObserverContract(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(writeConfig(t, dir, testConfigYAML(dir)))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	signerArgs := strings.Join(GethArgs(cfg, cfg.Nodes[0], "/data", "/plugin", "/nodekey", "/password.txt"), " ")
	for _, want := range []string{"--config /config.toml", "--mine", "--unlock " + testSigner, "--password /password.txt", "--miner.pending.feeRecipient " + testSigner, "--allow-insecure-unlock", "--ipcdisable"} {
		if !strings.Contains(signerArgs, want) {
			t.Fatalf("signer args missing %q: %s", want, signerArgs)
		}
	}
	if strings.Contains(signerArgs, "--miner.pendingfeerecipient") {
		t.Fatalf("signer args contain unsupported fee-recipient spelling: %s", signerArgs)
	}
	if strings.Contains(signerArgs, "--bootnodes") {
		t.Fatalf("static peers should be configured through config.toml, not bootnodes: %s", signerArgs)
	}
	observerArgs := strings.Join(GethArgs(cfg, cfg.Nodes[1], "/data", "/plugin", "/nodekey", "/password.txt"), " ")
	for _, forbidden := range []string{"--mine", "--unlock", "--password /password.txt", "--miner.pending.feeRecipient"} {
		if strings.Contains(observerArgs, forbidden) {
			t.Fatalf("observer args contain signer-only flag %q: %s", forbidden, observerArgs)
		}
	}
}

func testConfigYAML(dir string) string {
	writeNodeKey(tWriter{}, filepath.Join(dir, "node1.key"), testNodeKey1)
	writeNodeKey(tWriter{}, filepath.Join(dir, "node2.key"), testNodeKey2)
	writeFile(tWriter{}, filepath.Join(dir, "keystore", testKeystoreFileName()), `{"address":"`+strings.TrimPrefix(strings.ToLower(testSigner), "0x")+`"}`+"\n")
	writeFile(tWriter{}, filepath.Join(dir, "password.txt"), "123456\n")
	writeFile(tWriter{}, filepath.Join(dir, "add.go"), "package algorithm\n")
	return `
network:
  name: testnet
  chainId: 11223344
consensus:
  type: clique
  period: 1
  epoch: 30000
  signers:
    - ` + testSigner + `
cryptoupgrade:
  enabled: true
  addSource: ./add.go
accounts:
  signer:
    address: ` + testSigner + `
    keystore: ./keystore
    password: ./password.txt
nodes:
  - id: node1
    role: signer
    host: node1
    advertiseHost: node1
    p2pPort: 30303
    p2pHostPort: 30303
    httpPort: 8545
    httpHostPort: 8661
    datadir: ./data/node1
    pluginDir: ./plugin/node1
    nodeKey: ./node1.key
    account: signer
  - id: node2
    role: observer
    host: node2
    advertiseHost: node2
    p2pPort: 30303
    p2pHostPort: 30304
    httpPort: 8545
    httpHostPort: 8662
    datadir: ./data/node2
    pluginDir: ./plugin/node2
    nodeKey: ./node2.key
`
}

func testKeystoreFileName() string {
	return "UTC--2026-08-11T00-00-00.000000000Z--" + strings.TrimPrefix(strings.ToLower(testSigner), "0x")
}

type tWriter struct{}

func writeConfig(t *testing.T, dir, raw string) string {
	path := filepath.Join(dir, "network.yaml")
	writeFile(t, path, raw)
	return path
}

func writeNodeKey(t interface{ Fatalf(string, ...interface{}) }, path, raw string) {
	writeFile(t, path, raw+"\n")
}

func writeFile(t interface{ Fatalf(string, ...interface{}) }, path, raw string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func (tWriter) Fatalf(format string, args ...interface{}) {
	panic("unexpected test helper failure")
}
