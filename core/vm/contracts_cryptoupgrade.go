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
	// 调用计数按算法隔离；费用参数随选定版本读取，不再查找链级 Profile。
	var pricedName string
	index := uint64(1)
	if name, _, parseErr := cryptoupgrade.ParseCall(input); parseErr == nil {
		pricedName = cryptoupgrade.NormalizeAlgorithmName(name)
		count := evm.cryptoUpgradePricingCalls[pricedName]
		if count == ^uint64(0) {
			return nil, gas, ErrOutOfGas
		}
		index = count + 1
	}
	gasCost, err := cryptoupgrade.RequiredGasForCodeStorageCallAtIndex(input, blockNumber, index)
	if err != nil {
		if errors.Is(err, pricing.ErrGasOverflow) {
			return nil, gas, ErrOutOfGas
		}
		return nil, gas, err
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
