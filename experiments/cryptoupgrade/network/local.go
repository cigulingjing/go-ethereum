package network

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
)

// LocalConfig 用共享模板描述单机网络，避免手写重复的节点列表。
type LocalConfig struct {
	NodeCount     int               `yaml:"nodeCount"`
	OutputDir     string            `yaml:"outputDir"`
	HTTPPortBase  int               `yaml:"httpPortBase"`
	Expose        []HostExposeEntry `yaml:"expose,omitempty"`
	SignerAccount string            `yaml:"signerAccount"`
	ExtraArgs     []string          `yaml:"extraArgs,omitempty"`
}

func (cfg *Config) expandLocal() error {
	local := cfg.Local
	if local.NodeCount < 1 {
		return fmt.Errorf("local.nodeCount must be positive")
	}
	if local.HTTPPortBase == 0 {
		local.HTTPPortBase = 8761
	}
	if local.HTTPPortBase == 0 {
		local.HTTPPortBase = 8761
	}
	exposeCount := len(local.Expose)
	if exposeCount == 0 {
		exposeCount = 1
	}
	if local.HTTPPortBase < 1 || local.HTTPPortBase > 65535 ||
		exposeCount > 65536-local.HTTPPortBase {
		return fmt.Errorf("local httpPortBase and expose count must fit within 1..65535")
	}
	if local.SignerAccount == "" {
		local.SignerAccount = "signer"
	}
	account, ok := cfg.Accounts[local.SignerAccount]
	if !ok {
		return fmt.Errorf("local.signerAccount %q is not defined in accounts", local.SignerAccount)
	}
	if len(cfg.Consensus.Signers) == 0 {
		cfg.Consensus.Signers = []string{account.Address}
	}
	if local.OutputDir == "" {
		local.OutputDir = "runtime"
	}
	local.OutputDir = cfg.resolvePath(local.OutputDir)
	for i := 1; i <= local.NodeCount; i++ {
		id := fmt.Sprintf("node%d", i)
		root := filepath.Join(local.OutputDir, id)
		// 确定性公开测试密钥让重复 render 保持相同 enode；不用于生产网络。
		seed := sha256.Sum256([]byte("cryptoupgrade-local:" + cfg.Network.Name + ":" + id))
		n := new(big.Int).Sub(crypto.S256().Params().N, big.NewInt(1))
		scalar := new(big.Int).SetBytes(seed[:])
		scalar.Mod(scalar, n).Add(scalar, big.NewInt(1))
		node := NodeConfig{ID: id, Role: "observer", Datadir: filepath.Join(root, "datadir"),
			PluginDir: filepath.Join(root, "plugin"), NodeKey: filepath.Join(root, "nodekey"),
			ExtraArgs: append([]string(nil), local.ExtraArgs...), generatedKey: fmt.Sprintf("%064x", scalar)}
		if i == 1 {
			node.Role = "signer"
			node.Account = local.SignerAccount
		}
		cfg.Nodes = append(cfg.Nodes, node)
	}
	return nil
}
