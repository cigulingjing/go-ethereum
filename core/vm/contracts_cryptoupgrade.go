package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
)

func isCryptoUpgradeCall(addr common.Address, input []byte) bool {
	return cryptoupgrade.IsCodeStorageCall(addr, input)
}

func runCryptoUpgradeCall(stateDB StateDB, input []byte, gas GasBudget, logger *tracing.Hooks, readOnly bool) (ret []byte, remaining GasBudget, err error) {
	gasCost, err := cryptoupgrade.RequiredGasForCodeStorageCall(input)
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
	output, err := cryptoupgrade.RunCodeStorageCall(input, readOnly, func(topics []common.Hash, data []byte) {
		stateDB.AddLog(&types.Log{
			Address: common.CodeStorageAddress,
			Topics:  topics,
			Data:    data,
		})
	})
	return output, gas, err
}
