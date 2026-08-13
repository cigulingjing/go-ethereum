package network

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// EnodeForNode 根据 nodekey 和 advertise host 生成节点 enode。
func EnodeForNode(node NodeConfig) (string, error) {
	if node.NodeKey == "" {
		return "", fmt.Errorf("node %s requires nodeKey path", node.ID)
	}
	raw, err := os.ReadFile(node.NodeKey)
	if err != nil {
		return "", err
	}
	keyHex := strings.TrimSpace(string(raw))
	keyHex = strings.TrimPrefix(keyHex, "0x")
	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return "", fmt.Errorf("parse nodekey for %s: %w", node.ID, err)
	}
	pub := crypto.FromECDSAPub(&key.PublicKey)
	if len(pub) != 65 || pub[0] != 4 {
		return "", fmt.Errorf("unexpected public key format for %s", node.ID)
	}
	host := node.AdvertiseHost
	if host == "" {
		host = node.Host
	}
	return fmt.Sprintf("enode://%s@%s:%d", hex.EncodeToString(pub[1:]), host, node.P2PPort), nil
}

// StaticPeers 为每个节点生成不包含自身的 static peer 列表。
func StaticPeers(cfg *Config) (map[string][]string, error) {
	enodes := make(map[string]string, len(cfg.Nodes))
	for _, node := range cfg.Nodes {
		enode, err := EnodeForNode(node)
		if err != nil {
			return nil, err
		}
		enodes[node.ID] = enode
	}
	peers := make(map[string][]string, len(cfg.Nodes))
	for _, node := range cfg.Nodes {
		for _, peer := range cfg.Nodes {
			if peer.ID == node.ID {
				continue
			}
			peers[node.ID] = append(peers[node.ID], enodes[peer.ID])
		}
	}
	return peers, nil
}
