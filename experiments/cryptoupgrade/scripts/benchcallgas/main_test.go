package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCallGasExperiment(t *testing.T) {
	path := filepath.Join("..", "..", "results", "gas-complexity", "2026-10-22-calibration.json")
	first, err := calculate(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := calculate(path)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("experiment is not reproducible: %v", err)
	}
	if len(first.Scenarios) != 5 || len(first.Scenarios[0].Samples) != 8 {
		t.Fatalf("unexpected scenarios: %+v", first)
	}
	if first.Scenarios[0].Samples[0].TotalFiveCallGas != 678 {
		t.Fatalf("unexpected Add gas: %+v", first.Scenarios[0].Samples[0])
	}
	firstSchnorr := first.Scenarios[0].Samples[6]
	if firstSchnorr.MaxCallsWithinLimit != 3 || firstSchnorr.CallGas[2] != 9_946_750 {
		t.Fatalf("unexpected unloaded Schnorr gas: %+v", firstSchnorr)
	}
	loadedSchnorr := first.Scenarios[4].Samples[6]
	if loadedSchnorr.MaxCallsWithinLimit != 2 || loadedSchnorr.CallGas[0] != 11_140_360 {
		t.Fatalf("unexpected loaded Schnorr gas: %+v", loadedSchnorr)
	}
}

func TestRejectIncompleteCalibration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "calibration.json")
	if err := os.WriteFile(path, []byte(`{"algorithms":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := calculate(path); err == nil {
		t.Fatal("incomplete calibration accepted")
	}
}

func TestBlakeSolidityReferenceLevel(t *testing.T) {
	path := filepath.Join("..", "..", "results", "gas-complexity", "2026-10-22-calibration.json")
	result, err := calculateForInput(path, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range result.Scenarios {
		if scenario.Samples[2].ReferenceValue != 32 || scenario.Samples[2].BaseGas != 160 {
			t.Fatalf("unexpected Blake reference: %+v", scenario.Samples[2])
		}
	}
}
