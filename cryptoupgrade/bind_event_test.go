package cryptoupgrade

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

var errUnexpectedQuery = errors.New("unexpected query")

type fakeEventClient struct {
	callName string
	info     algoInfo
}

func (c *fakeEventClient) SubscribeFilterLogs(_ context.Context, q ethereum.FilterQuery, _ chan<- types.Log) (ethereum.Subscription, error) {
	if len(q.Addresses) != 1 || q.Addresses[0] != common.CodeStorageAddress {
		return nil, errUnexpectedQuery
	}
	if len(q.Topics) != 1 || len(q.Topics[0]) != 1 || q.Topics[0][0] != codeUploaded {
		return nil, errUnexpectedQuery
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	}), nil
}

func (c *fakeEventClient) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if msg.To == nil || *msg.To != common.CodeStorageAddress {
		return nil, errUnexpectedQuery
	}
	values, err := CodeStorageABI.Methods["getInfo"].Inputs.Unpack(msg.Data[4:])
	if err != nil {
		return nil, err
	}
	c.callName = values[0].(string)
	return CodeStorageABI.Methods["getInfo"].Outputs.Pack(c.info.Code, c.info.Gas, c.info.IType, c.info.OType)
}

func TestHandleCodeUploadedEventActivatesFromChainMetadata(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	withRuntimePluginPaths(t, newPluginPaths(t.TempDir()), nil)
	client := &fakeEventClient{
		info: algoInfo{
			Code:  "invalid compressed source",
			Gas:   11,
			IType: "bytes",
			OType: "bytes",
		},
	}
	eventData, err := CodeStorageABI.Events["codeUploaded"].Inputs.NonIndexed().Pack("add")
	if err != nil {
		t.Fatalf("pack codeUploaded: %v", err)
	}
	handleCodeUploadedEvent(client, types.Log{Address: common.CodeStorageAddress, Topics: []common.Hash{codeUploaded}, Data: eventData})
	if client.callName != "Add" {
		t.Fatalf("expected getInfo lookup for Add, got %q", client.callName)
	}
	if _, ok := getAlgorithmInfo("Add"); ok {
		t.Fatal("activation should not mark metadata active when ActivateAlgorithm fails")
	}
}

func TestHandleCodeUploadedEventSkipsAlreadyActive(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	info := algoInfo{
		Code:  "compressed-source",
		Gas:   11,
		IType: "bytes",
		OType: "bytes",
	}
	setAlgorithmInfo("Add", info)
	client := &fakeEventClient{info: info}
	eventData, err := CodeStorageABI.Events["codeUploaded"].Inputs.NonIndexed().Pack("Add")
	if err != nil {
		t.Fatalf("pack codeUploaded: %v", err)
	}
	handleCodeUploadedEvent(client, types.Log{Address: common.CodeStorageAddress, Topics: []common.Hash{codeUploaded}, Data: eventData})
	if client.callName != "Add" {
		t.Fatalf("expected getInfo lookup for Add, got %q", client.callName)
	}
}
