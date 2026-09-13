package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

type fakeState struct {
	active           map[string]model.AlgorithmInfo
	uploaded         map[string]model.AlgorithmInfo
	uploadedVersions map[string]map[uint64]model.AlgorithmVersionInfo
}

func (s *fakeState) Active(name string) (model.AlgorithmInfo, bool) {
	info, ok := s.active[name]
	return info, ok
}

func (s *fakeState) Uploaded(name string) (model.AlgorithmInfo, bool) {
	info, ok := s.uploaded[name]
	if !ok {
		info, ok = s.active[name]
	}
	return info, ok
}

func (s *fakeState) SetUploaded(name string, info model.AlgorithmInfo) {
	s.uploaded[name] = info
}

func (s *fakeState) UploadedVersion(name string, version uint64) (model.AlgorithmVersionInfo, bool) {
	if versions := s.uploadedVersions[name]; versions != nil {
		if info, ok := versions[version]; ok {
			return info, true
		}
	}
	if version == 1 {
		if info, ok := s.uploaded[name]; ok {
			return model.LegacyVersion(info), true
		}
	}
	return model.AlgorithmVersionInfo{}, false
}

func (s *fakeState) SetUploadedVersion(name string, info model.AlgorithmVersionInfo) {
	if s.uploadedVersions == nil {
		s.uploadedVersions = make(map[string]map[uint64]model.AlgorithmVersionInfo)
	}
	if s.uploadedVersions[name] == nil {
		s.uploadedVersions[name] = make(map[uint64]model.AlgorithmVersionInfo)
	}
	s.uploadedVersions[name][info.Version] = info
}

func (s *fakeState) ActiveVersionAt(name string, blockNumber uint64) (model.AlgorithmVersionInfo, bool) {
	var selected model.AlgorithmVersionInfo
	var ok bool
	if info, exists := s.active[name]; exists && blockNumber == 0 {
		selected = model.LegacyVersion(info)
		ok = true
	}
	if versions := s.uploadedVersions[name]; versions != nil {
		for _, info := range versions {
			if info.ActivationBlock <= blockNumber && (!ok || info.Version > selected.Version) {
				selected = info
				ok = true
			}
		}
	}
	return selected, ok
}

func (s *fakeState) UpdateGas(string, uint64) (bool, bool) { return false, false }
func (*fakeState) Save() error                             { return nil }

type fakeCaller struct {
	requiredGasCalls int
	runCalls         int
}

func (c *fakeCaller) RequiredGas([]byte) (uint64, error) {
	c.requiredGasCalls++
	return 1, nil
}

func (c *fakeCaller) Run([]byte) ([]byte, error) {
	c.runCalls++
	return nil, nil
}

func (c *fakeCaller) RequiredGasAt(input []byte, _ uint64) (uint64, error) {
	return c.RequiredGas(input)
}

func (c *fakeCaller) RunAt(input []byte, _ uint64) ([]byte, error) {
	return c.Run(input)
}

func TestRequiredGasOnlyChargesCallFunc(t *testing.T) {
	contractABI := MustCodeStorageABI()
	dispatcher := NewDispatcher(contractABI, &fakeState{}, new(fakeCaller))

	uploadInput, err := contractABI.Pack("uploadCode", "add", "encoded", uint64(7), "bytes", "bytes")
	if err != nil {
		t.Fatalf("pack uploadCode: %v", err)
	}
	uploadGas, err := dispatcher.RequiredGas(uploadInput)
	if err != nil {
		t.Fatalf("RequiredGas uploadCode: %v", err)
	}
	if uploadGas != 0 {
		t.Fatalf("uploadCode gas = %d, want 0", uploadGas)
	}

	readInput, err := contractABI.Pack("getGas", "Add")
	if err != nil {
		t.Fatalf("pack getGas: %v", err)
	}
	readGas, err := dispatcher.RequiredGas(readInput)
	if err != nil {
		t.Fatalf("RequiredGas getGas: %v", err)
	}
	if readGas != 0 {
		t.Fatalf("getGas gas = %d, want 0", readGas)
	}

	callInput, err := contractABI.Pack("callFunc", "Add", []byte{})
	if err != nil {
		t.Fatalf("pack callFunc: %v", err)
	}
	callGas, err := dispatcher.RequiredGas(callInput)
	if err != nil {
		t.Fatalf("RequiredGas callFunc: %v", err)
	}
	if callGas != 1 {
		t.Fatalf("callFunc gas = %d, want 1 from fake caller", callGas)
	}
}

