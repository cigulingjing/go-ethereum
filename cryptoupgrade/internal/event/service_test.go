package event

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

type fakeActivator struct {
	calls int
	name  string
	info  model.AlgorithmInfo
	err   error
}

func (a *fakeActivator) Activate(_ context.Context, name string, info model.AlgorithmInfo) error {
	a.calls++
	a.name = name
	a.info = info
	return a.err
}

type fakeClient struct {
	contractABI abi.ABI
	callName    string
	info        model.AlgorithmInfo
}

func (*fakeClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, errors.New("not used")
}

func (c *fakeClient) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	values, err := c.contractABI.Methods["getInfo"].Inputs.Unpack(msg.Data[4:])
	if err != nil {
		return nil, err
	}
	c.callName = values[0].(string)
	return c.contractABI.Methods["getInfo"].Outputs.Pack(c.info.Code, c.info.Gas, c.info.IType, c.info.OType)
}

func TestHandleOnlyQueriesAndDelegatesActivation(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		t.Fatalf("parse ABI: %v", err)
	}
	info := model.AlgorithmInfo{Code: "encoded", Gas: 7, IType: "bytes", OType: "bytes"}
	client := &fakeClient{contractABI: contractABI, info: info}
	activator := new(fakeActivator)
	service := NewService(contractABI, common.CodeStorageAddress, common.Hash{1}, activator)
	data, err := contractABI.Events["codeUploaded"].Inputs.NonIndexed().Pack("add")
	if err != nil {
		t.Fatalf("pack event: %v", err)
	}
	if err := service.Handle(context.Background(), client, types.Log{Data: data}); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if client.callName != "Add" {
		t.Fatalf("getInfo queried %q, want Add", client.callName)
	}
	if activator.calls != 1 || activator.name != "Add" || activator.info != info {
		t.Fatalf("unexpected activation delegation: calls=%d name=%q info=%#v", activator.calls, activator.name, activator.info)
	}
}

func TestHandleDoesNotHideActivationFailure(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		t.Fatalf("parse ABI: %v", err)
	}
	stageErr := errors.New("compile failed")
	activator := &fakeActivator{err: stageErr}
	client := &fakeClient{contractABI: contractABI}
	service := NewService(contractABI, common.CodeStorageAddress, common.Hash{1}, activator)
	data, err := contractABI.Events["codeUploaded"].Inputs.NonIndexed().Pack("Add")
	if err != nil {
		t.Fatalf("pack event: %v", err)
	}
	err = service.Handle(context.Background(), client, types.Log{Data: data})
	if !errors.Is(err, stageErr) || activator.calls != 1 {
		t.Fatalf("unexpected activation failure: calls=%d err=%v", activator.calls, err)
	}
}
