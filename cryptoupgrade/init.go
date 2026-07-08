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
	algoInfoPath   = "./plugin/algorithm_info.json"
	compressedPath = "./plugin"
	CodeStorageABI abi.ABI
)

func init() {
	// Load contract ABI
	var err error
	CodeStorageABI, err = abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		log.Error("Failed to load codestorage ABI", "err", err)
	}
	// Load stashed map
	if err = loadFromFile(algoInfoPath); err != nil {
		log.Error("Failed to load upgrade algorithm map", "err", err)
	}
}
