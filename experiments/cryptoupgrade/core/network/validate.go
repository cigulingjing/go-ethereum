package network

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// ValidateOptions 控制多节点网络验证行为。
type ValidateOptions struct {
	Timeout      time.Duration
	MinPeerCount int
}

// ValidationResult 是多节点网络验证的机器可读结果。
type ValidationResult struct {
	ConfigPath        string           `json:"configPath"`
	ChainID           uint64           `json:"chainId"`
	ExpectedPeerCount int              `json:"expectedPeerCount"`
	MinimumPeerCount  int              `json:"minimumPeerCount"`
	Nodes             []NodeValidation `json:"nodes"`
	OK                bool             `json:"ok"`
}

// NodeValidation 记录单个节点的验证结果。
type NodeValidation struct {
	ID         string   `json:"id"`
	Role       string   `json:"role"`
	RPCURL     string   `json:"rpcURL"`
	ChainID    uint64   `json:"chainId,omitempty"`
	PeerCount  uint64   `json:"peerCount,omitempty"`
	StartBlock uint64   `json:"startBlock,omitempty"`
	EndBlock   uint64   `json:"endBlock,omitempty"`
	Signers    []string `json:"signers,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// ValidateNetwork 只检查节点 RPC、chain ID、peer、出块和 Clique signer 状态。
func ValidateNetwork(ctx context.Context, cfg *Config, opts ValidateOptions) (*ValidationResult, error) {
	if opts.Timeout == 0 {
		opts.Timeout = time.Duration(cfg.Consensus.CliquePeriod()+2) * time.Second
	}
	result := &ValidationResult{
		ConfigPath:        cfg.ConfigPath(),
		ChainID:           cfg.Network.ChainID,
		ExpectedPeerCount: max(0, len(cfg.Nodes)-1),
		MinimumPeerCount:  requiredPeerCount(len(cfg.Nodes), opts.MinPeerCount),
		OK:                true,
	}
	wait := time.Duration(cfg.Consensus.CliquePeriod()+1) * time.Second
	results := make([]NodeValidation, len(cfg.Nodes))
	var wg sync.WaitGroup
	for i, node := range cfg.Nodes {
		wg.Add(1)
		go func(i int, node NodeConfig) {
			defer wg.Done()
			results[i] = validateNode(ctx, cfg, node, wait, result.MinimumPeerCount)
		}(i, node)
	}
	wg.Wait()
	for _, nr := range results {
		if nr.Error != "" {
			result.OK = false
		}
		result.Nodes = append(result.Nodes, nr)
	}
	return result, nil
}

// RPCURL 返回从宿主机访问节点 HTTP RPC 的 URL。
func RPCURL(node NodeConfig) string {
	if node.RPCURL != "" {
		return node.RPCURL
	}
	return fmt.Sprintf("http://127.0.0.1:%d", node.HTTPHostPort)
}

func validateNode(ctx context.Context, cfg *Config, node NodeConfig, wait time.Duration, minPeerCount int) NodeValidation {
	var last NodeValidation
	for {
		result := validateNodeOnce(ctx, cfg, node, wait, minPeerCount)
		if result.Error == "" {
			return result
		}
		last = result
		select {
		case <-ctx.Done():
			if last.Error == "" {
				last.Error = ctx.Err().Error()
			}
			return last
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func validateNodeOnce(ctx context.Context, cfg *Config, node NodeConfig, wait time.Duration, minPeerCount int) NodeValidation {
	rpcURL := RPCURL(node)
	out := NodeValidation{ID: node.ID, Role: node.Role, RPCURL: rpcURL}
	client, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer client.Close()
	var chainID hexutil.Big
	if err := client.CallContext(ctx, &chainID, "eth_chainId"); err != nil {
		out.Error = err.Error()
		return out
	}
	out.ChainID = (*big.Int)(&chainID).Uint64()
	if out.ChainID != cfg.Network.ChainID {
		out.Error = fmt.Sprintf("chain id mismatch: want %d got %d", cfg.Network.ChainID, out.ChainID)
		return out
	}
	var peerCount hexutil.Uint64
	if err := client.CallContext(ctx, &peerCount, "net_peerCount"); err != nil {
		out.Error = err.Error()
		return out
	}
	out.PeerCount = uint64(peerCount)
	var start hexutil.Uint64
	if err := client.CallContext(ctx, &start, "eth_blockNumber"); err != nil {
		out.Error = err.Error()
		return out
	}
	out.StartBlock = uint64(start)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		out.Error = ctx.Err().Error()
		return out
	case <-timer.C:
	}
	var end hexutil.Uint64
	if err := client.CallContext(ctx, &end, "eth_blockNumber"); err != nil {
		out.Error = err.Error()
		return out
	}
	out.EndBlock = uint64(end)
	var signers []common.Address
	if err := client.CallContext(ctx, &signers, "clique_getSigners", "latest"); err == nil {
		for _, signer := range signers {
			out.Signers = append(out.Signers, signer.Hex())
		}
	}
	if err := client.CallContext(ctx, &peerCount, "net_peerCount"); err != nil {
		out.Error = err.Error()
		return out
	}
	out.PeerCount = uint64(peerCount)
	if int(out.PeerCount) < minPeerCount {
		out.Error = fmt.Sprintf("peer count too low: want >= %d got %d", minPeerCount, out.PeerCount)
		return out
	}
	// period=0 时无交易不出块，不能用高度增长判断网络是否可用。
	if cfg.Consensus.CliquePeriod() > 0 && out.EndBlock <= out.StartBlock {
		out.Error = fmt.Sprintf("block height did not increase: start %d end %d", out.StartBlock, out.EndBlock)
		return out
	}
	return out
}

func requiredPeerCount(nodeCount, minPeerCount int) int {
	fullPeerCount := max(0, nodeCount-1)
	if minPeerCount <= 0 {
		return fullPeerCount
	}
	if minPeerCount > fullPeerCount {
		return fullPeerCount
	}
	return minPeerCount
}
