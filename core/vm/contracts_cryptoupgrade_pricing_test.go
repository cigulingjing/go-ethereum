package vm

import (
	"context"
	"math/big"
	"os"
	"os/exec"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/cryptoupgrade/pricing"
	"github.com/ethereum/go-ethereum/event"
)

// 使用独立进程，使模块仓库在 package init 时指向临时目录。
func TestCryptoUpgradeProposalPricingExecution(t *testing.T) {
	if os.Getenv("GETH_TEST_PRICING_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCryptoUpgradeProposalPricingExecution$", "-test.v")
		cmd.Env = append(os.Environ(), "GETH_TEST_PRICING_CHILD=1", "GETH_CRYPTOUPGRADE_PLUGIN_DIR="+t.TempDir())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated EVM test: %v\n%s", err, out)
		}
		return
	}
	code, err := cryptoupgrade.EncodeWasmFile("../../experiments/cryptoupgrade/algorithm/go/wasm/add.wasm")
	if err != nil {
		t.Fatal(err)
	}
	var logs []types.Log
	for _, proposal := range []struct {
		name                                                         string
		version, activation, threshold, numerator, denominator, base uint64
	}{
		{"MeterA", 1, 0, 1, 1, 1, 10},
		{"MeterB", 1, 0, 3, 1, 2, 30},
		{"MeterA", 2, 50, 2, 1, 2, 100},
	} {
		s := pricing.Schedule{Threshold: proposal.threshold, Alpha: pricing.Ratio{Numerator: proposal.numerator, Denominator: proposal.denominator}, Bands: []pricing.InputBand{{MaxMetric: ^uint64(0), Gas: proposal.base}}}
		raw, err := s.Encode()
		if err != nil {
			t.Fatal(err)
		}
		data, err := cryptoupgrade.CodeStorageABI.Pack("uploadCodeVersionWithPricing", proposal.name, proposal.version, code, uint64(999), "int256,int256", "int256", proposal.activation, raw)
		if err != nil {
			t.Fatal(err)
		}
		_, err = cryptoupgrade.RunCodeStorageCallAt(data, 1, false, func(topics []common.Hash, data []byte) {
			logs = append(logs, types.Log{Topics: topics, Data: data, BlockNumber: 1})
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	cryptoupgrade.BindCodeUploaded(&pricingEventClient{logs: logs})
	evm := &EVM{Context: BlockContext{BlockNumber: big.NewInt(49)}}
	input := make([]byte, 64)
	input[31], input[63] = 3, 4
	call := func(name string, budget, wantCharge uint64, wantErr error) {
		t.Helper()
		data, err := cryptoupgrade.CodeStorageABI.Pack("callFunc", name, input)
		if err != nil {
			t.Fatal(err)
		}
		out, remaining, err := runCryptoUpgradeCall(evm, data, NewGasBudget(budget, 0), nil, false)
		if err != wantErr {
			t.Fatalf("%s: error=%v want=%v", name, err, wantErr)
		}
		if remaining.RegularGas != budget-wantCharge {
			t.Fatalf("%s: charged=%d want=%d", name, budget-remaining.RegularGas, wantCharge)
		}
		if err == nil && (len(out) != 32 || out[31] != 7) {
			t.Fatalf("%s: incorrect output %x", name, out)
		}
	}
	call("MeterA", 100, 10, nil)
	call("MeterB", 100, 30, nil)
	call("meterA", 100, 20, nil)
	call("MeterB", 100, 30, nil)
	call("MeterA", 39, 0, ErrOutOfGas)
	call("MeterA", 40, 40, nil)
	if evm.cryptoUpgradePricingCalls["MeterA"] != 3 {
		t.Fatal("out-of-gas call advanced counter")
	}
	evm.SetTxContext(TxContext{})
	call("MeterA", 10, 10, nil)
	call("MeterB", 30, 30, nil)
	evm.Context.BlockNumber.SetUint64(50)
	evm.SetTxContext(TxContext{})
	call("MeterA", 100, 100, nil)
	call("MeterB", 30, 30, nil)
	evm.cryptoUpgradePricingCalls["MeterA"] = 1000
	call("MeterA", 1000, 0, ErrOutOfGas)
}

type pricingEventClient struct{ logs []types.Log }

func (c *pricingEventClient) SubscribeFilterLogs(_ context.Context, _ ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		for _, log := range c.logs {
			select {
			case ch <- log:
			case <-quit:
				return nil
			}
		}
		close(ch)
		<-quit
		return nil
	}), nil
}

func (*pricingEventClient) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	return cryptoupgrade.RunCodeStorageCallAt(msg.Data, 1, true, nil)
}
