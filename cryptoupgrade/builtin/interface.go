package builtin

// 内置的算法放置在这里，不走预编译合约执行

type Algorithm interface {
	Name() string
	RequiredGas() uint64
	GetTypeList() ([]string, []string)
	Run(input any) ([]interface{}, error)
}

func Lookup(name string) (Algorithm, bool) {
	return nil, false
}
