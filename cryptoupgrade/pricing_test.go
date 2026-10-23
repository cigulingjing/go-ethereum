package cryptoupgrade

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/repository"
	"github.com/ethereum/go-ethereum/cryptoupgrade/pricing"
)

func TestVersionPricingUploadActivationAndReload(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	withRuntimePluginPaths(t, newPluginPaths(t.TempDir()), nil)
	code, err := EncodeWasmFile("../experiments/cryptoupgrade/algorithm/go/wasm/add.wasm")
	if err != nil {
		t.Fatal(err)
	}
	publish := func(name string, version, height uint64, s *pricing.Schedule) algoVersionInfo {
		t.Helper()
		method := "uploadCodeVersion"
		args := []interface{}{name, version, code, uint64(99), "int256,int256", "int256", height}
		var raw []byte
		if s != nil {
			raw, err = s.Encode()
			if err != nil {
				t.Fatal(err)
			}
			method += "WithPricing"
			args = append(args, raw)
		}
		input, err := CodeStorageABI.Pack(method, args...)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RunCodeStorageCallAt(input, 1, false, nil); err != nil {
			t.Fatal(err)
		}
		info, ok := getUploadedAlgorithmVersionInfo(name, version)
		if !ok || info.Pricing != hex.EncodeToString(raw) {
			t.Fatalf("proposal lost pricing: %+v", info)
		}
		query, _ := CodeStorageABI.Pack("getVersionPricing", name, version)
		out, err := RunCodeStorageCallAt(query, 1, true, nil)
		if err != nil {
			t.Fatal(err)
		}
		values, err := CodeStorageABI.Unpack("getVersionPricing", out)
		if err != nil || !bytes.Equal(values[0].([]byte), raw) {
			t.Fatalf("query changed pricing: %v", err)
		}
		if _, err := RunCodeStorageCallAt(input, 1, false, nil); err != nil {
			t.Fatalf("identical proposal replay failed: %v", err)
		}
		args[3] = uint64(100)
		conflict, _ := CodeStorageABI.Pack(method, args...)
		if _, err := RunCodeStorageCallAt(conflict, 1, false, nil); err == nil {
			t.Fatal("conflicting proposal accepted")
		}
		if s != nil {
			changed := *s
			changed.Threshold++
			changedRaw, err := changed.Encode()
			if err != nil {
				t.Fatal(err)
			}
			args[3], args[len(args)-1] = uint64(99), changedRaw
			conflict, _ = CodeStorageABI.Pack(method, args...)
			if _, err := RunCodeStorageCallAt(conflict, 1, false, nil); err == nil {
				t.Fatal("pricing changed within admitted version")
			}
			after, _ := getUploadedAlgorithmVersionInfo(name, version)
			if after.Pricing != info.Pricing {
				t.Fatal("rejected update changed pricing")
			}
		}
		if err := runtimeActivationService().ActivateVersion(context.Background(), name, info); err != nil {
			t.Fatal(err)
		}
		return info
	}
	a := pricing.Schedule{Threshold: 1, Alpha: pricing.Ratio{Numerator: 1, Denominator: 1}, Bands: []pricing.InputBand{{MaxMetric: 64, Gas: 10}, {MaxMetric: ^uint64(0), Gas: 20}}}
	b := pricing.Schedule{Threshold: 3, Alpha: pricing.Ratio{Numerator: 1, Denominator: 2}, Bands: []pricing.InputBand{{MaxMetric: ^uint64(0), Gas: 30}}}
	publish("PriceA", 1, 0, &a)
	publish("PriceB", 1, 0, &b)
	publish("PriceA", 2, 50, &b)
	publish("PriceFixed", 1, 0, nil)
	check := func() {
		for _, tc := range []struct {
			name                      string
			height, index, size, want uint64
		}{
			{"PriceA", 49, 1, 64, 10}, {"PriceA", 49, 2, 64, 20}, {"PriceA", 49, 2, 65, 40},
			{"PriceB", 49, 2, 64, 30}, {"PriceB", 49, 4, 64, 45}, {"PriceA", 50, 2, 64, 30},
			{"PriceFixed", 50, 20, 64, 99},
		} {
			input, _ := CodeStorageABI.Pack("callFunc", tc.name, make([]byte, tc.size))
			got, err := RequiredGasForCodeStorageCallAtIndex(input, tc.height, tc.index)
			if err != nil || got != tc.want {
				t.Fatalf("%+v: gas=%d err=%v", tc, got, err)
			}
		}
	}
	check()
	if err := runtimeAlgorithmRepository.Save(); err != nil {
		t.Fatal(err)
	}
	reloaded := repository.New(runtimePluginPaths.workspace())
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	runtimeAlgorithmRepository = reloaded
	check()
	encoded := make([]byte, 64)
	encoded[31], encoded[63] = 3, 4
	input, _ := CodeStorageABI.Pack("callFunc", "PriceA", encoded)
	result, err := RunCodeStorageCallAt(input, 50, true, nil)
	if err != nil || len(result) != 32 || result[31] != 7 {
		t.Fatalf("execution: %x %v", result, err)
	}
}

func TestPricingProposalRejectsInvalidAndStaticWrites(t *testing.T) {
	resetAlgorithmInfoForTest(t)
	s := pricing.Schedule{Threshold: 1, Alpha: pricing.Ratio{Numerator: 1, Denominator: 2}, Bands: []pricing.InputBand{{MaxMetric: ^uint64(0), Gas: 10}}}
	valid, err := s.Encode()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw      []byte
		readOnly bool
	}{{nil, false}, {[]byte{1, 2, 3}, false}, {valid, true}} {
		input, _ := CodeStorageABI.Pack("uploadCodeVersionWithPricing", "Invalid", uint64(1), "code", uint64(7), "bytes", "bytes", uint64(42), tc.raw)
		emitted := false
		_, err := RunCodeStorageCallAt(input, 1, tc.readOnly, func([]common.Hash, []byte) { emitted = true })
		if err == nil || emitted {
			t.Fatalf("invalid write accepted: %v", err)
		}
		if _, ok := getUploadedAlgorithmVersionInfo("Invalid", 1); ok {
			t.Fatal("invalid proposal changed metadata")
		}
	}
}
