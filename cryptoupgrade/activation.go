package cryptoupgrade

import (
	"context"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activation"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmcodec"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/wasmruntime"
)

func runtimeActivationService() *activation.Service {
	return activation.NewService(
		activation.CodecFunc(wasmcodec.DecodeToFile),
		wasmruntime.Default,
		runtimeAlgorithmRepository,
	)
}

// ActivateAlgorithm activates uploaded WASM without terminating the node on failure.
func ActivateAlgorithm(name string, info algoInfo) error {
	return runtimeActivationService().Activate(context.Background(), name, info)
}
