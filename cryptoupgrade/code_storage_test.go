package cryptoupgrade

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/repository"
)

func TestUploadCodeStoresMetadataWithoutActivating(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	input, err := CodeStorageABI.Pack("uploadCode", "add", "compressed-source", uint64(7), "int256,int256", "int256")
	if err != nil {
		t.Fatalf("pack uploadCode: %v", err)
	}
	var gotEvents int
	var gotName string
	_, err = RunCodeStorageCall(input, false, func(_ []common.Hash, data []byte) {
		gotEvents++
		if err := CodeStorageABI.UnpackIntoInterface(&gotName, "codeUploaded", data); err != nil {
			t.Fatalf("unpack codeUploaded event: %v", err)
		}
	})
	if err != nil {
		t.Fatalf("RunCodeStorageCall uploadCode failed: %v", err)
	}
	if gotEvents != 1 {
		t.Fatalf("expected one codeUploaded event, got %d", gotEvents)
	}
	if gotName != "Add" {
		t.Fatalf("expected normalized event name Add, got %q", gotName)
	}
	if _, ok := getAlgorithmInfo("Add"); ok {
		t.Fatal("uploadCode should not mark the algorithm as locally active")
	}
	info, ok := getUploadedAlgorithmInfo("Add")
	if !ok {
		t.Fatal("uploadCode did not store uploaded metadata")
	}
	if info.Code != "compressed-source" || info.Gas != 7 || info.IType != "int256,int256" || info.OType != "int256" {
		t.Fatalf("unexpected uploaded metadata: %#v", info)
	}
}

func TestUploadCodeStaticContextDoesNotStoreOrEmit(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	input, err := CodeStorageABI.Pack("uploadCode", "Add", "compressed-source", uint64(7), "int256,int256", "int256")
	if err != nil {
		t.Fatalf("pack uploadCode: %v", err)
	}
	var emitted bool
	_, err = RunCodeStorageCall(input, true, func(_ []common.Hash, _ []byte) {
		emitted = true
	})
	if err == nil {
		t.Fatal("expected static uploadCode to fail")
	}
	if emitted {
		t.Fatal("static uploadCode emitted event")
	}
	if _, ok := getUploadedAlgorithmInfo("Add"); ok {
		t.Fatal("static uploadCode stored uploaded metadata")
	}
}

func TestCallFuncBeforeActivationUsesLocalActiveState(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	setUploadedAlgorithmInfo("Add", algoInfo{
		Code:  "compressed-source",
		Gas:   7,
		IType: "int256,int256",
		OType: "int256",
	})
	input, err := CodeStorageABI.Pack("callFunc", "Add", []byte{})
	if err != nil {
		t.Fatalf("pack callFunc: %v", err)
	}
	_, err = RunCodeStorageCall(input, false, nil)
	if err == nil {
		t.Fatal("expected callFunc to fail before local activation")
	}
	if !strings.Contains(err.Error(), "algorithm Add is not loaded") {
		t.Fatalf("unexpected callFunc error: %v", err)
	}
}

func TestUpdataGasUpdatesUploadedAndActiveMetadata(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	withRuntimePluginPaths(t, newPluginPaths(t.TempDir()), nil)
	info := algoInfo{
		Code:  "compressed-source",
		Gas:   7,
		IType: "bytes",
		OType: "bytes",
	}
	setUploadedAlgorithmInfo("Add", info)
	setAlgorithmInfo("Add", info)
	input, err := CodeStorageABI.Pack("updataGas", "Add", uint64(19))
	if err != nil {
		t.Fatalf("pack updataGas: %v", err)
	}
	if _, err := RunCodeStorageCall(input, false, nil); err != nil {
		t.Fatalf("RunCodeStorageCall updataGas failed: %v", err)
	}
	uploaded, ok := getUploadedAlgorithmInfo("Add")
	if !ok || uploaded.Gas != 19 {
		t.Fatalf("unexpected uploaded gas after update: ok=%t info=%#v", ok, uploaded)
	}
	active, ok := getAlgorithmInfo("Add")
	if !ok || active.Gas != 19 {
		t.Fatalf("unexpected active gas after update: ok=%t info=%#v", ok, active)
	}
}

func resetAlgorithmInfoForTest(t *testing.T) {
	t.Helper()

	oldRepository := runtimeAlgorithmRepository
	runtimeAlgorithmRepository = repository.New(runtimePluginPaths.workspace())
	t.Cleanup(func() {
		runtimeAlgorithmRepository = oldRepository
	})
}
