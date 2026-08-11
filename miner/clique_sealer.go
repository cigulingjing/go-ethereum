package miner

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Start launches the local Clique sealing loop when mining is explicitly
// enabled. 该循环只服务私链实验中的 signer 节点，observer 节点不会调用它。
func (miner *Miner) Start() {
	if !miner.config.Enabled || miner.chainConfig.Clique == nil {
		return
	}
	miner.sealingMu.Lock()
	defer miner.sealingMu.Unlock()
	if miner.sealingStop != nil {
		return
	}
	miner.sealingStop = make(chan struct{})
	miner.sealingDone = make(chan struct{})
	go miner.sealingLoop(miner.sealingStop, miner.sealingDone)
}

// Stop terminates the local sealing loop if it is running.
func (miner *Miner) Stop() {
	miner.sealingMu.Lock()
	stop, done := miner.sealingStop, miner.sealingDone
	if stop != nil {
		close(stop)
		miner.sealingStop = nil
		miner.sealingDone = nil
	}
	miner.sealingMu.Unlock()
	if done != nil {
		<-done
	}
}

func (miner *Miner) sealingLoop(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	log.Info("Starting local Clique sealer")
	for {
		select {
		case <-stop:
			log.Info("Stopped local Clique sealer")
			return
		default:
		}
		if err := miner.sealCliqueBlock(stop); err != nil {
			if !errors.Is(err, errSealingStopped) {
				log.Warn("Clique sealing round failed", "err", err)
			}
		}
		select {
		case <-stop:
			log.Info("Stopped local Clique sealer")
			return
		case <-time.After(time.Second):
		}
	}
}

var errSealingStopped = errors.New("clique sealing stopped")

func (miner *Miner) sealCliqueBlock(stop <-chan struct{}) error {
	miner.confMu.RLock()
	coinbase := miner.config.PendingFeeRecipient
	miner.confMu.RUnlock()
	if coinbase == (common.Address{}) {
		return errors.New("missing fee recipient for Clique sealing")
	}
	work := miner.generateWork(context.Background(), &generateParams{
		timestamp: uint64(time.Now().Unix()),
		coinbase:  coinbase,
		noTxs:     false,
	}, false)
	if work.err != nil {
		return work.err
	}
	results := make(chan *types.Block, 1)
	sealStop := make(chan struct{})
	if err := miner.engine.Seal(miner.chain, work.block, results, sealStop); err != nil {
		close(sealStop)
		return err
	}
	timeout := time.Until(time.Unix(int64(work.block.Time()), 0)) + time.Duration(miner.chainConfig.Clique.Period+5)*time.Second
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-stop:
		close(sealStop)
		return errSealingStopped
	case <-timer.C:
		close(sealStop)
		return errors.New("timed out waiting for Clique seal")
	case block := <-results:
		close(sealStop)
		if block == nil {
			return errors.New("Clique sealer returned nil block")
		}
		if _, err := miner.chain.InsertChain(types.Blocks{block}); err != nil {
			return err
		}
		miner.confMu.RLock()
		hook := miner.config.SealedBlockHook
		miner.confMu.RUnlock()
		if hook != nil {
			hook(block)
		}
		log.Info("Sealed Clique block", "number", block.NumberU64(), "hash", block.Hash(), "txs", len(block.Transactions()))
		return nil
	}
}
