package evm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

type fakeState struct {
	active   map[string]model.AlgorithmInfo
	uploaded map[string]model.AlgorithmInfo
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

func TestUploadReceiptDoesNotMeanLocalActivation(t *testing.T) {
	contractABI := MustCodeStorageABI()
	state := &fakeState{
		active:   make(map[string]model.AlgorithmInfo),
		uploaded: make(map[string]model.AlgorithmInfo),
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
