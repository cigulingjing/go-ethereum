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
	testNodeKey3 = "0000000000000000000000000000000000000000000000000000000000000001"
	testSigner   = "0x7D81acf2C790a8133b53EB089C4E85629F0416Cf"
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
	if cfg.Docker.CPUs != "0.50" || cfg.Docker.Memory != "768m" {
		t.Fatalf("docker resources were not normalized: %#v", cfg.Docker)
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

func TestLoadConfigAcceptsInlinePrivateKey(t *testing.T) {
	dir := t.TempDir()
	raw := strings.Replace(testConfigYAML(dir), "privateKey: ./signer.key", "privateKey: "+testNodeKey1, 1)
	cfg, err := LoadConfig(writeConfig(t, dir, raw))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Accounts["signer"].PrivateKey != testNodeKey1 {
		t.Fatalf("unexpected inline private key: %q", cfg.Accounts["signer"].PrivateKey)
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

func TestStaticPeersStarTopology(t *testing.T) {
	dir := t.TempDir()
	writeNodeKey(t, filepath.Join(dir, "node1.key"), testNodeKey1)
	writeNodeKey(t, filepath.Join(dir, "node2.key"), testNodeKey2)
	writeNodeKey(t, filepath.Join(dir, "node3.key"), testNodeKey3)
	cfg := &Config{
		Network: NetworkConfig{PeerTopology: "star"},
		Nodes: []NodeConfig{
			{ID: "node1", Role: "signer", Host: "node1", AdvertiseHost: "node1", P2PPort: 30303, NodeKey: filepath.Join(dir, "node1.key")},
			{ID: "node2", Role: "observer", Host: "node2", AdvertiseHost: "node2", P2PPort: 30303, NodeKey: filepath.Join(dir, "node2.key")},
			{ID: "node3", Role: "observer", Host: "node3", AdvertiseHost: "node3", P2PPort: 30303, NodeKey: filepath.Join(dir, "node3.key")},
		},
	}
	peers, err := StaticPeers(cfg)
	if err != nil {
		t.Fatalf("StaticPeers failed: %v", err)
	}
	if len(peers["node1"]) != 2 {
		t.Fatalf("signer peers = %d, want 2: %#v", len(peers["node1"]), peers["node1"])
	}
	if len(peers["node2"]) != 1 || !strings.Contains(peers["node2"][0], "@node1:30303") {
		t.Fatalf("observer node2 should only peer with signer: %#v", peers["node2"])
	}
	if len(peers["node3"]) != 1 || !strings.Contains(peers["node3"][0], "@node1:30303") {
		t.Fatalf("observer node3 should only peer with signer: %#v", peers["node3"])
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
	if !strings.Contains(string(compose), "./node1/config.toml:/config.toml:ro") {
		t.Fatalf("compose must mount geth config.toml: %s", compose)
	}
	if got := strings.Count(string(compose), `cpus: "0.50"`); got != len(cfg.Nodes) {
		t.Fatalf("compose CPU limit count = %d, want %d: %s", got, len(cfg.Nodes), compose)
	}
	if got := strings.Count(string(compose), `mem_limit: "768m"`); got != len(cfg.Nodes) {
		t.Fatalf("compose memory limit count = %d, want %d: %s", got, len(cfg.Nodes), compose)
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
	if !strings.Contains(string(startScript), filepath.Join(out, "node1/nodekey")) {
		t.Fatalf("start script does not use node-local nodekey: %s", startScript)
	}
	keystoreDir := filepath.Join(artifacts.Nodes[0].Datadir, "keystore")
	entries, err := os.ReadDir(keystoreDir)
	if err != nil {
		t.Fatalf("expected rendered signer keystore dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one keystore file, got %d", len(entries))
	}
	keystore, err := os.ReadFile(filepath.Join(keystoreDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read rendered keystore: %v", err)
	}
	if !strings.Contains(string(keystore), strings.TrimPrefix(strings.ToLower(testSigner), "0x")) {
		t.Fatalf("rendered signer keystore has unexpected content: %s", keystore)
	}
	password, err := os.ReadFile(filepath.Join(filepath.Dir(artifacts.Nodes[0].Datadir), "password.txt"))
	if err != nil {
		t.Fatalf("read rendered password: %v", err)
	}
	if string(password) != "123456" {
		t.Fatalf("unexpected password file content: %q", password)
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
	writeNodeKey(tWriter{}, filepath.Join(dir, "signer.key"), testNodeKey1)
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
docker:
  cpus: "0.50"
  memory: 768m
  expose:
    - node: node1
      httpHostPort: 8661
cryptoupgrade:
  enabled: true
  addSource: ./add.go
accounts:
  signer:
    address: ` + testSigner + `
    privateKey: ./signer.key
    password: "123456"
nodes:
  - id: node1
    role: signer
    host: node1
    advertiseHost: node1
    p2pPort: 30303
    httpPort: 8545
    datadir: ./data/node1
    pluginDir: ./plugin/node1
    nodeKey: ./node1.key
    account: signer
  - id: node2
    role: observer
    host: node2
    advertiseHost: node2
    p2pPort: 30303
    httpPort: 8545
    datadir: ./data/node2
    pluginDir: ./plugin/node2
    nodeKey: ./node2.key
`
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
