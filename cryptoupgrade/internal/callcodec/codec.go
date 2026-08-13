// Package callcodec implements ABI encoding for dynamic algorithm calls.
package callcodec

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ParseCall decodes callFunc(name,input) from CodeStorage calldata.
func ParseCall(contractABI abi.ABI, data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", nil, fmt.Errorf("cryptoupgrade call input is shorter than method selector")
	}
	method, ok := contractABI.Methods["callFunc"]
	if !ok {
		return "", nil, fmt.Errorf("CodeStorage ABI does not define callFunc")
	}
	if !bytes.Equal(method.ID, data[:4]) {
		return "", nil, fmt.Errorf("cryptoupgrade call selector %x does not match callFunc", data[:4])
	}
	params, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return "", nil, err
	}
	if len(params) != 2 {
		return "", nil, fmt.Errorf("cryptoupgrade callFunc expected 2 arguments, got %d", len(params))
	}
	name, ok := params[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("cryptoupgrade callFunc first argument has type %T, want string", params[0])
	}
	input, ok := params[1].([]byte)
	if !ok {
		return "", nil, fmt.Errorf("cryptoupgrade callFunc second argument has type %T, want []byte", params[1])
	}
	return name, input, nil
}

// UnpackInput decodes algorithm arguments using Solidity ABI type names.
func UnpackInput(encoded []byte, typeNames []string) ([]interface{}, error) {
	args, err := arguments(typeNames)
	if err != nil {
		return nil, err
	}
	return args.Unpack(encoded)
}

// PackOutput encodes algorithm return values using Solidity ABI type names.
func PackOutput(output []interface{}, typeNames []string) ([]byte, error) {
	args, err := arguments(typeNames)
	if err != nil {
		return nil, err
	}
	return args.Pack(output...)
}

func arguments(typeNames []string) (abi.Arguments, error) {
	args := make(abi.Arguments, 0, len(typeNames))
	for _, typeName := range typeNames {
		abiType, err := abi.NewType(typeName, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Type: abiType})
	}
	return args, nil
}
