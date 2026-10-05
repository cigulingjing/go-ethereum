package network

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	zeroBlock = big.NewInt(0)
)

// BuildGenesis 根据网络配置生成 Clique 私链 genesis。
func BuildGenesis(cfg *Config) (*core.Genesis, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	extra, err := cliqueExtraData(cfg.Consensus.Signers)
	if err != nil {
		return nil, err
	}
	genesis := &core.Genesis{
		Config:     cliqueChainConfig(cfg),
		Difficulty: big.NewInt(1),
		GasLimit:   cfg.Network.GasLimit,
		ExtraData:  extra,
		Alloc:      make(types.GenesisAlloc),
	}
	if genesis.Config.IsLondon(common.Big0) {
		genesis.BaseFee = big.NewInt(params.InitialBaseFee)
	}
	addFundedAccounts(cfg, genesis.Alloc)
	if cfg.CryptoUpgradeEnabled() {
		addCryptoUpgradeAlloc(genesis.Alloc)
	}
	return genesis, nil
}

// MarshalGenesisJSON 输出稳定缩进的 genesis JSON，便于实验归档。
func MarshalGenesisJSON(genesis *core.Genesis) ([]byte, error) {
	return json.MarshalIndent(genesis, "", "  ")
}

func cliqueChainConfig(cfg *Config) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:                 new(big.Int).SetUint64(cfg.Network.ChainID),
		HomesteadBlock:          new(big.Int).Set(zeroBlock),
		EIP150Block:             new(big.Int).Set(zeroBlock),
		EIP155Block:             new(big.Int).Set(zeroBlock),
		EIP158Block:             new(big.Int).Set(zeroBlock),
		ByzantiumBlock:          new(big.Int).Set(zeroBlock),
		ConstantinopleBlock:     new(big.Int).Set(zeroBlock),
		PetersburgBlock:         new(big.Int).Set(zeroBlock),
		IstanbulBlock:           new(big.Int).Set(zeroBlock),
		BerlinBlock:             new(big.Int).Set(zeroBlock),
		LondonBlock:             new(big.Int).Set(zeroBlock),
		TerminalTotalDifficulty: big.NewInt(math.MaxInt64),
		Clique: &params.CliqueConfig{
			Period: cfg.Consensus.CliquePeriod(),
			Epoch:  cfg.Consensus.Epoch,
		},
	}
}

func cliqueExtraData(signers []string) ([]byte, error) {
	extra := make([]byte, 32, 32+len(signers)*common.AddressLength+65)
	for _, signer := range signers {
		if !common.IsHexAddress(signer) {
			return nil, fmt.Errorf("invalid clique signer address %q", signer)
		}
		addr := common.HexToAddress(signer)
		extra = append(extra, addr[:]...)
	}
	extra = append(extra, make([]byte, 65)...)
	return extra, nil
}

func addFundedAccounts(cfg *Config, alloc types.GenesisAlloc) {
	for _, account := range cfg.Accounts {
		if common.IsHexAddress(account.Address) {
			balance := parseBalance(account.Balance)
			alloc[common.HexToAddress(account.Address)] = types.Account{Balance: balance}
		}
	}
	for _, node := range cfg.Nodes {
		if common.IsHexAddress(node.Account) {
			balance := defaultBalance()
			if account, ok := cfg.Accounts[node.Account]; ok {
				balance = parseBalance(account.Balance)
			}
			alloc[common.HexToAddress(node.Account)] = types.Account{Balance: balance}
		}
	}
}

func addCryptoUpgradeAlloc(alloc types.GenesisAlloc) {
	// CodeStorage 在该 fork 中通过 EVM special-case dispatcher 执行。
	// genesis 中仍创建一个非空账户，方便 RPC 检查确认实验链已启用 cryptoupgrade。
	alloc[common.CodeStorageAddress] = types.Account{
		Nonce:   1,
		Balance: common.Big0,
		Code:    []byte{0x00},
	}
}

func parseBalance(raw string) *big.Int {
	if raw == "" {
		return defaultBalance()
	}
	value, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return defaultBalance()
	}
	return value
}

func defaultBalance() *big.Int {
	value, _ := new(big.Int).SetString(defaultFundedBalance, 10)
	return value
}
