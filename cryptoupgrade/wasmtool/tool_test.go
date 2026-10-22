package wasmtool

import (
	"bytes"
	"testing"
)

func TestWrapperKeepsOutputAlive(t *testing.T) {
	for _, spec := range []Spec{
		{Function: "Add", InputTypes: []string{"int256", "int256"}, OutputTypes: []string{"int256"}},
		{Function: "Sha256", InputTypes: []string{"bytes"}, OutputTypes: []string{"bytes"}},
		{Function: "SchnorrVerify", InputTypes: []string{"bytes", "bytes"}, OutputTypes: []string{"bool"}},
		{Function: "PolynomialMul", InputTypes: []string{"uint256[]", "uint256[]", "uint256"}, OutputTypes: []string{"uint256[]"}},
	} {
		wrapper, err := wrapperSource(spec)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(wrapper, []byte("wasmOutputBuffer = output")) || !bytes.Contains(wrapper, []byte("return wasmKeepOutput(out)")) {
			t.Fatalf("output buffer is not retained for %s", spec.Function)
		}
	}
}
