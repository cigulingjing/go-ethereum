package vm

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
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

func runCryptoUpgradeCall(stateDB StateDB, input []byte, blockNumber uint64, gas GasBudget, logger *tracing.Hooks, readOnly bool, traceContext ...context.Context) (ret []byte, remaining GasBudget, err error) {
	gasCost, err := cryptoupgrade.RequiredGasForCodeStorageCallAt(input, blockNumber)
	if err != nil {
		return nil, gas, err
	}
	prior, ok := gas.ChargeRegular(gasCost)
	if !ok {
		return nil, gas, ErrOutOfGas
	}
	if logger.HasGasHook() {
		logger.EmitGasChange(prior.AsTracing(), gas.AsTracing(), tracing.GasChangeCallPrecompiledContract)
	}
	ctx := context.Background()
	if len(traceContext) > 0 && traceContext[0] != nil {
		ctx = traceContext[0]
	}
	output, err := cryptoupgrade.RunCodeStorageCallAtContext(ctx, input, blockNumber, readOnly, func(topics []common.Hash, data []byte) {
		stateDB.AddLog(&types.Log{
			Address: common.CodeStorageAddress,
			Topics:  topics,
			Data:    data,
		})
	})
	return output, gas, err
}
