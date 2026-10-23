package main

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade"
)

func TestPricingProposalPayload(t *testing.T) {
	s, err := readSchedule([]byte(`{"threshold":2,"alpha":{"numerator":1,"denominator":2},"bands":[{"maxMetric":18446744073709551615,"gas":3000}]}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := s.Encode()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"version", "immediate"} {
		data, err := packPricedUpload(mode, "Verify", "wasm", 2, 42, 7, "bytes", "bool", raw)
		if err != nil {
			t.Fatal(err)
		}
		method, err := cryptoupgrade.CodeStorageABI.MethodById(data[:4])
		if err != nil {
			t.Fatal(err)
		}
		values, err := method.Inputs.Unpack(data[4:])
		if err != nil || !bytes.Equal(values[len(values)-1].([]byte), raw) {
			t.Fatalf("pricing not in proposal: %v", err)
		}
	}
	if _, err := packPricedUpload("upload", "Verify", "wasm", 2, 42, 7, "bytes", "bool", raw); err == nil {
		t.Fatal("unversioned pricing accepted")
	}
	if _, err := readSchedule([]byte(`{"threshold":2,"sharedProfile":true}`)); err == nil {
		t.Fatal("unknown pricing field accepted")
	}
}
