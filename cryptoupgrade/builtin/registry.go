// Package builtin contains algorithms compiled directly into the node binary.
package builtin

import (
	"github.com/ethereum/go-ethereum/cryptoupgrade/builtin/vote"
	tally "github.com/ethereum/go-ethereum/cryptoupgrade/builtin/vote/tally"
	thresholdencrypt "github.com/ethereum/go-ethereum/cryptoupgrade/builtin/vote/threshold_encrypt"
)

// Algorithm defines the runtime boundary for algorithms compiled into the node.
type Algorithm interface {
	RequiredGas() uint64
	TargetFunc() interface{}
	GetTypeList() ([]string, []string)
}

// registry is intentionally assembled only from builtin implementations.
// 动态候选源码位于 algorithm/go，不得成为节点运行时的静态依赖。
var registry = map[string]Algorithm{
	"VerifyDecryptionShareZKP": &thresholdencrypt.VerifyShare{},
	"AggregateShare":           &tally.AggrShare{},
	"VerifyBallotZKP":          &vote.VerifyBallot{},
	"AggregateBallot":          &tally.AggrBallot{},
}

// Lookup returns the builtin algorithm registered under name.
func Lookup(name string) (Algorithm, bool) {
	algorithm, ok := registry[name]
	return algorithm, ok
}
