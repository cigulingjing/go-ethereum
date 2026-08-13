// Package evm contains the CodeStorage and precompile integration boundary.
package evm

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

const (
	codeStorageReadGas  uint64 = 3000
	codeStorageWriteGas uint64 = 50000
)

// CodeUploadedTopic is the stable codeUploaded event signature.
var CodeUploadedTopic = crypto.Keccak256Hash([]byte("codeUploaded(string)"))

// MustCodeStorageABI parses the built-in CodeStorage ABI or panics during initialization.
func MustCodeStorageABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		panic(fmt.Sprintf("parse CodeStorage ABI: %v", err))
	}
	return parsed
}

// State is the metadata boundary used by EVM transaction dispatch.
type State interface {
	Active(name string) (model.AlgorithmInfo, bool)
	Uploaded(name string) (model.AlgorithmInfo, bool)
	SetUploaded(name string, info model.AlgorithmInfo)
	UpdateGas(name string, gas uint64) (found, activeUpdated bool)
	Save() error
}

// Caller executes locally active algorithms outside the upload path.
type Caller interface {
	RequiredGas(input []byte) (uint64, error)
	Run(input []byte) ([]byte, error)
}

// LogSink emits a CodeStorage event through the hosting EVM.
type LogSink func(topics []common.Hash, data []byte)

// Dispatcher adapts CodeStorage ABI calls to metadata state and runtime calls.
type Dispatcher struct {
	contractABI abi.ABI
	state       State
	caller      Caller
}

// NewDispatcher creates a CodeStorage dispatcher with injected state and runtime call boundaries.
func NewDispatcher(contractABI abi.ABI, state State, caller Caller) *Dispatcher {
	return &Dispatcher{contractABI: contractABI, state: state, caller: caller}
}

// IsCall reports whether input selects a known CodeStorage method.
func (d *Dispatcher) IsCall(addr common.Address, input []byte) bool {
	return addr == common.CodeStorageAddress && len(input) >= 4 && d.method(input[:4]) != nil
}

// RequiredGas returns the unchanged CodeStorage gas schedule.
func (d *Dispatcher) RequiredGas(input []byte) (uint64, error) {
	if len(input) < 4 {
		return 0, errors.New("CodeStorage input is shorter than method selector")
	}
	method := d.method(input[:4])
	if method == nil {
		return 0, fmt.Errorf("unknown CodeStorage selector %x", input[:4])
	}
	switch method.Name {
	case "callFunc":
		return d.caller.RequiredGas(input)
	case "uploadCode", "updataGas":
		return codeStorageWriteGas + uint64(len(input))*16, nil
	default:
		return codeStorageReadGas + uint64(len(input))*4, nil
	}
}

// Run dispatches one CodeStorage call without compiling uploads in the EVM path.
func (d *Dispatcher) Run(input []byte, readOnly bool, emitLog LogSink) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("CodeStorage input is shorter than method selector")
	}
	method := d.method(input[:4])
	if method == nil {
		return nil, fmt.Errorf("unknown CodeStorage selector %x", input[:4])
	}
	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, err
	}
	switch method.Name {
	case "callFunc":
		return d.caller.Run(input)
	case "getCode":
		info, _ := d.state.Uploaded(model.NormalizeAlgorithmName(args[0].(string)))
		return method.Outputs.Pack(info.Code)
	case "getGas":
		info, _ := d.state.Uploaded(model.NormalizeAlgorithmName(args[0].(string)))
		return method.Outputs.Pack(info.Gas)
	case "getInfo":
		info, _ := d.state.Uploaded(model.NormalizeAlgorithmName(args[0].(string)))
		return method.Outputs.Pack(info.Code, info.Gas, info.IType, info.OType)
	case "updataGas":
		if readOnly {
			return nil, errors.New("CodeStorage updataGas is not allowed in static context")
		}
		name := model.NormalizeAlgorithmName(args[0].(string))
		found, activeUpdated := d.state.UpdateGas(name, args[1].(uint64))
		if !found {
			return nil, fmt.Errorf("algorithm %s is not loaded", name)
		}
		if activeUpdated {
			return nil, d.state.Save()
		}
		return nil, nil
	case "uploadCode":
		if readOnly {
			return nil, errors.New("CodeStorage uploadCode is not allowed in static context")
		}
		name := model.NormalizeAlgorithmName(args[0].(string))
		info := model.AlgorithmInfo{
			Code:  args[1].(string),
			Gas:   args[2].(uint64),
			IType: args[3].(string),
			OType: args[4].(string),
		}
		// uploadCode 只登记链上数据并发出事件；编译和加载必须由 event/activation 异步完成。
		d.state.SetUploaded(name, info)
		if emitLog != nil {
			eventData, err := d.contractABI.Events["codeUploaded"].Inputs.NonIndexed().Pack(name)
			if err != nil {
				return nil, err
			}
			emitLog([]common.Hash{CodeUploadedTopic}, eventData)
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported CodeStorage method %s", method.Name)
	}
}

func (d *Dispatcher) method(selector []byte) *abi.Method {
	for _, method := range d.contractABI.Methods {
		if bytes.Equal(method.ID, selector) {
			return &method
		}
	}
	return nil
}