func TestUploadReceiptDoesNotMeanLocalActivation(t *testing.T) {
	contractABI := MustCodeStorageABI()
	state := &fakeState{
		active:           make(map[string]model.AlgorithmInfo),
		uploaded:         make(map[string]model.AlgorithmInfo),
		uploadedVersions: make(map[string]map[uint64]model.AlgorithmVersionInfo),
	}
	caller := new(fakeCaller)
	dispatcher := NewDispatcher(contractABI, state, caller)
	input, err := contractABI.Pack("uploadCode", "add", "encoded", uint64(7), "bytes", "bytes")
	if err != nil {
		t.Fatalf("pack uploadCode: %v", err)
	}
	var events int
	if _, err := dispatcher.Run(input, false, func(topics []common.Hash, _ []byte) {
		events++
		if len(topics) != 1 || topics[0] != CodeUploadedTopic {
			t.Fatalf("unexpected topics: %v", topics)
		}
	}); err != nil {
		t.Fatalf("uploadCode failed: %v", err)
	}
	if events != 1 {
		t.Fatalf("expected one event, got %d", events)
	}
	if _, ok := state.uploaded["Add"]; !ok {
		t.Fatal("upload metadata was not recorded")
	}
	if _, ok := state.active["Add"]; ok {
		t.Fatal("successful upload was treated as local activation")
	}
	if caller.requiredGasCalls != 0 || caller.runCalls != 0 {
		t.Fatalf("upload synchronously entered runtime call path: gas=%d run=%d", caller.requiredGasCalls, caller.runCalls)
	}
}

func TestUploadCodeVersionStoresPlanAndEmitsVersionEvent(t *testing.T) {
	contractABI := MustCodeStorageABI()
	state := &fakeState{
		active:           make(map[string]model.AlgorithmInfo),
		uploaded:         make(map[string]model.AlgorithmInfo),
		uploadedVersions: make(map[string]map[uint64]model.AlgorithmVersionInfo),
	}
	dispatcher := NewDispatcher(contractABI, state, new(fakeCaller)).WithBlockNumber(10)
	input, err := contractABI.Pack("uploadCodeVersion", "add", uint64(2), "encoded-v2", uint64(9), "bytes", "bytes", uint64(42))
	if err != nil {
		t.Fatalf("pack uploadCodeVersion: %v", err)
	}
	var gotName string
	var gotVersion uint64
	var gotActivation uint64
	if _, err := dispatcher.Run(input, false, func(topics []common.Hash, data []byte) {
		if len(topics) != 1 || topics[0] != CodeVersionUploadedTopic {
			t.Fatalf("unexpected topics: %v", topics)
		}
		values, err := contractABI.Unpack("codeVersionUploaded", data)
		if err != nil {
			t.Fatalf("unpack codeVersionUploaded: %v", err)
		}
		gotName = values[0].(string)
		gotVersion = values[1].(uint64)
		gotActivation = values[2].(uint64)
	}); err != nil {
		t.Fatalf("uploadCodeVersion failed: %v", err)
	}
	info, ok := state.UploadedVersion("Add", 2)
	if !ok {
		t.Fatal("version metadata was not recorded")
	}
	if info.Code != "encoded-v2" || info.Gas != 9 || info.Version != 2 || info.ActivationBlock != 42 {
		t.Fatalf("unexpected version metadata: %#v", info)
	}
	if gotName != "Add" || gotVersion != 2 || gotActivation != 42 {
		t.Fatalf("unexpected version event: %q v%d block %d", gotName, gotVersion, gotActivation)
	}
}

func TestActiveVersionUsesDispatcherBlockNumber(t *testing.T) {
	contractABI := MustCodeStorageABI()
	state := &fakeState{
		active:           make(map[string]model.AlgorithmInfo),
		uploaded:         make(map[string]model.AlgorithmInfo),
		uploadedVersions: make(map[string]map[uint64]model.AlgorithmVersionInfo),
	}
	state.SetUploadedVersion("Add", model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "v1", Gas: 1},
		Version:         1,
		ActivationBlock: 0,
	})
	state.SetUploadedVersion("Add", model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: "v2", Gas: 2},
		Version:         2,
		ActivationBlock: 42,
	})
	input, err := contractABI.Pack("getActiveVersion", "Add")
	if err != nil {
		t.Fatalf("pack getActiveVersion: %v", err)
	}
	dispatcher := NewDispatcher(contractABI, state, new(fakeCaller))
	before, err := dispatcher.WithBlockNumber(41).Run(input, true, nil)
	if err != nil {
		t.Fatalf("getActiveVersion before failed: %v", err)
	}
	beforeValues, err := contractABI.Unpack("getActiveVersion", before)
	if err != nil {
		t.Fatalf("unpack before: %v", err)
	}
	after, err := dispatcher.WithBlockNumber(42).Run(input, true, nil)
	if err != nil {
		t.Fatalf("getActiveVersion after failed: %v", err)
	}
	afterValues, err := contractABI.Unpack("getActiveVersion", after)
	if err != nil {
		t.Fatalf("unpack after: %v", err)
	}
	if beforeValues[0].(uint64) != 1 || afterValues[0].(uint64) != 2 {
		t.Fatalf("unexpected active versions before=%v after=%v", beforeValues, afterValues)
	}
}
