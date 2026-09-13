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

// CodeUploadedTopic is the stable legacy codeUploaded event signature.
var CodeUploadedTopic = crypto.Keccak256Hash([]byte("codeUploaded(string)"))

// CodeVersionUploadedTopic is the stable versioned upgrade event signature.
var CodeVersionUploadedTopic = crypto.Keccak256Hash([]byte("codeVersionUploaded(string,uint64,uint64)"))

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
	UploadedVersion(name string, version uint64) (model.AlgorithmVersionInfo, bool)
	SetUploadedVersion(name string, info model.AlgorithmVersionInfo)
	ActiveVersionAt(name string, blockNumber uint64) (model.AlgorithmVersionInfo, bool)
	UpdateGas(name string, gas uint64) (found, activeUpdated bool)
	Save() error
}

// Caller executes locally active algorithms outside the upload path.
type Caller interface {
	RequiredGas(input []byte) (uint64, error)
	Run(input []byte) ([]byte, error)
	RequiredGasAt(input []byte, blockNumber uint64) (uint64, error)
	RunAt(input []byte, blockNumber uint64) ([]byte, error)
}

// LogSink emits a CodeStorage event through the hosting EVM.
type LogSink func(topics []common.Hash, data []byte)

// Dispatcher adapts CodeStorage ABI calls to metadata state and runtime calls.
type Dispatcher struct {
	contractABI abi.ABI
	state       State
	caller      Caller
	blockNumber uint64
}

// NewDispatcher creates a CodeStorage dispatcher with injected state and runtime call boundaries.
func NewDispatcher(contractABI abi.ABI, state State, caller Caller) *Dispatcher {
	return &Dispatcher{contractABI: contractABI, state: state, caller: caller}
}

// WithBlockNumber returns a dispatcher view that resolves scheduled versions at blockNumber.
func (d *Dispatcher) WithBlockNumber(blockNumber uint64) *Dispatcher {
	copy := *d
	copy.blockNumber = blockNumber
	return &copy
}

// IsCall reports whether input selects a known CodeStorage method.
func (d *Dispatcher) IsCall(addr common.Address, input []byte) bool {
	return addr == common.CodeStorageAddress && len(input) >= 4 && d.method(input[:4]) != nil
}

// RequiredGas 仅对 callFunc 收取算法元数据中的执行 Gas。
// 升级与查询路径不在此重复定价，由标准交易固有 Gas（21000 + calldata）覆盖。
func (d *Dispatcher) RequiredGas(input []byte) (uint64, error) {
	if len(input) < 4 {
		return 0, errors.New("CodeStorage input is shorter than method selector")
	}
	method := d.method(input[:4])
	if method == nil {
		return 0, fmt.Errorf("unknown CodeStorage selector %x", input[:4])
	}
	if method.Name == "callFunc" {
		return d.caller.RequiredGasAt(input, d.blockNumber)
	}
	return 0, nil
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
		return d.caller.RunAt(input, d.blockNumber)
	case "getCode":
		info := d.activeOrUploadedInfo(model.NormalizeAlgorithmName(args[0].(string)))
		return method.Outputs.Pack(info.Code)
	case "getGas":
		info := d.activeOrUploadedInfo(model.NormalizeAlgorithmName(args[0].(string)))
		return method.Outputs.Pack(info.Gas)
	case "getInfo":
		info := d.activeOrUploadedInfo(model.NormalizeAlgorithmName(args[0].(string)))
		return method.Outputs.Pack(info.Code, info.Gas, info.IType, info.OType)
	case "getVersionInfo":
		name := model.NormalizeAlgorithmName(args[0].(string))
		info, _ := d.state.UploadedVersion(name, args[1].(uint64))
		return method.Outputs.Pack(info.Code, info.Gas, info.IType, info.OType, info.Version, info.ActivationBlock)
	case "getActiveVersion":
		name := model.NormalizeAlgorithmName(args[0].(string))
		info, _ := d.state.ActiveVersionAt(name, d.blockNumber)
		return method.Outputs.Pack(info.Version, info.ActivationBlock)
	case "getActiveInfo":
		name := model.NormalizeAlgorithmName(args[0].(string))
		info, _ := d.state.ActiveVersionAt(name, d.blockNumber)
		return method.Outputs.Pack(info.Code, info.Gas, info.IType, info.OType, info.Version, info.ActivationBlock)
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
	case "uploadCodeVersion":
		if readOnly {
			return nil, errors.New("CodeStorage uploadCodeVersion is not allowed in static context")
		}
		name := model.NormalizeAlgorithmName(args[0].(string))
		info := model.AlgorithmVersionInfo{
			AlgorithmInfo: model.AlgorithmInfo{
				Code:  args[2].(string),
				Gas:   args[3].(uint64),
				IType: args[4].(string),
				OType: args[5].(string),
			},
			Version:         args[1].(uint64),
			ActivationBlock: args[6].(uint64),
		}
		if err := validateVersionInfo(name, info); err != nil {
			return nil, err
		}
		d.state.SetUploadedVersion(name, info)
		return nil, d.emitCodeVersionUploaded(emitLog, name, info.Version, info.ActivationBlock)
	case "uploadCodeImmediate":
		if readOnly {
			return nil, errors.New("CodeStorage uploadCodeImmediate is not allowed in static context")
		}
		name := model.NormalizeAlgorithmName(args[0].(string))
		info := model.AlgorithmVersionInfo{
			AlgorithmInfo: model.AlgorithmInfo{
				Code:  args[2].(string),
				Gas:   args[3].(uint64),
				IType: args[4].(string),
				OType: args[5].(string),
			},
			Version:         args[1].(uint64),
			ActivationBlock: d.blockNumber,
		}
		if err := validateVersionInfo(name, info); err != nil {
			return nil, err
		}
		d.state.SetUploadedVersion(name, info)
		return nil, d.emitCodeVersionUploaded(emitLog, name, info.Version, info.ActivationBlock)
	default:
		return nil, fmt.Errorf("unsupported CodeStorage method %s", method.Name)
	}
}

func (d *Dispatcher) activeOrUploadedInfo(name string) model.AlgorithmInfo {
	if version, ok := d.state.ActiveVersionAt(name, d.blockNumber); ok {
		return version.Base()
	}
	info, _ := d.state.Uploaded(name)
	return info
}

func (d *Dispatcher) emitCodeVersionUploaded(emitLog LogSink, name string, version, activationBlock uint64) error {
	if emitLog == nil {
		return nil
	}
	eventData, err := d.contractABI.Events["codeVersionUploaded"].Inputs.NonIndexed().Pack(name, version, activationBlock)
	if err != nil {
		return err
	}
	emitLog([]common.Hash{CodeVersionUploadedTopic}, eventData)
	return nil
}

func validateVersionInfo(name string, info model.AlgorithmVersionInfo) error {
	if name == "" {
		return errors.New("algorithm name is empty")
	}
	if info.Version == 0 {
		return errors.New("algorithm version is zero")
	}
	if info.Code == "" {
		return errors.New("algorithm code is empty")
	}
	if info.Gas == 0 {
		return errors.New("algorithm gas is zero")
	}
	return nil
}

func (d *Dispatcher) method(selector []byte) *abi.Method {
	for _, method := range d.contractABI.Methods {
		if bytes.Equal(method.ID, selector) {
			return &method
		}
	}
	return nil
}
