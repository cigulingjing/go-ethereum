// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package clique

import (
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// This test case is a repro of an annoying bug that took us forever to catch.
// In Clique PoA networks, consecutive blocks might have the same state root (no
// block subsidy, empty block). If a node crashes, the chain ends up losing the
// recent state and needs to regenerate it from blocks already in the database.
// The bug was that processing the block *prior* to an empty one **also
// completes** the empty one, ending up in a known-block error.
func TestReimportMirroredState(t *testing.T) {
	// Initialize a Clique chain with a single signer
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		engine = New(params.AllCliqueProtocolChanges.Clique, db)
		signer = new(types.HomesteadSigner)
	)
	genspec := &core.Genesis{
		Config:    params.AllCliqueProtocolChanges,
		ExtraData: make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc: map[common.Address]types.Account{
			addr: {Balance: big.NewInt(10000000000000000)},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])

	// Generate a batch of blocks, each properly signed
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), genspec, engine, nil)
	defer chain.Stop()

	_, blocks, _ := core.GenerateChainWithGenesis(genspec, engine, 3, func(i int, block *core.BlockGen) {
		// The chain maker doesn't have access to a chain, so the difficulty will be
		// lets unset (nil). Set it here to the correct value.
		block.SetDifficulty(diffInTurn)

		// We want to simulate an empty middle block, having the same state as the
		// first one. The last is needs a state change again to force a reorg.
		if i != 1 {
			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(addr), common.Address{0x00}, new(big.Int), params.TxGas, block.BaseFee(), nil), signer, key)
			if err != nil {
				panic(err)
			}
			block.AddTxWithChain(chain, tx)
		}
	})
	for i, block := range blocks {
		header := block.Header()
		if i > 0 {
			header.ParentHash = blocks[i-1].Hash()
		}
		header.Extra = make([]byte, extraVanity+extraSeal)
		header.Difficulty = diffInTurn

		sig, _ := crypto.Sign(SealHash(header).Bytes(), key)
		copy(header.Extra[len(header.Extra)-extraSeal:], sig)
		blocks[i] = block.WithSeal(header)
	}
	// Insert the first two blocks and make sure the chain is valid
	db = rawdb.NewMemoryDatabase()
	chain, _ = core.NewBlockChain(db, genspec, engine, nil)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert initial blocks: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 2 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 2)
	}

	// Simulate a crash by creating a new chain on top of the database, without
	// flushing the dirty states out. Insert the last block, triggering a sidechain
	// reimport.
	chain, _ = core.NewBlockChain(db, genspec, engine, nil)
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks[2:]); err != nil {
		t.Fatalf("failed to insert final block: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 3 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 3)
	}
}

func TestSealHash(t *testing.T) {
	have := SealHash(&types.Header{
		Difficulty: new(big.Int),
		Number:     new(big.Int),
		Extra:      make([]byte, 32+65),
		BaseFee:    new(big.Int),
	})
	want := common.HexToHash("0xbd3d1fa43fbc4c5bfcc91b179ec92e2861df3654de60468beb908ff805359e8f")
	if have != want {
		t.Errorf("have %x, want %x", have, want)
	}
}

func TestSealAuthorizedSigner(t *testing.T) {
	engine, chain, key, addr := newSealTestChain(t)
	engine.Authorize(addr, testSignerFn(key))

	block := newSealTestBlock(t, engine, chain)
	results := make(chan *types.Block, 1)
	if err := engine.Seal(chain, block, results, make(chan struct{})); err != nil {
		t.Fatalf("Seal failed: %v", err)
	}
	select {
	case sealed := <-results:
		if err := engine.VerifyHeader(chain, sealed.Header()); err != nil {
			t.Fatalf("sealed header failed verification: %v", err)
		}
		signer, err := engine.Author(sealed.Header())
		if err != nil {
			t.Fatalf("failed to recover signer: %v", err)
		}
		if signer != addr {
			t.Fatalf("signer mismatch: have %s want %s", signer, addr)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sealed block")
	}
}

func TestSealRejectsUnauthorizedSigner(t *testing.T) {
	engine, chain, _, _ := newSealTestChain(t)
	key, _ := crypto.GenerateKey()
	engine.Authorize(crypto.PubkeyToAddress(key.PublicKey), testSignerFn(key))

	err := engine.Seal(chain, newSealTestBlock(t, engine, chain), make(chan *types.Block, 1), make(chan struct{}))
	if err != errUnauthorizedSigner {
		t.Fatalf("unexpected error: have %v want %v", err, errUnauthorizedSigner)
	}
}

func TestSealRejectsMissingSignerFn(t *testing.T) {
	engine, chain, _, addr := newSealTestChain(t)
	engine.Authorize(addr, nil)

	err := engine.Seal(chain, newSealTestBlock(t, engine, chain), make(chan *types.Block, 1), make(chan struct{}))
	if err != errMissingSignerFn {
		t.Fatalf("unexpected error: have %v want %v", err, errMissingSignerFn)
	}
}

func TestSealStopChannelInterruptsResult(t *testing.T) {
	engine, chain, key, addr := newSealTestChain(t)
	engine.Authorize(addr, testSignerFn(key))

	block := newSealTestBlock(t, engine, chain)
	header := block.Header()
	header.Time = uint64(time.Now().Add(time.Second).Unix())
	block = block.WithSeal(header)

	results := make(chan *types.Block, 1)
	stop := make(chan struct{})
	if err := engine.Seal(chain, block, results, stop); err != nil {
		t.Fatalf("Seal failed: %v", err)
	}
	close(stop)
	select {
	case sealed := <-results:
		t.Fatalf("unexpected sealed block after stop: %v", sealed.Hash())
	case <-time.After(100 * time.Millisecond):
	}
}

func newSealTestChain(t *testing.T) (*Clique, *core.BlockChain, *ecdsa.PrivateKey, common.Address) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	db := rawdb.NewMemoryDatabase()
	engine := New(params.AllCliqueProtocolChanges.Clique, db)
	genspec := &core.Genesis{
		Config:     params.AllCliqueProtocolChanges,
		Difficulty: diffInTurn,
		GasLimit:   8000000,
		ExtraData:  make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc:      types.GenesisAlloc{addr: {Balance: big.NewInt(10000000000000000)}},
		BaseFee:    big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])
	chain, err := core.NewBlockChain(db, genspec, engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(chain.Stop)
	return engine, chain, key, addr
}

func newSealTestBlock(t *testing.T, engine *Clique, chain *core.BlockChain) *types.Block {
	t.Helper()
	parent := chain.CurrentHeader()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:   parent.GasLimit,
		Time:       uint64(time.Now().Unix()),
		UncleHash:  uncleHash,
		BaseFee:    eip1559.CalcBaseFee(chain.Config(), parent),
	}
	if err := engine.Prepare(chain, header); err != nil {
		t.Fatal(err)
	}
	return types.NewBlockWithHeader(header)
}

func testSignerFn(key *ecdsa.PrivateKey) SignerFn {
	return func(account accounts.Account, mimeType string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), key)
	}
}
