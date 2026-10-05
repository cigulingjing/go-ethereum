package cryptoupgrade

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
)

func TestAigisSig2WASMUpgradeAndCall(t *testing.T) {
	if len(model.AigisSig2VerifyWASM) == 0 {
		t.Fatal("missing embedded Aigis WASM (run experiments/cryptoupgrade/algorithm/pqcgo/build.sh)")
	}

	dir := t.TempDir()
	withRuntimePluginPaths(t, newPluginPaths(filepath.Join(dir, "plugin")), nil)
	resetAlgorithmInfoForTest(t)

	encoded, err := EncodeWasm(model.AigisSig2VerifyWASM)
	if err != nil {
		t.Fatalf("EncodeWasm: %v", err)
	}
	info := model.AigisSig2VerifyInfo(encoded)

	const (
		version         = uint64(1)
		activationBlock = uint64(12)
		uploadBlock     = uint64(10)
		callBlock       = uint64(12)
	)
	name := model.AigisSig2VerifyName

	upload, err := CodeStorageABI.Pack("uploadCodeVersion", name, version, encoded, info.Gas, info.IType, info.OType, activationBlock)
	if err != nil {
		t.Fatalf("pack uploadCodeVersion: %v", err)
	}
	var gotName string
	var gotVersion uint64
	var gotActivation uint64
	if _, err := RunCodeStorageCallAt(upload, uploadBlock, false, func(_ []common.Hash, data []byte) {
		values, err := CodeStorageABI.Unpack("codeVersionUploaded", data)
		if err != nil {
			t.Fatalf("unpack codeVersionUploaded: %v", err)
		}
		gotName = values[0].(string)
		gotVersion = values[1].(uint64)
		gotActivation = values[2].(uint64)
	}); err != nil {
		t.Fatalf("uploadCodeVersion failed: %v", err)
	}
	if gotName != name || gotVersion != version || gotActivation != activationBlock {
		t.Fatalf("unexpected upgrade event name=%s version=%d activation=%d", gotName, gotVersion, gotActivation)
	}

	if err := ActivateAlgorithm(name, info); err != nil {
		t.Fatalf("ActivateAlgorithm (upgrade prepare) failed: %v", err)
	}

	input, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, model.AigisSig2Signature)
	if err != nil {
		t.Fatalf("pack verify input: %v", err)
	}
	call, err := CodeStorageABI.Pack("callFunc", name, input)
	if err != nil {
		t.Fatalf("pack callFunc: %v", err)
	}
	out, err := RunCodeStorageCallAt(call, callBlock, true, nil)
	if err != nil {
		t.Fatalf("callFunc valid signature failed: %v", err)
	}
	if !unpackAigisBool(t, out) {
		t.Fatalf("expected valid Aigis-sig2 signature to verify, output=%x", out)
	}

	tampered := append([]byte(nil), model.AigisSig2Signature...)
	tampered[0] ^= 0x01
	badInput, err := packAigisVerifyInput(model.AigisSig2PublicKey, model.AigisSig2Message, tampered)
	if err != nil {
		t.Fatalf("pack tampered input: %v", err)
	}
	badCall, err := CodeStorageABI.Pack("callFunc", name, badInput)
	if err != nil {
		t.Fatalf("pack tampered callFunc: %v", err)
	}
	badOut, err := RunCodeStorageCallAt(badCall, callBlock, true, nil)
	if err != nil {
		t.Fatalf("callFunc tampered signature failed: %v", err)
	}
	if unpackAigisBool(t, badOut) {
		t.Fatal("expected tampered Aigis-sig2 signature to fail verification")
	}
}

func packAigisVerifyInput(pk, message, signature []byte) ([]byte, error) {
	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{{Type: bytesType}, {Type: bytesType}, {Type: bytesType}}
	return args.Pack(pk, message, signature)
}

func unpackAigisBool(t *testing.T, encoded []byte) bool {
	t.Helper()
	boolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		t.Fatalf("bool abi type: %v", err)
	}
	values, err := abi.Arguments{{Type: boolType}}.Unpack(encoded)
	if err != nil {
		t.Fatalf("unpack bool output %x: %v", encoded, err)
	}
	ok, okType := values[0].(bool)
	if !okType {
		t.Fatalf("expected bool output, got %T", values[0])
	}
	return ok
}
