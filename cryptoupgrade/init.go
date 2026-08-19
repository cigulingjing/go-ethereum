package cryptoupgrade

import (
	evmadapter "github.com/ethereum/go-ethereum/cryptoupgrade/internal/evm"
	"github.com/ethereum/go-ethereum/log"
)

var (
	codeUploaded        = evmadapter.CodeUploadedTopic
	codeVersionUploaded = evmadapter.CodeVersionUploadedTopic
	CodeStorageABI      = evmadapter.MustCodeStorageABI()
)

func init() {
	if runtimePluginPathsErr != nil {
		log.Error("Failed to resolve cryptoupgrade plugin directory", "env", pluginDirEnvVar, "err", runtimePluginPathsErr)
	}
	// Load stashed map
	if err := loadFromFile(algorithmInfoPath()); err != nil {
		log.Error("Failed to load upgrade algorithm map", "path", algorithmInfoPath(), "err", err)
	}
}
