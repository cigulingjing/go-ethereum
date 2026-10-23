package vm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/cryptoupgrade/pricing"
)

type cryptoUpgradePrecompile struct {
	entry cryptoupgrade.Precompile
}

func (c *cryptoUpgradePrecompile) RequiredGas(input []byte) uint64 {
	return c.entry.RequiredGas(input)
}

func (c *cryptoUpgradePrecompile) Run(input []byte) ([]byte, error) {
	return c.entry.Run(input)
}

func (c *cryptoUpgradePrecompile) Name() string {
	return c.entry.Name()
}

func cryptoUpgradePrecompiledContracts() PrecompiledContracts {
	entries := cryptoupgrade.Precompiles()
	contracts := make(PrecompiledContracts, len(entries))
	for _, entry := range entries {
		entry := entry
		contracts[entry.Address()] = &cryptoUpgradePrecompile{entry: entry}
	}
	return contracts
}

func cryptoUpgradePrecompileAddresses() []common.Address {
	return cryptoupgrade.PrecompileAddresses()
}

func isCryptoUpgradeCall(addr common.Address, input []byte) bool {
	return cryptoupgrade.IsCodeStorageCall(addr, input)
}

func runCryptoUpgradeCall(evm *EVM, input []byte, gas GasBudget, logger *tracing.Hooks, readOnly bool) (ret []byte, remaining GasBudget, err error) {
	blockNumber := evm.Context.BlockNumber.Uint64()
	gasCost, err := cryptoupgrade.RequiredGasForCodeStorageCallAt(input, blockNumber)
	if err != nil {
		return nil, gas, err
	}
	var pricedName string
	configuration := evm.chainConfig.CryptoUpgradePricing
	if configuration != nil && blockNumber >= configuration.ActivationBlock {
		name, encodedInput, parseErr := cryptoupgrade.ParseCall(input)
		if parseErr == nil {
			name = cryptoupgrade.NormalizeAlgorithmName(name)
			for _, algorithm := range configuration.Profile.Algorithms {
				if algorithm.Name != name {
					continue
				}
				version, ok := cryptoupgrade.ActiveVersionAt(name, blockNumber)
				if !ok {
					return nil, gas, errors.New("configured algorithm version is not active")
				}
				index := evm.cryptoUpgradePricingCalls[name] + 1
				gasCost, err = configuration.Profile.Quote(name, version, uint64(len(encodedInput)), index, 0)
				if err != nil {
					if errors.Is(err, pricing.ErrGasOverflow) {
						return nil, gas, ErrOutOfGas
					}
					return nil, gas, err
				}
				pricedName = name
				break
			}
		}
	}
	prior, ok := gas.ChargeRegular(gasCost)
	if !ok {
		return nil, gas, ErrOutOfGas
	}
	if pricedName != "" {
		if evm.cryptoUpgradePricingCalls == nil {
			evm.cryptoUpgradePricingCalls = make(map[string]uint64)
		}
		evm.cryptoUpgradePricingCalls[pricedName]++
	}
	if logger.HasGasHook() {
		logger.EmitGasChange(prior.AsTracing(), gas.AsTracing(), tracing.GasChangeCallPrecompiledContract)
	}
	ctx := context.Background()
	if evm.CryptoUpgradeContext != nil {
		ctx = evm.CryptoUpgradeContext
	}
	output, err := cryptoupgrade.RunCodeStorageCallAtContext(ctx, input, blockNumber, readOnly, func(topics []common.Hash, data []byte) {
		evm.StateDB.AddLog(&types.Log{
			Address: common.CodeStorageAddress,
			Topics:  topics,
			Data:    data,
		})
	})
	return output, gas, err
}
