package cryptoupgrade

import (
	"context"

	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activation"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/pluginruntime"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/sourcecodec"
)

func runtimeActivationService() *activation.Service {
	return activation.NewService(
		activation.CodecFunc(sourcecodec.DecodeToFile),
		activation.CompilerFunc(compilePlugin),
		runtimeAlgorithmRepository,
		activation.LoaderFunc(pluginruntime.Default.Activate),
	)
}

// ActivateAlgorithm activates uploaded source without terminating the node on failure.
func ActivateAlgorithm(name string, info algoInfo) error {
	return runtimeActivationService().Activate(context.Background(), name, info)
}
