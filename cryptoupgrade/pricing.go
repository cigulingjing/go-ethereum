package cryptoupgrade

import "github.com/ethereum/go-ethereum/cryptoupgrade/builtin"

// RequiredGasForCallAtIndex 只读取生效提案的参数，节点配置不能覆盖费用。
func RequiredGasForCallAtIndex(input []byte, blockNumber, index uint64) (uint64, error) {
	name, encoded, err := ParseCall(input)
	if err != nil {
		return 0, err
	}
	name = NormalizeAlgorithmName(name)
	gas, err := RequiredGasAt(name, blockNumber)
	if err != nil {
		return 0, err
	}
	if _, ok := builtin.Lookup(name); ok {
		return gas, nil
	}
	info, _ := getActiveAlgorithmVersionInfoAt(name, blockNumber)
	return quoteAlgorithmInput(info.Base(), encoded, index)
}

func quoteAlgorithmInput(info algoInfo, encoded []byte, index uint64) (uint64, error) {
	if info.Pricing == "" {
		return info.Gas, nil
	}
	schedule, err := info.PricingSchedule()
	if err != nil {
		return 0, err
	}
	return schedule.Quote(uint64(len(encoded)), index)
}

// RequiredGasForCodeStorageCallAtIndex 让升级、查询继续使用原有 Gas 边界。
func RequiredGasForCodeStorageCallAtIndex(input []byte, blockNumber, index uint64) (uint64, error) {
	if _, _, err := ParseCall(input); err == nil {
		return RequiredGasForCallAtIndex(input, blockNumber, index)
	}
	return runtimeCodeStorageDispatcher().WithBlockNumber(blockNumber).RequiredGas(input)
}
