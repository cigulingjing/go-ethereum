package network

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

const (
	defaultNetworkName   = "cryptoupgrade-local"
	defaultChainID       = uint64(11223344)
	defaultP2PPort       = 30303
	defaultHTTPPort      = 8545
	defaultCliquePeriod  = uint64(5)
	defaultCliqueEpoch   = uint64(30000)
	defaultGasLimit      = uint64(8000000)
	defaultDockerImage   = "cryptoupgrade-geth:lab"
	defaultFundedBalance = "30000000000000000000000"
)

// Config 描述一个可复现的 cryptoupgrade 多节点私链网络。
type Config struct {
	Network       NetworkConfig            `yaml:"network"`
	Consensus     ConsensusConfig          `yaml:"consensus"`
	Docker        DockerConfig             `yaml:"docker"`
	CryptoUpgrade CryptoUpgradeConfig      `yaml:"cryptoupgrade"`
	Accounts      map[string]AccountConfig `yaml:"accounts"`
	Nodes         []NodeConfig             `yaml:"nodes"`

	configPath string
	baseDir    string
}

// NetworkConfig 定义所有节点共享的链参数。
type NetworkConfig struct {
	Name      string `yaml:"name"`
	ChainID   uint64 `yaml:"chainId"`
	NetworkID uint64 `yaml:"networkId"`
	GasLimit  uint64 `yaml:"gasLimit"`
}

// ConsensusConfig 定义私链共识参数。
type ConsensusConfig struct {
	Type    string   `yaml:"type"`
	Period  uint64   `yaml:"period"`
	Epoch   uint64   `yaml:"epoch"`
	Signers []string `yaml:"signers"`
}

// DockerConfig 定义单机容器网络的默认镜像和 Docker network 名称。
type DockerConfig struct {
	Image       string `yaml:"image"`
	NetworkName string `yaml:"networkName"`
}

// CryptoUpgradeConfig 控制是否在 genesis 中保留 cryptoupgrade 实验账户。
type CryptoUpgradeConfig struct {
	Enabled   *bool  `yaml:"enabled"`
	SmokeTest bool   `yaml:"smokeTest"`
	AddSource string `yaml:"addSource"`
}

// AccountConfig 通过外部文件引用账户密钥，避免在 YAML 中保存私钥明文。
type AccountConfig struct {
	Address    string `yaml:"address"`
	Keystore   string `yaml:"keystore"`
	Password   string `yaml:"password"`
	Balance    string `yaml:"balance"`
	PrivateKey string `yaml:"privateKey"`
}

// NodeConfig 定义单个 geth 节点的网络地址、目录和角色。
type NodeConfig struct {
	ID            string   `yaml:"id"`
	Name          string   `yaml:"name"`
	Role          string   `yaml:"role"`
	Host          string   `yaml:"host"`
	AdvertiseHost string   `yaml:"advertiseHost"`
	P2PPort       int      `yaml:"p2pPort"`
	P2PHostPort   int      `yaml:"p2pHostPort"`
	HTTPPort      int      `yaml:"httpPort"`
	HTTPHostPort  int      `yaml:"httpHostPort"`
	RPCURL        string   `yaml:"rpcURL"`
	Datadir       string   `yaml:"datadir"`
	PluginDir     string   `yaml:"pluginDir"`
	NodeKey       string   `yaml:"nodeKey"`
	Account       string   `yaml:"account"`
	Keystore      string   `yaml:"keystore"`
	Password      string   `yaml:"password"`
	HTTPAPIs      []string `yaml:"httpApis"`
	ExtraArgs     []string `yaml:"extraArgs"`
	PrivateKey    string   `yaml:"privateKey"`
}

// LoadConfig 读取并归一化多节点网络配置。
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := rejectPrivateKeyFields(raw); err != nil {
		return nil, err
	}
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	cfg.configPath = abs
	cfg.baseDir = filepath.Dir(abs)
	if err := cfg.Normalize(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Normalize 填充默认值并把相对路径解析到配置文件所在目录。
func (cfg *Config) Normalize() error {
	if cfg.baseDir == "" {
		cfg.baseDir = "."
	}
	if cfg.Network.Name == "" {
		cfg.Network.Name = defaultNetworkName
	}
	if cfg.Network.ChainID == 0 {
		cfg.Network.ChainID = defaultChainID
	}
	if cfg.Network.NetworkID == 0 {
		cfg.Network.NetworkID = cfg.Network.ChainID
	}
	if cfg.Network.GasLimit == 0 {
		cfg.Network.GasLimit = defaultGasLimit
	}
	cfg.Consensus.Type = strings.ToLower(strings.TrimSpace(cfg.Consensus.Type))
	if cfg.Consensus.Type == "" {
		cfg.Consensus.Type = "clique"
	}
	if cfg.Consensus.Period == 0 {
		cfg.Consensus.Period = defaultCliquePeriod
	}
	if cfg.Consensus.Epoch == 0 {
		cfg.Consensus.Epoch = defaultCliqueEpoch
	}
	if cfg.Docker.Image == "" {
		cfg.Docker.Image = defaultDockerImage
	}
	if cfg.Docker.NetworkName == "" {
		cfg.Docker.NetworkName = cfg.Network.Name
	}
	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]AccountConfig)
	}
	if cfg.CryptoUpgrade.AddSource == "" {
		cfg.CryptoUpgrade.AddSource = "cryptoupgrade/algorithm/go/add.go"
	}
	for i := range cfg.Nodes {
		node := &cfg.Nodes[i]
		node.ID = strings.TrimSpace(node.ID)
		if node.Name == "" {
			node.Name = node.ID
		}
		node.Role = strings.ToLower(strings.TrimSpace(node.Role))
		if node.Role == "" {
			node.Role = "observer"
		}
		if node.Host == "" {
			node.Host = node.ID
		}
		if node.AdvertiseHost == "" {
			node.AdvertiseHost = node.Host
		}
		if node.P2PPort == 0 {
			node.P2PPort = defaultP2PPort
		}
		if node.HTTPPort == 0 {
			node.HTTPPort = defaultHTTPPort
		}
		if node.P2PHostPort == 0 {
			node.P2PHostPort = node.P2PPort
		}
		if node.HTTPHostPort == 0 {
			node.HTTPHostPort = node.HTTPPort
		}
		if len(node.HTTPAPIs) == 0 {
			node.HTTPAPIs = []string{"web3", "eth", "net", "admin", "debug", "clique", "txpool", "miner"}
		}
		if node.Datadir == "" {
			node.Datadir = filepath.Join("nodes", node.ID, "datadir")
		}
		if node.PluginDir == "" {
			node.PluginDir = filepath.Join("nodes", node.ID, "plugin")
		}
		if account, ok := cfg.Accounts[node.Account]; ok {
			node.Account = account.Address
			if node.Keystore == "" {
				node.Keystore = account.Keystore
			}
			if node.Password == "" {
				node.Password = account.Password
			}
		}
		node.Datadir = cfg.resolvePath(node.Datadir)
		node.PluginDir = cfg.resolvePath(node.PluginDir)
		node.NodeKey = cfg.resolvePath(node.NodeKey)
		node.Keystore = cfg.resolvePath(node.Keystore)
		node.Password = cfg.resolvePath(node.Password)
	}
	for name, account := range cfg.Accounts {
		account.Keystore = cfg.resolvePath(account.Keystore)
		account.Password = cfg.resolvePath(account.Password)
		cfg.Accounts[name] = account
	}
	cfg.CryptoUpgrade.AddSource = cfg.resolvePath(cfg.CryptoUpgrade.AddSource)
	return cfg.Validate()
}

