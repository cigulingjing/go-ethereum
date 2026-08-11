package network

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ValidateOptions 控制多节点网络验证行为。
type ValidateOptions struct {
	Timeout            time.Duration
	CryptoUpgradeSmoke bool
}

// ValidationResult 是多节点网络验证的机器可读结果。
type ValidationResult struct {
	ConfigPath        string           `json:"configPath"`
	ChainID           uint64           `json:"chainId"`
	ExpectedPeerCount int              `json:"expectedPeerCount"`
	CryptoUpgrade     *SmokeTestResult `json:"cryptoUpgrade,omitempty"`
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

// SmokeTestResult 记录 cryptoupgrade Add smoke test 结果。
type SmokeTestResult struct {
	NodeID     string `json:"nodeId"`
	From       string `json:"from"`
	UploadHash string `json:"uploadHash,omitempty"`
	Output     string `json:"output,omitempty"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
}

// ValidateNetwork 检查节点 RPC、peer、Clique signer 和可选 cryptoupgrade 调用。
func ValidateNetwork(ctx context.Context, cfg *Config, opts ValidateOptions) (*ValidationResult, error) {
	if opts.Timeout == 0 {
		opts.Timeout = time.Duration(cfg.Consensus.Period+2) * time.Second
	}
	result := &ValidationResult{
		ConfigPath:        cfg.ConfigPath(),
		ChainID:           cfg.Network.ChainID,
		ExpectedPeerCount: max(0, len(cfg.Nodes)-1),
		OK:                true,
	}
	wait := time.Duration(cfg.Consensus.Period+1) * time.Second
	for _, node := range cfg.Nodes {
		nr := validateNode(ctx, cfg, node, wait)
		if nr.Error != "" {
			result.OK = false
		}
		result.Nodes = append(result.Nodes, nr)
	}
	if opts.CryptoUpgradeSmoke {
		result.CryptoUpgrade = runAddSmokeTest(ctx, cfg)
		if !result.CryptoUpgrade.OK {
			result.OK = false
		}
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

func validateNode(ctx context.Context, cfg *Config, node NodeConfig, wait time.Duration) NodeValidation {
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
	time.Sleep(wait)
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
	if int(out.PeerCount) < max(0, len(cfg.Nodes)-1) {
		out.Error = fmt.Sprintf("peer count too low: want >= %d got %d", max(0, len(cfg.Nodes)-1), out.PeerCount)
		return out
	}
	if out.EndBlock <= out.StartBlock {
		out.Error = fmt.Sprintf("block height did not increase: start %d end %d", out.StartBlock, out.EndBlock)
		return out
	}
	return out
}

func runAddSmokeTest(ctx context.Context, cfg *Config) *SmokeTestResult {
	node, ok := firstSignerNode(cfg)
	if !ok {
		return &SmokeTestResult{OK: false, Error: "no signer node available for cryptoupgrade smoke test"}
	}
	result := &SmokeTestResult{NodeID: node.ID, From: common.HexToAddress(node.Account).Hex()}
	rpcClient, err := rpc.DialContext(ctx, RPCURL(node))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer rpcClient.Close()
	ethClient := ethclient.NewClient(rpcClient)
	source, err := compressFile(nodeSmokeSource(cfg))
	if err != nil {
		result.Error = err.Error()
		return result
	}
	uploadData, err := cryptoupgrade.CodeStorageABI.Pack("uploadCode", "Add", source, uint64(1), "uint256,uint256", "uint256")
	if err != nil {
		result.Error = err.Error()
		return result
	}
	from := common.HexToAddress(node.Account)
	var txHash common.Hash
	tx := map[string]interface{}{
		"from": from,
		"to":   common.CodeStorageAddress,
		"gas":  hexutil.Uint64(5000000),
		"data": hexutil.Bytes(uploadData),
	}
	if err := rpcClient.CallContext(ctx, &txHash, "eth_sendTransaction", tx); err != nil {
		result.Error = err.Error()
		return result
	}
	result.UploadHash = txHash.Hex()
	receipt, err := waitReceipt(ctx, ethClient, txHash)
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
	output, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		From: from,
		To:   &common.CodeStorageAddress,
		Gas:  5000000,
		Data: callData,
	}, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Output = hexutil.Encode(output)
	decoded, err := unpackUint256(output)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if decoded.Cmp(big.NewInt(200)) != 0 {
		result.Error = fmt.Sprintf("unexpected Add output: want 200 got %s", decoded)
		return result
	}
	result.OK = true
	return result
}

func firstSignerNode(cfg *Config) (NodeConfig, bool) {
	for _, node := range cfg.Nodes {
		if node.Role == "signer" && common.IsHexAddress(node.Account) {
			return node, true
		}
	}
	return NodeConfig{}, false
}

func nodeSmokeSource(cfg *Config) string {
	return cfg.CryptoUpgrade.AddSource
}

func packAddCallData() ([]byte, error) {
	uint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{{Type: uint256}, {Type: uint256}}
	encodedInput, err := args.Pack(big.NewInt(100), big.NewInt(100))
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
	args := abi.Arguments{{Type: uint256}}
	values, err := args.Unpack(output)
	if err != nil {
		return nil, err
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected uint256 output type %T", values[0])
	}
	return value, nil
}

func compressFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(raw); err != nil {
		writer.Close()
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func waitReceipt(ctx context.Context, client *ethclient.Client, hash common.Hash) (*types.Receipt, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("timed out waiting for receipt %s", hash)
		case <-ticker.C:
		}
	}
}
