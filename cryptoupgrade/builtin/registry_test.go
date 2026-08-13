package builtin

import (
	"reflect"
	"testing"
)

func TestRegistryFixtures(t *testing.T) {
	fixtures := []struct {
		name    string
		inputs  []string
		outputs []string
	}{
		{
			name:    "VerifyDecryptionShareZKP",
			inputs:  []string{"uint64", "uint64", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes"},
			outputs: []string{"bool"},
		},
		{
			name:    "AggregateShare",
			inputs:  []string{"bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes"},
			outputs: []string{"bytes", "bytes", "bytes"},
		},
		{
			name:    "VerifyBallotZKP",
			inputs:  []string{"bytes", "bytes", "bytes", "bytes", "bytes", "bytes"},
			outputs: []string{"bool"},
		},
		{
			name:    "AggregateBallot",
			inputs:  []string{"bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes"},
			outputs: []string{"bytes", "bytes", "bytes"},
		},
	}

	if len(registry) != len(fixtures) {
		t.Fatalf("registry size changed: got %d, want %d", len(registry), len(fixtures))
	}
	for _, fixture := range fixtures {
		algorithm, ok := Lookup(fixture.name)
		if !ok {
			t.Fatalf("missing builtin algorithm %s", fixture.name)
		}
		if algorithm.RequiredGas() != 100 {
			t.Fatalf("%s gas changed: got %d, want 100", fixture.name, algorithm.RequiredGas())
		}
		inputs, outputs := algorithm.GetTypeList()
		if !reflect.DeepEqual(inputs, fixture.inputs) {
			t.Fatalf("%s input types changed: got %v, want %v", fixture.name, inputs, fixture.inputs)
		}
		if !reflect.DeepEqual(outputs, fixture.outputs) {
			t.Fatalf("%s output types changed: got %v, want %v", fixture.name, outputs, fixture.outputs)
		}
		if target := reflect.ValueOf(algorithm.TargetFunc()); target.Kind() != reflect.Func || target.IsNil() {
			t.Fatalf("%s target is not a callable function", fixture.name)
		}
	}
}

func TestLookupRejectsUnknownAlgorithm(t *testing.T) {
	if _, ok := Lookup("Add"); ok {
		t.Fatal("dynamic candidate Add must not be registered as a builtin")
	}
}
