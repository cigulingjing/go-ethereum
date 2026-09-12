package event

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/activationtrace"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

type fakeActivator struct {
	calls        int
	versionCalls int
	name         string
	version      uint64
	info         model.AlgorithmInfo
	versionInfo  model.AlgorithmVersionInfo
	err          error
}

func (a *fakeActivator) Activate(_ context.Context, name string, info model.AlgorithmInfo) error {
	a.calls++
	a.name = name
	a.info = info
	return a.err
}

func (a *fakeActivator) ActivateVersion(_ context.Context, name string, info model.AlgorithmVersionInfo) error {
	a.versionCalls++
	a.name = name
	a.version = info.Version
	a.versionInfo = info
	return a.err
}

type fakeClient struct {
	contractABI abi.ABI
	callName    string
	callVersion uint64
	info        model.AlgorithmInfo
	versionInfo model.AlgorithmVersionInfo
}

func (*fakeClient) SubscribeFilterLogs(context.Context, ethereum.FilterQuery, chan<- types.Log) (ethereum.Subscription, error) {
	return nil, errors.New("not used")
}

func (c *fakeClient) CallContract(_ context.Context, msg ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	for _, methodName := range []string{"getInfo", "getVersionInfo"} {
		method := c.contractABI.Methods[methodName]
		if len(msg.Data) >= 4 && bytes.Equal(method.ID, msg.Data[:4]) {
			values, err := method.Inputs.Unpack(msg.Data[4:])
			if err != nil {
				return nil, err
			}
			c.callName = values[0].(string)
			if methodName == "getInfo" {
				return method.Outputs.Pack(c.info.Code, c.info.Gas, c.info.IType, c.info.OType)
			}
			c.callVersion = values[1].(uint64)
			info := c.versionInfo
			return method.Outputs.Pack(info.Code, info.Gas, info.IType, info.OType, info.Version, info.ActivationBlock)
		}
	}
	return nil, errors.New("unexpected method")
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

func TestHandleWritesActivationTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "activation_trace.jsonl")
	t.Setenv(activationtrace.TraceFileEnvVar, tracePath)
	contractABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		t.Fatalf("parse ABI: %v", err)
	}
	info := model.AlgorithmInfo{Code: "encoded", Gas: 7, IType: "bytes", OType: "bytes"}
	client := &fakeClient{contractABI: contractABI, info: info}
	service := NewService(contractABI, common.CodeStorageAddress, common.Hash{1}, new(fakeActivator))
	data, err := contractABI.Events["codeUploaded"].Inputs.NonIndexed().Pack("add")
	if err != nil {
		t.Fatalf("pack event: %v", err)
	}
	log := types.Log{Data: data, TxHash: common.HexToHash("0x1234"), BlockNumber: 9}
	if err := service.Handle(context.Background(), client, log); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected trace line count %d: %s", len(lines), raw)
	}
	var first, second activationtrace.Event
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first trace: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("unmarshal second trace: %v", err)
	}
	if first.Stage != "event_received" || second.Stage != "activation_started" {
		t.Fatalf("unexpected stages: %#v %#v", first, second)
	}
	if first.Name != "Add" || first.Version != 1 || first.TxHash != log.TxHash.Hex() || first.BlockNumber != 9 {
		t.Fatalf("unexpected first trace: %#v", first)
	}
}

func TestHandleVersionedEventQueriesAndDelegatesActivation(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(common.CodeStorageABI_json))
	if err != nil {
		t.Fatalf("parse ABI: %v", err)
	}
	info := model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "encoded", Gas: 7, IType: "bytes", OType: "bytes"},
		Version:         2,
		ActivationBlock: 42,
	}
	client := &fakeClient{contractABI: contractABI, versionInfo: info}
	activator := new(fakeActivator)
	service := NewService(contractABI, common.CodeStorageAddress, common.Hash{1}, activator)
	event := contractABI.Events["codeVersionUploaded"]
	data, err := event.Inputs.NonIndexed().Pack("add", uint64(2), uint64(42))
	if err != nil {
		t.Fatalf("pack event: %v", err)
	}
	if err := service.Handle(context.Background(), client, types.Log{Topics: []common.Hash{event.ID}, Data: data}); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if client.callName != "Add" || client.callVersion != 2 {
		t.Fatalf("getVersionInfo queried %q v%d, want Add v2", client.callName, client.callVersion)
	}
	if activator.versionCalls != 1 || activator.name != "Add" || activator.version != 2 || activator.versionInfo != info {
		t.Fatalf("unexpected version activation delegation: calls=%d name=%q info=%#v", activator.versionCalls, activator.name, activator.versionInfo)
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
