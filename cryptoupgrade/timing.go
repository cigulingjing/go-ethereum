package cryptoupgrade

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
)

// TimingFields 只提取可关联元数据，不向性能日志复制 WASM 或调用参数。
func TimingFields(to *common.Address, input []byte) stagelog.Fields {
	fields := stagelog.Fields{"flow": "transaction"}
	if to == nil || *to != common.CodeStorageAddress || len(input) < 4 {
		return fields
	}
	method, err := CodeStorageABI.MethodById(input[:4])
	if err != nil {
		return fields
	}
	switch method.Name {
	case "callFunc":
		fields["flow"] = "call"
	case "uploadCode", "uploadCodeVersion", "uploadCodeImmediate":
		fields["flow"] = "upgrade"
	default:
		return fields
	}
	fields["contractMethod"] = method.Name
	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return fields
	}
	if len(args) > 0 {
		if name, ok := args[0].(string); ok {
			fields["algorithm"] = capitalString(name)
		}
	}
	if method.Name == "uploadCode" {
		fields["version"] = uint64(1)
	}
	if method.Name == "uploadCodeVersion" || method.Name == "uploadCodeImmediate" {
		fields["version"] = args[1]
	}
	if method.Name == "uploadCodeVersion" {
		fields["activationBlock"] = args[6]
	}
	return fields
}