// Validate 检查配置是否满足多节点私链启动的最低要求。
func (cfg *Config) Validate() error {
	if len(cfg.Nodes) == 0 {
		return fmt.Errorf("network must define at least one node")
	}
	if cfg.Consensus.Type != "clique" {
		return fmt.Errorf("unsupported consensus type %q", cfg.Consensus.Type)
	}
	if len(cfg.Consensus.Signers) == 0 {
		return fmt.Errorf("clique consensus requires at least one signer")
	}
	seenNodes := make(map[string]struct{})
	seenHTTPPorts := make(map[int]string)
	seenP2PPorts := make(map[int]string)
	signerNodes := make(map[string]struct{})
	for i, signer := range cfg.Consensus.Signers {
		if !common.IsHexAddress(signer) {
			return fmt.Errorf("consensus signer %d has invalid address %q", i, signer)
		}
		cfg.Consensus.Signers[i] = common.HexToAddress(signer).Hex()
	}
	for _, node := range cfg.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node id is required")
		}
		if _, ok := seenNodes[node.ID]; ok {
			return fmt.Errorf("duplicate node id %q", node.ID)
		}
		seenNodes[node.ID] = struct{}{}
		if owner, ok := seenHTTPPorts[node.HTTPHostPort]; ok {
			return fmt.Errorf("duplicate http host port %d on nodes %s and %s", node.HTTPHostPort, owner, node.ID)
		}
		seenHTTPPorts[node.HTTPHostPort] = node.ID
		if owner, ok := seenP2PPorts[node.P2PHostPort]; ok {
			return fmt.Errorf("duplicate p2p host port %d on nodes %s and %s", node.P2PHostPort, owner, node.ID)
		}
		seenP2PPorts[node.P2PHostPort] = node.ID
		if node.NodeKey == "" {
			return fmt.Errorf("node %s requires nodeKey path", node.ID)
		}
		if node.Role == "signer" {
			if !common.IsHexAddress(node.Account) {
				return fmt.Errorf("signer node %s requires account address", node.ID)
			}
			if node.Keystore == "" || node.Password == "" {
				return fmt.Errorf("signer node %s requires keystore and password paths", node.ID)
			}
			signerNodes[common.HexToAddress(node.Account).Hex()] = struct{}{}
		}
		if node.Role != "signer" && node.Role != "observer" && node.Role != "rpc" {
			return fmt.Errorf("node %s has unsupported role %q", node.ID, node.Role)
		}
	}
	for _, signer := range cfg.Consensus.Signers {
		if _, ok := signerNodes[common.HexToAddress(signer).Hex()]; !ok {
			return fmt.Errorf("clique signer %s has no signer node", signer)
		}
	}
	return nil
}

// ConfigPath 返回配置文件绝对路径。
func (cfg *Config) ConfigPath() string {
	return cfg.configPath
}

// CryptoUpgradeEnabled 返回是否在 genesis 中创建 cryptoupgrade 实验账户。
func (cfg *Config) CryptoUpgradeEnabled() bool {
	return cfg.CryptoUpgrade.Enabled == nil || *cfg.CryptoUpgrade.Enabled
}

func (cfg *Config) resolvePath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(cfg.baseDir, path))
}

func rejectPrivateKeyFields(raw []byte) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	var walk func(*yaml.Node) error
	walk = func(n *yaml.Node) error {
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				key := n.Content[i].Value
				switch strings.ToLower(strings.ReplaceAll(key, "_", "")) {
				case "privatekey", "secretkey":
					return fmt.Errorf("yaml field %q is not allowed; use keystore/password or nodeKey file paths", key)
				}
				if err := walk(n.Content[i+1]); err != nil {
					return err
				}
			}
		}
		for _, child := range n.Content {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(&doc)
}
