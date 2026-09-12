package network

import (
	"strings"
	"testing"
)

func TestComposeOnlyExposesConfiguredNodes(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(writeConfig(t, dir, testConfigYAML(dir)))
	if err != nil {
		t.Fatal(err)
	}
	compose, err := ComposeYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	text := string(compose)
	if !strings.Contains(text, `"8661:8545"`) {
		t.Fatalf("node1 should expose http port: %s", text)
	}
	if strings.Contains(text, "  node2:\n") {
		node2 := text[strings.Index(text, "  node2:"):]
		if end := strings.Index(node2, "  node"); end > 0 {
			node2 = node2[:end]
		}
		if strings.Contains(node2, "ports:") {
			t.Fatalf("node2 should not expose host ports: %s", node2)
		}
	}
	if !strings.Contains(text, "networks:\n  testnet:\n    driver: bridge") {
		t.Fatalf("compose must attach services to internal bridge network: %s", text)
	}
}
