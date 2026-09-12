package core

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade/stagelog"
	"github.com/ethereum/go-ethereum/params"
)

func TestStageTimingCanonicalInclusion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stage.jsonl")
	t.Setenv(stagelog.FileEnv, path)
	t.Cleanup(stagelog.Close)
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	genesis := &Genesis{Config: params.TestChainConfig, Alloc: types.GenesisAlloc{addr: {Balance: big.NewInt(10000000000000000)}}}
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), genesis, ethash.NewFaker(), DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	var tx *types.Transaction
	_, blocks, _ := GenerateChainWithGenesis(genesis, ethash.NewFaker(), 1, func(_ int, b *BlockGen) {
		tx, err = types.SignTx(types.NewTransaction(0, common.Address{0x12}, big.NewInt(1), 21000, b.header.BaseFee, nil), types.LatestSigner(genesis.Config), key)
		if err != nil {
			t.Fatal(err)
		}
		b.AddTx(tx)
	})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("candidate execution created inclusion record: %v", err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatal(string(raw))
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatal(err)
	}
	if event["stage"] != "transaction_included" || event["txHash"] != tx.Hash().Hex() || event["blockHash"] != chain.CurrentBlock().Hash().Hex() {
		t.Fatal(event)
	}
}
