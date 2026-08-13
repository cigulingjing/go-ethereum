package cryptoupgrade

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/cryptoupgrade/builtin/bls12381"
)

func TestBuiltinCallProcessorFixture(t *testing.T) {
	group := bls12381.NewG1()
	point := group.ToBytes(group.One())
	input := mustPackABI(t,
		[]string{"bytes", "bytes", "bytes", "bytes", "bytes", "bytes", "bytes"},
		point, point, point, point, point, point, point,
	)

	output, remainingGas, err := CallProcessor("AggregateShare", 150, input)
	if err != nil {
		t.Fatalf("AggregateShare builtin call failed: %v", err)
	}
	if remainingGas != 50 {
		t.Fatalf("remaining gas changed: got %d, want 50", remainingGas)
	}
	values := mustUnpackABI(t, []string{"bytes", "bytes", "bytes"}, output)
	doubled := group.ToBytes(group.Add(group.New(), group.One(), group.One()))
	if !bytes.Equal(values[0].([]byte), doubled) || !bytes.Equal(values[1].([]byte), doubled) {
		t.Fatal("AggregateShare sum output changed")
	}
	// 保留第三个输出的既有行为，目录迁移不得顺带修正算法语义。
	if !bytes.Equal(values[2].([]byte), point) {
		t.Fatal("AggregateShare third output changed")
	}
}
