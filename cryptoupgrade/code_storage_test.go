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

func TestUploadCodeVersionStoresMetadataWithoutActivating(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	input, err := CodeStorageABI.Pack("uploadCodeVersion", "add", uint64(2), "compressed-source-v2", uint64(17), "bytes", "bytes", uint64(42))
	if err != nil {
		t.Fatalf("pack uploadCodeVersion: %v", err)
	}
	var gotName string
	var gotVersion uint64
	var gotActivation uint64
	_, err = RunCodeStorageCallAt(input, 10, false, func(_ []common.Hash, data []byte) {
		values, err := CodeStorageABI.Unpack("codeVersionUploaded", data)
		if err != nil {
			t.Fatalf("unpack codeVersionUploaded event: %v", err)
		}
		gotName = values[0].(string)
		gotVersion = values[1].(uint64)
		gotActivation = values[2].(uint64)
	})
	if err != nil {
		t.Fatalf("RunCodeStorageCall uploadCodeVersion failed: %v", err)
	}
	if gotName != "Add" || gotVersion != 2 || gotActivation != 42 {
		t.Fatalf("unexpected event: %q v%d block %d", gotName, gotVersion, gotActivation)
	}
	info, ok := getUploadedAlgorithmVersionInfo("Add", 2)
	if !ok {
		t.Fatal("uploadCodeVersion did not store uploaded version metadata")
	}
	if info.Code != "compressed-source-v2" || info.Gas != 17 || info.Version != 2 || info.ActivationBlock != 42 {
		t.Fatalf("unexpected uploaded version metadata: %#v", info)
	}
	if _, ok := getActiveAlgorithmVersionInfo("Add"); ok {
		t.Fatal("uploadCodeVersion should not mark the version locally active")
	}
}

func TestGetActiveVersionUsesBlockNumber(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	setUploadedAlgorithmVersionInfo("Add", algoVersionInfo{
		AlgorithmInfo:   algoInfo{Code: "v1", Gas: 7, IType: "bytes", OType: "bytes"},
		Version:         1,
		ActivationBlock: 0,
	})
	setUploadedAlgorithmVersionInfo("Add", algoVersionInfo{
		AlgorithmInfo:   algoInfo{Code: "v2", Gas: 9, IType: "bytes", OType: "bytes"},
		Version:         2,
		ActivationBlock: 42,
	})
	input, err := CodeStorageABI.Pack("getActiveVersion", "Add")
	if err != nil {
		t.Fatalf("pack getActiveVersion: %v", err)
	}
	before, err := RunCodeStorageCallAt(input, 41, true, nil)
	if err != nil {
		t.Fatalf("getActiveVersion before failed: %v", err)
	}
	beforeValues, err := CodeStorageABI.Unpack("getActiveVersion", before)
	if err != nil {
		t.Fatalf("unpack before: %v", err)
	}
	after, err := RunCodeStorageCallAt(input, 42, true, nil)
	if err != nil {
		t.Fatalf("getActiveVersion after failed: %v", err)
	}
	afterValues, err := CodeStorageABI.Unpack("getActiveVersion", after)
	if err != nil {
		t.Fatalf("unpack after: %v", err)
	}
	if beforeValues[0].(uint64) != 1 || afterValues[0].(uint64) != 2 {
		t.Fatalf("unexpected active versions before=%v after=%v", beforeValues, afterValues)
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
