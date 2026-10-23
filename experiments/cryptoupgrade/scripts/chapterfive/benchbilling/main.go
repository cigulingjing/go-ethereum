// benchbilling collects actual one-transaction-per-block PBKDF2 receipts on a private chain.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/cryptoupgrade/pricing"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/pbkdf2"
)

var levels = []uint64{1, 2, 3, 4, 5, 6, 8, 12, 16}

type transaction struct {
	From common.Address  `json:"from"`
	To   *common.Address `json:"to,omitempty"`
	Gas  hexutil.Uint64  `json:"gas"`
	Data hexutil.Bytes   `json:"data,omitempty"`
}

type block struct {
	Hash         common.Hash   `json:"hash"`
	ParentHash   common.Hash   `json:"parentHash"`
	Transactions []common.Hash `json:"transactions"`
}

type sample struct {
	Path            string      `json:"path"`
	CallsPerTx      uint64      `json:"callsPerTx"`
	Iterations      uint64      `json:"iterations"`
	Round           int         `json:"round"`
	Repeat          int         `json:"repeat"`
	GasUsed         uint64      `json:"gasUsed"`
	Status          uint64      `json:"status"`
	TxHash          common.Hash `json:"txHash"`
	BlockHash       common.Hash `json:"blockHash"`
	ParentHash      common.Hash `json:"parentHash"`
	ParentCalls     uint64      `json:"parentCalls"`
	ParentTxHash    common.Hash `json:"parentTxHash"`
	SeparatorTxHash common.Hash `json:"separatorTxHash"`
	OutputHash      common.Hash `json:"outputHash"`
	QuotedCallGas   []uint64    `json:"quotedCallGas,omitempty"`
	Error           string      `json:"error,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	rpcURL := flag.String("rpc", "http://127.0.0.1:8545", "private single-node RPC")
	source := flag.String("source", "experiments/cryptoupgrade/algorithm/contracts/src/Pbkdf2Batch.sol", "Solidity batch source")
	solc := flag.String("solc", "/home/liuqi/project/.tools/solc/solc-0.8.26", "Solidity compiler")
	reference := flag.String("reference", "", "deployed PBKDF2 Solidity reference address")
	genesisPath := flag.String("genesis", "", "genesis.json for this experiment chain")
	output := flag.String("output", "", "new results directory")
	rounds := flag.Int("rounds", 3, "independent rounds")
	repeats := flag.Int("repeats", 10, "receipts per path per level per round")
	warmup := flag.Int("warmup", 10, "eth_call and mined warmups per path and level")
	iterations := flag.Uint64("iterations", 10, "fixed PBKDF2 internal iterations")
	baseGas := flag.Uint64("base-gas", 3000, "expected PBKDF2 base gas frozen in genesis")
	alphaNumerator := flag.Uint64("alpha-numerator", 1, "expected alpha numerator frozen in genesis")
	alphaDenominator := flag.Uint64("alpha-denominator", 2, "expected alpha denominator frozen in genesis")
	threshold := flag.Uint64("threshold", 2, "expected intra-transaction threshold frozen in genesis")
	flag.Parse()
	if !common.IsHexAddress(*reference) || *output == "" || *genesisPath == "" || *rounds < 1 || *repeats < 1 || *iterations < 1 {
		return errors.New("reference, genesis, new output directory, rounds, repeats and iterations are required")
	}
	if err := os.Mkdir(*output, 0755); err != nil {
		return err
	}
	ctx := context.Background()
	client, err := rpc.DialContext(ctx, *rpcURL)
	if err != nil {
		return err
	}
	defer client.Close()
	var accounts []common.Address
	if err := client.CallContext(ctx, &accounts, "eth_accounts"); err != nil || len(accounts) != 1 {
		return fmt.Errorf("requires exactly one unlocked account: %v, %d", err, len(accounts))
	}
	from := accounts[0]
	contractABI, bytecode, err := compile(*solc, *source)
	if err != nil {
		return err
	}
	args, err := contractABI.Pack("", common.HexToAddress(*reference))
	if err != nil {
		return err
	}
	_, deployed, err := submit(ctx, client, transaction{From: from, Gas: 28_000_000, Data: append(bytecode, args...)})
	if err != nil {
		return fmt.Errorf("batch deployment failed: %w", err)
	}
	if deployed.Status != 1 {
		return fmt.Errorf("batch deployment failed: status=%d", deployed.Status)
	}
	address := deployed.ContractAddress
	var genesis block
	if err := client.CallContext(ctx, &genesis, "eth_getBlockByNumber", "0x0", false); err != nil {
		return err
	}
	profile := pricing.Profile{Threshold: *threshold, Alpha: pricing.Ratio{Numerator: *alphaNumerator, Denominator: *alphaDenominator},
		Algorithms: []pricing.Algorithm{{Name: "Pbkdf2Sha256", Version: 1, Metric: "encodedInputBytes",
			Bands: []pricing.InputBand{{MaxMetric: ^uint64(0), Gas: *baseGas}}}},
		Load: []pricing.LoadBand{{MaxCalls: ^uint64(0), Multiplier: pricing.Ratio{Numerator: 1, Denominator: 1}}}}
	if err := profile.Validate(); err != nil {
		return err
	}
	genesisJSON, err := os.ReadFile(*genesisPath)
	if err != nil {
		return err
	}
	var genesisConfig struct {
		Config struct {
			Pricing *params.CryptoUpgradePricingConfig `json:"cryptoUpgradePricing"`
		} `json:"config"`
	}
	if err := json.Unmarshal(genesisJSON, &genesisConfig); err != nil {
		return err
	}
	if genesisConfig.Config.Pricing == nil || genesisConfig.Config.Pricing.ActivationBlock > deployed.BlockNumber.Uint64() || !reflect.DeepEqual(genesisConfig.Config.Pricing.Profile, profile) {
		return errors.New("genesis tariff does not match frozen experiment profile or has not activated")
	}
	genesisDigest := sha256.Sum256(genesisJSON)
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	profileDigest := sha256.Sum256(profileJSON)
	derived := pbkdf2.Key([]byte("password"), []byte("salt"), int(*iterations), 32, sha256.New)
	expected := crypto.Keccak256Hash(derived)
	manifest := map[string]interface{}{"section": "5.2.2", "nodeCount": 1, "genesisHash": genesis.Hash,
		"batchAddress": address, "referenceAddress": *reference, "profile": profile, "levels": levels,
		"genesisFileSha256": hex.EncodeToString(genesisDigest[:]),
		"profileSha256":     hex.EncodeToString(profileDigest[:]),
		"rounds":            *rounds, "repeats": *repeats, "ethCallWarmupPerPathAndLevel": *warmup,
		"minedWarmupPerPathAndLevel": *warmup,
		"iterations":                 *iterations, "expectedOutputHash": expected,
		"gasBoundary":       "mined receipt.gasUsed including intrinsic and calldata",
		"parentLoad":        "parent is a neutral self-transfer or empty block; empty blocks between target and preceding transfer are checked; M(C)=1 fixed at genesis",
		"solidityPrimitive": "SHA-256 precompile used within Solidity HMAC",
		"warning":           "configured base gas is not a measured execution-time advantage; transaction-local tariff does not imply cross-transaction attack resistance"}
	if err := writeJSON(filepath.Join(*output, "manifest.json"), manifest); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(*output, "samples.jsonl"))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, count := range levels {
		for _, method := range []string{"solidityCalls", "evocryptCalls"} {
			data, err := contractABI.Pack(method, new(big.Int).SetUint64(count), new(big.Int).SetUint64(*iterations), expected)
			if err != nil {
				return err
			}
			for index := 0; index < *warmup; index++ {
				if err := checkCall(ctx, client, from, address, data, contractABI, method, count, derived); err != nil {
					return fmt.Errorf("preflight %s n=%d: %w", method, count, err)
				}
			}
		}
	}
	warmupFile, err := os.Create(filepath.Join(*output, "warmup.jsonl"))
	if err != nil {
		return err
	}
	defer warmupFile.Close()
	warmupEncoder := json.NewEncoder(warmupFile)
	for _, count := range levels {
		for _, method := range []string{"solidityCalls", "evocryptCalls"} {
			data, err := contractABI.Pack(method, new(big.Int).SetUint64(count), new(big.Int).SetUint64(*iterations), expected)
			if err != nil {
				return err
			}
			for index := 0; index < *warmup; index++ {
				_, separatorReceipt, err := submit(ctx, client, transaction{From: from, To: &from, Gas: 21_000})
				if err != nil {
					return fmt.Errorf("warmup neutral parent %s n=%d: %w", method, count, err)
				}
				if separatorReceipt.Status != 1 {
					return fmt.Errorf("warmup neutral parent %s n=%d: status=%d", method, count, separatorReceipt.Status)
				}
				hash, receipt, err := submit(ctx, client, transaction{From: from, To: &address, Gas: 28_000_000, Data: data})
				gasUsed := uint64(0)
				status := uint64(0)
				if receipt != nil {
					gasUsed, status = receipt.GasUsed, receipt.Status
				}
				record := map[string]interface{}{"path": method, "callsPerTx": count, "sample": index + 1,
					"txHash": hash, "gasUsed": gasUsed, "status": status}
				if err != nil {
					record["error"] = err.Error()
				}
				if err := warmupEncoder.Encode(record); err != nil {
					return err
				}
				if err != nil || status != 1 {
					return fmt.Errorf("mined warmup %s n=%d: %v status=%d", method, count, err, status)
				}
			}
		}
	}
	for round := 1; round <= *rounds; round++ {
		for _, count := range levels {
			for _, method := range []string{"solidityCalls", "evocryptCalls"} {
				data, err := contractABI.Pack(method, new(big.Int).SetUint64(count), new(big.Int).SetUint64(*iterations), expected)
				if err != nil {
					return err
				}
				for repeat := 1; repeat <= *repeats; repeat++ {
					entry := sample{Path: method, CallsPerTx: count, Iterations: *iterations, Round: round, Repeat: repeat, OutputHash: expected}
					if method == "evocryptCalls" {
						for index := uint64(1); index <= count; index++ {
							quoted, err := profile.Quote("Pbkdf2Sha256", 1, 0, index, 0)
							if err != nil {
								return err
							}
							entry.QuotedCallGas = append(entry.QuotedCallGas, quoted)
						}
					}
					separator, separatorReceipt, err := submit(ctx, client, transaction{From: from, To: &from, Gas: 21_000})
					if err != nil {
						return fmt.Errorf("neutral parent failed: %w", err)
					}
					if separatorReceipt.Status != 1 {
						return fmt.Errorf("neutral parent failed: %v", err)
					}
					txHash, receipt, err := submit(ctx, client, transaction{From: from, To: &address, Gas: 28_000_000, Data: data})
					entry.TxHash, entry.SeparatorTxHash = txHash, separator
					if err == nil {
						entry.Status, entry.GasUsed, entry.BlockHash = receipt.Status, receipt.GasUsed, receipt.BlockHash
						var target block
						err = client.CallContext(ctx, &target, "eth_getBlockByHash", receipt.BlockHash, false)
						if err == nil {
							entry.ParentHash = target.ParentHash
							if len(target.Transactions) != 1 || target.Transactions[0] != txHash {
								err = errors.New("target block must contain only its target transaction")
							} else {
								entry.ParentTxHash, err = checkNeutralAncestors(ctx, client, target.ParentHash, separatorReceipt.BlockHash, separator)
							}
						}
						if err == nil && receipt.Status == 1 {
							err = checkDigest(ctx, client, from, address, contractABI, count, derived)
						}
						if err == nil && receipt.Status != 1 {
							err = fmt.Errorf("receipt status %d", receipt.Status)
						}
					}
					if err != nil {
						entry.Error = err.Error()
					}
					if err := encoder.Encode(entry); err != nil {
						return err
					}
					if entry.Error != "" {
						return fmt.Errorf("%s n=%d round=%d repeat=%d: %s", method, count, round, repeat, entry.Error)
					}
				}
			}
		}
	}
	return nil
}

func checkNeutralAncestors(ctx context.Context, client *rpc.Client, parentHash, separatorBlock, separatorTx common.Hash) (common.Hash, error) {
	for depth := 0; depth < 32; depth++ {
		var parent block
		if err := client.CallContext(ctx, &parent, "eth_getBlockByHash", parentHash, false); err != nil {
			return common.Hash{}, err
		}
		if parent.Hash == separatorBlock {
			if len(parent.Transactions) != 1 || parent.Transactions[0] != separatorTx {
				return common.Hash{}, errors.New("neutral transfer block does not contain the separator")
			}
			if depth == 0 {
				return separatorTx, nil
			}
			return common.Hash{}, nil
		}
		if parent.Hash == (common.Hash{}) || len(parent.Transactions) != 0 {
			return common.Hash{}, errors.New("parent branch contains another transaction")
		}
		parentHash = parent.ParentHash
	}
	return common.Hash{}, errors.New("neutral separator not found before parent block")
}

func compile(solc, source string) (abi.ABI, []byte, error) {
	output, err := exec.Command(solc, "--optimize", "--via-ir", "--evm-version", "paris", "--combined-json", "abi,bin", source).CombinedOutput()
	if err != nil {
		return abi.ABI{}, nil, fmt.Errorf("solc: %w: %s", err, output)
	}
	var contracts struct {
		Contracts map[string]struct {
			ABI json.RawMessage `json:"abi"`
			Bin string          `json:"bin"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(output, &contracts); err != nil {
		return abi.ABI{}, nil, err
	}
	for name, contract := range contracts.Contracts {
		if !strings.HasSuffix(name, ":Pbkdf2Batch") {
			continue
		}
		parsed, err := abi.JSON(bytes.NewReader(contract.ABI))
		code, decodeErr := hex.DecodeString(contract.Bin)
		if err != nil {
			return abi.ABI{}, nil, err
		}
		return parsed, code, decodeErr
	}
	return abi.ABI{}, nil, errors.New("missing Pbkdf2Batch contract")
}

