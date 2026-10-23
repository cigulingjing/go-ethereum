package main

import (
	"math/big"
	"reflect"
	"testing"
)

func TestPbkdf2FixtureIterationsAndArtifact(t *testing.T) {
	for _, iterations := range []int{1, 10, 100} {
		cfg := config{a: "1", b: "2", polyModulus: "12289", polyLeftLen: 4, polyRightLen: 4,
			pbkdfIterations: iterations, pbkdfWasmPath: "/tmp/test-pbkdf2.wasm"}
		fixtures, _, err := buildFixtures(cfg)
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for _, fixture := range fixtures {
			if fixture.Algorithm != "Pbkdf2Sha256" {
				continue
			}
			found = true
			if fixture.SourcePath != cfg.pbkdfWasmPath || fixture.ContractSource != "experiments/cryptoupgrade/algorithm/contracts/src/Pbkdf2Sha256.sol" {
				t.Fatalf("unexpected PBKDF2 source: %s, %s", fixture.SourcePath, fixture.ContractSource)
			}
			for _, values := range [][]interface{}{fixture.UpgradeValues, fixture.ContractValues, fixture.PrecompileValues} {
				if values[2].(*big.Int).Int64() != int64(iterations) || values[3].(*big.Int).Int64() != 32 || !reflect.DeepEqual(values[:2], fixture.UpgradeValues[:2]) {
					t.Fatalf("path input mismatch at iterations=%d", iterations)
				}
			}
		}
		if !found {
			t.Fatal("PBKDF2 fixture missing")
		}
	}
}

func TestPbkdf2IterationValidation(t *testing.T) {
	for _, iterations := range []int{0, 10001} {
		if err := (&config{rpc: "local", samples: 1, solcPath: "solc", pbkdfIterations: iterations}).validate(); err == nil {
			t.Fatalf("accepted invalid iteration count %d", iterations)
		}
	}
}
