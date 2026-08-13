// Package smoke implements end-to-end cryptoupgrade checks for experiment networks.
package smoke

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/experiments/cryptoupgrade/network"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultTimeout      = 30 * time.Second
	defaultPollInterval = 500 * time.Millisecond
)

// Options controls the Add upload and activation check.
type Options struct {
	Timeout      time.Duration
	PollInterval time.Duration
}

// Result records the cryptoupgrade Add smoke test using the legacy JSON fields.
type Result struct {
	NodeID     string `json:"nodeId"`
	From       string `json:"from"`
	UploadHash string `json:"uploadHash,omitempty"`
	Output     string `json:"output,omitempty"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
}

type backend interface {
	SendTransaction(context.Context, map[string]interface{}) (common.Hash, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	CallContract(context.Context, ethereum.CallMsg) ([]byte, error)
	Close()
}

type rpcBackend struct {
	rpc *rpc.Client
	eth *ethclient.Client
}

// RunAdd uploads the configured Add source and waits until callFunc proves local activation.
func RunAdd(ctx context.Context, cfg *network.Config, opts Options) *Result {
	node, ok := firstSignerNode(cfg)
	if !ok {
		return &Result{Error: "no signer node available for cryptoupgrade smoke test"}
	}
	result := &Result{NodeID: node.ID, From: common.HexToAddress(node.Account).Hex()}
	client, err := newRPCBackend(ctx, network.RPCURL(node))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer client.Close()

	source, err := cryptoupgrade.EncodeSourceFile(cfg.CryptoUpgrade.AddSource)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	return runAdd(ctx, node, source, opts, client)
}

func runAdd(ctx context.Context, node network.NodeConfig, source string, opts Options, client backend) *Result {
	result := &Result{NodeID: node.ID, From: common.HexToAddress(node.Account).Hex()}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultPollInterval
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	uploadData, err := cryptoupgrade.CodeStorageABI.Pack("uploadCode", "Add", source, uint64(1), "uint256,uint256", "uint256")
	if err != nil {
		result.Error = err.Error()
		return result
	}
	from := common.HexToAddress(node.Account)
	txHash, err := client.SendTransaction(ctx, map[string]interface{}{
		"from": from,
		"to":   common.CodeStorageAddress,
		"gas":  hexutil.Uint64(5000000),
		"data": hexutil.Bytes(uploadData),
	})
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.UploadHash = txHash.Hex()

	receipt, err := waitReceipt(ctx, client, txHash, opts.PollInterval)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		result.Error = fmt.Sprintf("upload transaction failed with status %d", receipt.Status)
		return result
	}

	callData, err := packAddCallData()
	if err != nil {
		result.Error = err.Error()
		return result
	}
	// receipt 只证明 uploadCode 已上链；bind-event 在节点侧异步编译和激活，
	// 因此必须持续调用 callFunc，直到新实现真正可用或整体超时。
	output, err := waitActivated(ctx, client, ethereum.CallMsg{
		From: from,
		To:   &common.CodeStorageAddress,
		Gas:  5000000,
		Data: callData,
	}, opts.PollInterval)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Output = hexutil.Encode(output)
	result.OK = true
	return result
}

func newRPCBackend(ctx context.Context, url string) (*rpcBackend, error) {
	client, err := rpc.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return &rpcBackend{rpc: client, eth: ethclient.NewClient(client)}, nil
}

func (b *rpcBackend) SendTransaction(ctx context.Context, tx map[string]interface{}) (common.Hash, error) {
	var hash common.Hash
	err := b.rpc.CallContext(ctx, &hash, "eth_sendTransaction", tx)
	return hash, err
}

func (b *rpcBackend) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	return b.eth.TransactionReceipt(ctx, hash)
}

func (b *rpcBackend) CallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error) {
	return b.eth.CallContract(ctx, msg, nil)
}

func (b *rpcBackend) Close() {
	b.rpc.Close()
}

func waitReceipt(ctx context.Context, client backend, hash common.Hash, interval time.Duration) (*types.Receipt, error) {
	var lastErr error
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		lastErr = err
		if err := waitPoll(ctx, interval); err != nil {
			return nil, fmt.Errorf("timed out waiting for receipt %s: %w", hash, errors.Join(lastErr, err))
		}
	}
}

func waitActivated(ctx context.Context, client backend, msg ethereum.CallMsg, interval time.Duration) ([]byte, error) {
	var lastErr error
	for {
		output, err := client.CallContract(ctx, msg)
		if err == nil {
			value, decodeErr := unpackUint256(output)
			if decodeErr == nil && value.Cmp(big.NewInt(200)) == 0 {
				return output, nil
			}
			if decodeErr != nil {
				lastErr = decodeErr
			} else {
				lastErr = fmt.Errorf("unexpected Add output: want 200 got %s", value)
			}
		} else {
			lastErr = err
		}
		if err := waitPoll(ctx, interval); err != nil {
			return nil, fmt.Errorf("timed out waiting for event-driven activation: %w", errors.Join(lastErr, err))
		}
	}
}

func waitPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func firstSignerNode(cfg *network.Config) (network.NodeConfig, bool) {
	for _, node := range cfg.Nodes {
		if node.Role == "signer" && common.IsHexAddress(node.Account) {
			return node, true
		}
	}
	return network.NodeConfig{}, false
}

func packAddCallData() ([]byte, error) {
	uint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	inputs := abi.Arguments{{Type: uint256}, {Type: uint256}}
	encodedInput, err := inputs.Pack(big.NewInt(100), big.NewInt(100))
	if err != nil {
		return nil, err
	}
	return cryptoupgrade.CodeStorageABI.Pack("callFunc", "Add", encodedInput)
}

func unpackUint256(output []byte) (*big.Int, error) {
	uint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	values, err := (abi.Arguments{{Type: uint256}}).Unpack(output)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("unexpected uint256 output count %d", len(values))
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected uint256 output type %T", values[0])
	}
	return value, nil
}
