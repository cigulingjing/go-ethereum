package network

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type testEthAPI struct {
	block uint64
}

func (*testEthAPI) ChainId() hexutil.Uint64 {
	return hexutil.Uint64(11223344)
}

func (api *testEthAPI) BlockNumber() hexutil.Uint64 {
	api.block++
	return hexutil.Uint64(api.block)
}

type testNetAPI struct{}

func (*testNetAPI) PeerCount() hexutil.Uint64 {
	return 0
}

type testCliqueAPI struct{}

func (*testCliqueAPI) GetSigners(string) []common.Address {
	return []common.Address{common.HexToAddress(testSigner)}
}

func TestValidateNetworkChecksOnlyNetworkState(t *testing.T) {
	server := rpc.NewServer()
	if err := server.RegisterName("eth", new(testEthAPI)); err != nil {
		t.Fatal(err)
	}
	if err := server.RegisterName("net", new(testNetAPI)); err != nil {
		t.Fatal(err)
	}
	if err := server.RegisterName("clique", new(testCliqueAPI)); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	cfg := &Config{
		Network:   NetworkConfig{ChainID: 11223344},
		Consensus: ConsensusConfig{Period: 0},
		Nodes: []NodeConfig{{
			ID:     "node1",
			Role:   "signer",
			RPCURL: httpServer.URL,
		}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := ValidateNetwork(ctx, cfg, ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || len(result.Nodes) != 1 {
		t.Fatalf("unexpected validation result: %+v", result)
	}
	if result.Nodes[0].EndBlock <= result.Nodes[0].StartBlock {
		t.Fatalf("block height did not increase: %+v", result.Nodes[0])
	}
}

func TestRequiredPeerCount(t *testing.T) {
	tests := []struct {
		nodes int
		min   int
		want  int
	}{
		{nodes: 1, min: 0, want: 0},
		{nodes: 5, min: 0, want: 4},
		{nodes: 5, min: 2, want: 2},
		{nodes: 5, min: 99, want: 4},
	}
	for _, tt := range tests {
		if got := requiredPeerCount(tt.nodes, tt.min); got != tt.want {
			t.Fatalf("requiredPeerCount(%d, %d) = %d, want %d", tt.nodes, tt.min, got, tt.want)
		}
	}
}
