package ethapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestStageTimingRPCTransactionHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stage.jsonl")
	t.Setenv(stagelog.FileEnv, path)
	t.Cleanup(stagelog.Close)
	b := newTestBackend(t, 0, &core.Genesis{Config: params.TestChainConfig, Alloc: types.GenesisAlloc{}}, ethash.NewFaker(), nil)
	defer b.chain.Stop()
	b.autoMine = true
	api := NewTransactionAPI(b, new(AddrLocker))
	raw, tx := makeSelfSignedRaw(t, api, b.acc.Address)
	server := rpc.NewServer()
	defer server.Stop()
	if err := server.RegisterName("eth", api); err != nil {
		t.Fatal(err)
	}
	client := rpc.DialInProc(server)
	defer client.Close()
	var got common.Hash
	if err := client.CallContext(context.Background(), &got, "eth_sendRawTransaction", raw); err != nil {
		t.Fatal(err)
	}
	if got != tx.Hash() {
		t.Fatal(got)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var received, bound map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(bytes)), "\n") {
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event["stage"] == "rpc_received" {
			received = event
		}
		if event["stage"] == "rpc_transaction" {
			bound = event
		}
	}
	if received == nil || bound == nil || received["requestId"] != bound["requestId"] || bound["txHash"] != got.Hex() {
		t.Fatal(received, bound)
	}
}
