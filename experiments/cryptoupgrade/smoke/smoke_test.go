package smoke

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
)

type fakeBackend struct {
	receiptCalls int
	callCalls    int
}

func (f *fakeBackend) SendTransaction(context.Context, map[string]interface{}) (common.Hash, error) {
	return common.HexToHash("0x1"), nil
}

func (f *fakeBackend) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	f.receiptCalls++
	if f.receiptCalls == 1 {
		return nil, ethereum.NotFound
	}
	return &types.Receipt{Status: types.ReceiptStatusSuccessful}, nil
}

func (f *fakeBackend) CallContract(context.Context, ethereum.CallMsg) ([]byte, error) {
	f.callCalls++
	if f.callCalls < 3 {
		return nil, errors.New("algorithm is not activated")
	}
	uint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	return (abi.Arguments{{Type: uint256}}).Pack(big.NewInt(200))
}

func (*fakeBackend) Close() {}

func TestRunAddWaitsForActivationAfterReceipt(t *testing.T) {
	client := new(fakeBackend)
	node := network.NodeConfig{
		ID:      "node1",
		Role:    "signer",
		Account: "0xF5F871aA6Bd253914705898c66251f994aa426FA",
	}
	result := runAdd(context.Background(), node, "encoded-source", Options{
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	}, client)
	if !result.OK {
		t.Fatalf("runAdd failed: %s", result.Error)
	}
	if client.receiptCalls != 2 {
		t.Fatalf("receipt calls = %d, want 2", client.receiptCalls)
	}
	if client.callCalls != 3 {
		t.Fatalf("callFunc calls = %d, want 3", client.callCalls)
	}
	if result.Output == "" {
		t.Fatal("missing callFunc output")
	}
}
