package cryptoupgrade

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	codeStorageReadGas  uint64 = 3000
	codeStorageWriteGas uint64 = 50000
)

type CodeStorageLogSink func(topics []common.Hash, data []byte)

func IsCodeStorageCall(addr common.Address, input []byte) bool {
	if addr != common.CodeStorageAddress || len(input) < 4 {
		return false
	}
	return codeStorageMethodBySelector(input[:4]) != nil
}

func RequiredGasForCodeStorageCall(input []byte) (uint64, error) {
	method := codeStorageMethodBySelector(input[:4])
	if method == nil {
		return 0, fmt.Errorf("unknown CodeStorage selector %x", input[:4])
	}
	switch method.Name {
	case "callFunc":
		return RequiredGasForCall(input)
	case "uploadCode", "updataGas":
		return codeStorageWriteGas + uint64(len(input))*16, nil
	default:
		return codeStorageReadGas + uint64(len(input))*4, nil
	}
}

func RunCodeStorageCall(input []byte, readOnly bool, emitLog CodeStorageLogSink) ([]byte, error) {
	if len(input) < 4 {
		return nil, fmt.Errorf("CodeStorage input is shorter than method selector")
	}
	method := codeStorageMethodBySelector(input[:4])
	if method == nil {
		return nil, fmt.Errorf("unknown CodeStorage selector %x", input[:4])
	}
	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, err
	}

	switch method.Name {
	case "callFunc":
		return RunCall(input)
	case "getCode":
		info, _ := getAlgorithmInfo(capitalString(args[0].(string)))
		return method.Outputs.Pack(info.code)
	case "getGas":
		info, _ := getAlgorithmInfo(capitalString(args[0].(string)))
		return method.Outputs.Pack(info.gas)
	case "getInfo":
		info, _ := getAlgorithmInfo(capitalString(args[0].(string)))
		return method.Outputs.Pack(info.code, info.gas, info.itype, info.otype)
	case "updataGas":
		if readOnly {
			return nil, errors.New("CodeStorage updataGas is not allowed in static context")
		}
		name := capitalString(args[0].(string))
		gas := args[1].(uint64)
		info, ok := getAlgorithmInfo(name)
		if !ok {
			return nil, fmt.Errorf("algorithm %s is not loaded", name)
		}
		info.gas = gas
		setAlgorithmInfo(name, info)
		return nil, Store()
	case "uploadCode":
		if readOnly {
			return nil, errors.New("CodeStorage uploadCode is not allowed in static context")
		}
		name := capitalString(args[0].(string))
		info := algoInfo{
			code:  args[1].(string),
			gas:   args[2].(uint64),
			itype: args[3].(string),
			otype: args[4].(string),
		}
		if err := ActivateAlgorithm(name, info); err != nil {
			return nil, err
		}
		if emitLog != nil {
			eventData, err := CodeStorageABI.Events["codeUploaded"].Inputs.NonIndexed().Pack(name)
			if err != nil {
				return nil, err
			}
			emitLog([]common.Hash{codeUploaded}, eventData)
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported CodeStorage method %s", method.Name)
	}
}

func ActivateAlgorithm(name string, info algoInfo) error {
	if name == "" {
		return errors.New("algorithm name is empty")
	}
	name = capitalString(name)
	if err := directoryInit(); err != nil {
		return err
	}

	sourcefilePath := gofilePath(name)
	if err := decompressStringToFile(info.code, sourcefilePath); err != nil {
		return err
	}
	pluginfilePath := sofilePath(name)
	if err := PluginCompile(sourcefilePath, pluginfilePath); err != nil {
		return err
	}
	setAlgorithmInfo(name, info)
	return Store()
}

func codeStorageMethodBySelector(selector []byte) *abi.Method {
	for _, method := range CodeStorageABI.Methods {
		if bytes.Equal(method.ID, selector) {
			return &method
		}
	}
	return nil
}
