package eth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func (s *Ethereum) configureCliqueSigner() error {
	if !s.config.Miner.Enabled {
		return nil
	}
	if s.blockchain.Config().Clique == nil {
		return errors.New("local mining is only supported for Clique private chains")
	}
	engine, ok := innerClique(s.engine)
	if !ok {
		return errors.New("clique chain is not backed by a Clique engine")
	}
	signer := s.config.Miner.PendingFeeRecipient
	if signer == (common.Address{}) && len(s.config.Miner.UnlockAccounts) > 0 {
		signer = s.config.Miner.UnlockAccounts[0]
		s.config.Miner.PendingFeeRecipient = signer
	}
	if signer == (common.Address{}) {
		return errors.New("clique sealing requires --unlock or --miner.pending.feeRecipient")
	}
	passphrase, err := readCliquePassword(s.config.Miner.PasswordFile)
	if err != nil {
		return err
	}
	ks, err := localKeyStore(s.accountManager)
	if err != nil {
		return err
	}
	for _, address := range appendUnique(s.config.Miner.UnlockAccounts, signer) {
		account := accounts.Account{Address: address}
		if err := ks.Unlock(account, passphrase); err != nil {
			return fmt.Errorf("unlock account %s: %w", address, err)
		}
		log.Info("Unlocked account for local Clique network", "address", address)
	}
	wallet, err := s.accountManager.Find(accounts.Account{Address: signer})
	if err != nil {
		return fmt.Errorf("find signer wallet %s: %w", signer, err)
	}
	engine.Authorize(signer, wallet.SignData)
	log.Info("Authorized local Clique signer", "address", signer)
	return nil
}

func innerClique(engine consensus.Engine) (*clique.Clique, bool) {
	if c, ok := engine.(*clique.Clique); ok {
		return c, true
	}
	type inner interface {
		InnerEngine() consensus.Engine
	}
	if wrapped, ok := engine.(inner); ok {
		return innerClique(wrapped.InnerEngine())
	}
	return nil, false
}

func localKeyStore(am *accounts.Manager) (*keystore.KeyStore, error) {
	keystores := am.Backends(keystore.KeyStoreType)
	if len(keystores) == 0 {
		return nil, errors.New("local keystore is not available")
	}
	return keystores[0].(*keystore.KeyStore), nil
}

func readCliquePassword(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	text, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read password file: %w", err)
	}
	lines := strings.Split(string(text), "\n")
	return strings.TrimRight(lines[0], "\r"), nil
}

func appendUnique(addresses []common.Address, address common.Address) []common.Address {
	for _, existing := range addresses {
		if existing == address {
			return addresses
		}
	}
	return append(addresses, address)
}

// CliqueAPI exposes the minimal Clique RPC surface required by the multi-node
// validation workflow.
type CliqueAPI struct {
	chain  *core.BlockChain
	engine *clique.Clique
}

func newCliqueAPI(eth *Ethereum) *CliqueAPI {
	engine, ok := innerClique(eth.engine)
	if !ok {
		return nil
	}
	return &CliqueAPI{chain: eth.blockchain, engine: engine}
}

// GetSigners returns the authorized Clique signer set at the requested block.
func (api *CliqueAPI) GetSigners(number rpc.BlockNumber) ([]common.Address, error) {
	header := headerByNumber(api.chain, number)
	if header == nil {
		return nil, fmt.Errorf("unknown block %v", number)
	}
	return api.engine.Signers(api.chain, header)
}

func headerByNumber(chain *core.BlockChain, number rpc.BlockNumber) *types.Header {
	switch number {
	case rpc.LatestBlockNumber, rpc.PendingBlockNumber, rpc.SafeBlockNumber, rpc.FinalizedBlockNumber:
		return chain.CurrentHeader()
	case rpc.EarliestBlockNumber:
		return chain.GetHeaderByNumber(0)
	default:
		if number < 0 {
			return nil
		}
		return chain.GetHeaderByNumber(uint64(number))
	}
}
