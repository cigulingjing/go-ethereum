package network

import (
	"fmt"
	"strings"
)

// ComposeYAML 根据网络配置生成单服务器 Docker Compose 文件。
func ComposeYAML(cfg *Config) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("services:\n")
	for _, node := range cfg.Nodes {
		writeComposeService(&b, cfg, node)
	}
	b.WriteString("networks:\n")
	b.WriteString(fmt.Sprintf("  %s:\n", cfg.Docker.NetworkName))
	b.WriteString("    driver: bridge\n")
	return b.String(), nil
}

func writeComposeService(b *strings.Builder, cfg *Config, node NodeConfig) {
	service := node.ID
	b.WriteString(fmt.Sprintf("  %s:\n", service))
	b.WriteString(fmt.Sprintf("    image: %s\n", cfg.Docker.Image))
	b.WriteString(fmt.Sprintf("    container_name: %s-%s\n", cfg.Network.Name, node.ID))
	if cfg.Docker.CPUs != "" {
		b.WriteString(fmt.Sprintf("    cpus: %q\n", cfg.Docker.CPUs))
	}
	if cfg.Docker.Memory != "" {
		b.WriteString(fmt.Sprintf("    mem_limit: %q\n", cfg.Docker.Memory))
	}
	b.WriteString("    working_dir: /go-ethereum\n")
	b.WriteString("    entrypoint: []\n")
	b.WriteString("    environment:\n")
	b.WriteString("      GETH_CRYPTOUPGRADE_PLUGIN_DIR: /plugin\n")
	b.WriteString("      CRYPTOUPGRADE_MODULE: /go-ethereum\n")
	b.WriteString("    volumes:\n")
	b.WriteString(fmt.Sprintf("      - ./%s/datadir:/data\n", node.ID))
	b.WriteString(fmt.Sprintf("      - ./%s/plugin:/plugin\n", node.ID))
	b.WriteString(fmt.Sprintf("      - ./%s/config.toml:/config.toml:ro\n", node.ID))
	b.WriteString(fmt.Sprintf("      - ./%s/nodekey:/nodekey:ro\n", node.ID))
	b.WriteString(fmt.Sprintf("      - ./%s/password.txt:/password.txt:ro\n", node.ID))
	b.WriteString("      - ./genesis.json:/network/genesis.json:ro\n")
	if node.exposedToHost() {
		b.WriteString("    ports:\n")
		b.WriteString(fmt.Sprintf("      - \"%d:%d\"\n", node.HTTPHostPort, node.HTTPPort))
		if node.P2PHostPort > 0 {
			b.WriteString(fmt.Sprintf("      - \"%d:%d/tcp\"\n", node.P2PHostPort, node.P2PPort))
			b.WriteString(fmt.Sprintf("      - \"%d:%d/udp\"\n", node.P2PHostPort, node.P2PPort))
		}
	}
	args := GethArgs(cfg, node, "/data", "/plugin", "/nodekey", "/password.txt")
	b.WriteString("    command:\n")
	b.WriteString("      - /bin/sh\n")
	b.WriteString("      - -c\n")
	b.WriteString(fmt.Sprintf("      - %q\n", "geth init --datadir /data /network/genesis.json >/dev/null && exec geth "+shellJoin(args)))
	b.WriteString("    networks:\n")
	b.WriteString(fmt.Sprintf("      - %s\n", cfg.Docker.NetworkName))
}