func submit(ctx context.Context, client *rpc.Client, tx transaction) (common.Hash, *types.Receipt, error) {
	var hash common.Hash
	if err := client.CallContext(ctx, &hash, "eth_sendTransaction", tx); err != nil {
		return hash, nil, err
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		var receipt *types.Receipt
		if err := client.CallContext(ctx, &receipt, "eth_getTransactionReceipt", hash); err == nil && receipt != nil {
			return hash, receipt, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return hash, nil, fmt.Errorf("receipt timeout %s", hash)
}

func checkCall(ctx context.Context, client *rpc.Client, from, address common.Address, data []byte, parsed abi.ABI, method string, count uint64, derived []byte) error {
	var result hexutil.Bytes
	if err := client.CallContext(ctx, &result, "eth_call", transaction{From: from, To: &address, Gas: 28_000_000, Data: data}, "latest"); err != nil {
		return err
	}
	return checkOutput(result, parsed, method, count, derived)
}

func checkDigest(ctx context.Context, client *rpc.Client, from, address common.Address, parsed abi.ABI, count uint64, derived []byte) error {
	data, err := parsed.Pack("lastDigest")
	if err != nil {
		return err
	}
	var result hexutil.Bytes
	if err := client.CallContext(ctx, &result, "eth_call", transaction{From: from, To: &address, Gas: 28_000_000, Data: data}, "latest"); err != nil {
		return err
	}
	return checkOutput(result, parsed, "lastDigest", count, derived)
}

func checkOutput(raw []byte, parsed abi.ABI, method string, count uint64, derived []byte) error {
	values, err := parsed.Unpack(method, raw)
	if err != nil {
		return err
	}
	var digest common.Hash
	for index := uint64(0); index < count; index++ {
		digest = crypto.Keccak256Hash(digest[:], derived)
	}
	if len(values) != 1 || values[0].([32]byte) != digest {
		return fmt.Errorf("PBKDF2 batch digest mismatch at n=%d", count)
	}
	return nil
}

func writeJSON(path string, value interface{}) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0644)
}
