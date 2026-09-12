package cryptoupgrade

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	evmadapter "github.com/ethereum/go-ethereum/cryptoupgrade/internal/evm"
)

type CodeStorageLogSink = evmadapter.LogSink

type runtimeCaller struct{ ctx context.Context }

func (runtimeCaller) RequiredGas(input []byte) (uint64, error) {
	return RequiredGasForCall(input)
}

func (runtimeCaller) Run(input []byte) ([]byte, error) {
	return RunCall(input)
}

func (runtimeCaller) RequiredGasAt(input []byte, blockNumber uint64) (uint64, error) {
	return RequiredGasForCallAt(input, blockNumber)
}

func (c runtimeCaller) RunAt(input []byte, blockNumber uint64) ([]byte, error) {
	return RunCallAtContext(c.ctx, input, blockNumber)
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

func RequiredGasForCodeStorageCallAt(input []byte, blockNumber uint64) (uint64, error) {
	return runtimeCodeStorageDispatcher().WithBlockNumber(blockNumber).RequiredGas(input)
}

func RunCodeStorageCall(input []byte, readOnly bool, emitLog CodeStorageLogSink) ([]byte, error) {
	return runtimeCodeStorageDispatcher().Run(input, readOnly, emitLog)
}

func RunCodeStorageCallAt(input []byte, blockNumber uint64, readOnly bool, emitLog CodeStorageLogSink) ([]byte, error) {
	return runtimeCodeStorageDispatcher().WithBlockNumber(blockNumber).Run(input, readOnly, emitLog)
}

// RunCodeStorageCallAtContext 将观测标识沿调用链传递，不改变版本选择或 Gas。
func RunCodeStorageCallAtContext(ctx context.Context, input []byte, blockNumber uint64, readOnly bool, emitLog CodeStorageLogSink) ([]byte, error) {
	return evmadapter.NewDispatcher(CodeStorageABI, runtimeAlgorithmRepository, runtimeCaller{ctx: ctx}).WithBlockNumber(blockNumber).Run(input, readOnly, emitLog)
}
