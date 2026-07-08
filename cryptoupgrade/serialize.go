package cryptoupgrade

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/log"
)

func ParseCall(data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", nil, fmt.Errorf("cryptoupgrade call input is shorter than method selector")
	}
	callFuncAbi := CodeStorageABI.Methods["callFunc"]
	if !bytes.Equal(callFuncAbi.ID, data[:4]) {
		return "", nil, fmt.Errorf("cryptoupgrade call selector %x does not match callFunc", data[:4])
	}
	params, err := callFuncAbi.Inputs.Unpack(data[4:])
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
	var args abi.Arguments
	// Construct arguments list
	for i := range paramsType {
		NType, err := abi.NewType(paramsType[i], "", nil)
		if err == nil {
			args = append(args, abi.Argument{Type: NType})
		} else {
			return nil, err
		}
	}
	// fmt.Printf("Args: %v\n", args)
	// fmt.Printf("hex:%x\n", encodedInput)
	params, err := args.Unpack(encodedInput)
	if err != nil {
		return nil, err
	}
	return params, nil
}

// Encode output with abi
func PackOutput(output []interface{}, returnType []string) ([]byte, error) {
	var args abi.Arguments
	var decodeOutput []byte
	// Construct arguments list
	for i := range returnType {
		NType, err := abi.NewType(returnType[i], "", nil)
		if err == nil {
			args = append(args, abi.Argument{Type: NType})
		} else {
			return nil, err
		}
	}
	decodeOutput, err := args.Pack(output...)
	if err != nil {
		return nil, err
	}
	return decodeOutput, err
}
