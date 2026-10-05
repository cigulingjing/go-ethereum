package network

import "fmt"

// HostExposeEntry 指定哪些节点向宿主机暴露端口；未列出的节点只在 Docker 子网内通信。
type HostExposeEntry struct {
	Node         string `yaml:"node"`
	HTTPHostPort int    `yaml:"httpHostPort,omitempty"`
	P2PHostPort  int    `yaml:"p2pHostPort,omitempty"`
}

func (cfg *Config) applyHostExpose() error {
	for i := range cfg.Nodes {
		cfg.Nodes[i].HTTPHostPort = 0
		cfg.Nodes[i].P2PHostPort = 0
	}
	entries := cfg.Docker.Expose
	if cfg.Local != nil && len(cfg.Local.Expose) > 0 {
		entries = cfg.Local.Expose
	}
	if len(entries) == 0 {
		if cfg.Local != nil {
			entries = []HostExposeEntry{{Node: "node1"}}
		} else {
			return nil
		}
	}
	portBase := 8761
	if cfg.Local != nil && cfg.Local.HTTPPortBase > 0 {
		portBase = cfg.Local.HTTPPortBase
	}
	nextPort := portBase
	for _, entry := range entries {
		idx := cfg.nodeIndex(entry.Node)
		if idx < 0 {
			return fmt.Errorf("expose node %q is not defined", entry.Node)
		}
		node := &cfg.Nodes[idx]
		httpHost := entry.HTTPHostPort
		if httpHost == 0 {
			httpHost = nextPort
			nextPort++
		} else if httpHost >= nextPort {
			nextPort = httpHost + 1
		}
		if httpHost < 1 || httpHost > 65535 {
			return fmt.Errorf("expose node %q has invalid httpHostPort %d", entry.Node, httpHost)
		}
		node.HTTPHostPort = httpHost
		node.P2PHostPort = entry.P2PHostPort
	}
	return nil
}

func (cfg *Config) nodeIndex(id string) int {
	for i, node := range cfg.Nodes {
		if node.ID == id {
			return i
		}
	}
	return -1
}

func (node NodeConfig) exposedToHost() bool {
	return node.HTTPHostPort > 0
}
