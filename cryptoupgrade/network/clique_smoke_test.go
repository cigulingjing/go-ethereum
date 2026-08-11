package network

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestCliqueSealSmokeSealsBlock(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	db := rawdb.NewMemoryDatabase()
	engine := clique.New(params.AllCliqueProtocolChanges.Clique, db)
	engine.Authorize(addr, func(account accounts.Account, mimeType string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), key)
	})
	genesis := &core.Genesis{
		Config:     params.AllCliqueProtocolChanges,
		Difficulty: big.NewInt(1),
		GasLimit:   8000000,
		ExtraData:  make([]byte, 32+common.AddressLength+crypto.SignatureLength),
		Alloc:      types.GenesisAlloc{addr: {Balance: big.NewInt(10000000000000000)}},
		BaseFee:    big.NewInt(params.InitialBaseFee),
	}
	copy(genesis.ExtraData[32:], addr[:])
	chain, err := core.NewBlockChain(db, genesis, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(chain.Stop)

	parent := chain.CurrentHeader()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     big.NewInt(1),
		GasLimit:   parent.GasLimit,
		Time:       uint64(time.Now().Unix()),
		UncleHash:  types.EmptyUncleHash,
		BaseFee:    eip1559.CalcBaseFee(chain.Config(), parent),
	}
	if err := engine.Prepare(chain, header); err != nil {
		t.Fatal(err)
	}
	results := make(chan *types.Block, 1)
	if err := engine.Seal(chain, types.NewBlockWithHeader(header), results, make(chan struct{})); err != nil {
		t.Fatal(err)
	}
	select {
	case sealed := <-results:
		if err := engine.VerifyHeader(chain, sealed.Header()); err != nil {
			t.Fatalf("sealed header failed verification: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sealed block")
	}
}
