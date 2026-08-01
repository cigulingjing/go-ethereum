package cryptoupgrade

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// Private attributes
var (
	codeUploaded   = crypto.Keccak256Hash([]byte("codeUploaded(string)"))
	CodeStorageABI abi.ABI
)

func init() {
	// Load contract ABI
	var err error
	CodeStorageABI, err = abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		log.Error("Failed to load codestorage ABI", "err", err)
	}
	if runtimePluginPathsErr != nil {
		log.Error("Failed to resolve cryptoupgrade plugin directory", "env", pluginDirEnvVar, "err", runtimePluginPathsErr)
	}
	// Load stashed map
	if err = loadFromFile(algorithmInfoPath()); err != nil {
		log.Error("Failed to load upgrade algorithm map", "path", algorithmInfoPath(), "err", err)
	}
}
