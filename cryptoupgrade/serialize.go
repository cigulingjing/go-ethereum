package cryptoupgrade

import (
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/callcodec"
	"github.com/ethereum/go-ethereum/log"
)

func ParseCall(data []byte) (string, []byte, error) {
	return callcodec.ParseCall(CodeStorageABI, data)
}

// Decode callFunc(name,input), which is abi code
func UnpackCall(data []byte) []interface{} {
	name, input, err := ParseCall(data)
	if err != nil {
		log.Error("Failed to unpack callFunc inputs", "err", err)
		return nil
	}

	return []interface{}{name, input}
}

// Decode input in callFunc(name,input)
func UnpackInput(encodedInput []byte, paramsType []string) ([]interface{}, error) {
	return callcodec.UnpackInput(encodedInput, paramsType)
}

// Encode output with abi
func PackOutput(output []interface{}, returnType []string) ([]byte, error) {
	return callcodec.PackOutput(output, returnType)
}
