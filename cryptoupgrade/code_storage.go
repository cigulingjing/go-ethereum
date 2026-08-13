package cryptoupgrade

import (
	"github.com/ethereum/go-ethereum/common"
	evmadapter "github.com/ethereum/go-ethereum/cryptoupgrade/internal/evm"
)

type CodeStorageLogSink = evmadapter.LogSink

type runtimeCaller struct{}

func (runtimeCaller) RequiredGas(input []byte) (uint64, error) {
	return RequiredGasForCall(input)
}

func (runtimeCaller) Run(input []byte) ([]byte, error) {
	return RunCall(input)
}

func runtimeCodeStorageDispatcher() *evmadapter.Dispatcher {
	return evmadapter.NewDispatcher(CodeStorageABI, runtimeAlgorithmRepository, runtimeCaller{})
}

func IsCodeStorageCall(addr common.Address, input []byte) bool {
	return runtimeCodeStorageDispatcher().IsCall(addr, input)
}

func RequiredGasForCodeStorageCall(input []byte) (uint64, error) {
	return runtimeCodeStorageDispatcher().RequiredGas(input)
}

func RunCodeStorageCall(input []byte, readOnly bool, emitLog CodeStorageLogSink) ([]byte, error) {
	return runtimeCodeStorageDispatcher().Run(input, readOnly, emitLog)
}
