package cryptoupgrade

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

// Define Interface to avoid cricle import:"ethclient->core->upgradecrptoupgrade"
type client interface {
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// Parse event from receipt
func ParseReceipt(receipt *types.Receipt) {
	for _, vLog := range receipt.Logs {
		fmt.Println(vLog)
		if len(vLog.Topics) > 0 && vLog.Topics[0] == codeUploaded {
			fmt.Println("Event found in transaction logs!")
			// Event in codestorage.sol

			var name string
			err := CodeStorageABI.UnpackIntoInterface(&name, "codeUploaded", vLog.Data)
			if err != nil {
				fmt.Printf("Failed to unpack log data: %v", err)
			}
			fmt.Printf("Event data: Name=%s\n", name)
		}
	}
}

func BindCodeUploaded(client client) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.CodeStorageAddress},
		Topics:    [][]common.Hash{{codeUploaded}}, // Event hash
	}
	logCh := make(chan types.Log)
	// Subscribe to logs that meet FilterQuery,and logs will be stored in the logCh
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logCh)
	if err != nil {
		log.Error("Failed to subscribe to cryptoupgrade logs", "err", err)
		return
	}
	defer sub.Unsubscribe()

	// Subsribe coinbase add event
	eventHash := crypto.Keccak256Hash([]byte("CoinbaseAdded(string,string,uint256,address[],uint256[])"))
	coinSub, coinLogCh, err := bindCoinBaseEvent(client, eventHash)
	var coinErrCh <-chan error
	if err != nil {
		log.Error("Failed to subscribe to coinbase logs", "err", err)
		coinLogCh = nil
	} else {
		defer coinSub.Unsubscribe()
		coinErrCh = coinSub.Err()
	}
	CoinBaseABI, _ := abi.JSON(strings.NewReader(common.CoinbaseABI_json))

	for {
		select {
		case err := <-sub.Err():
			log.Error("Error while listening for cryptoupgrade log", "err", err)
		case Log := <-logCh:
			// Parse name from event
			log.Info("Catch codeUploaded event!")
			var name string
			err = CodeStorageABI.UnpackIntoInterface(&name, "codeUploaded", Log.Data)
			name = capitalString(name)
			if err != nil {
				log.Error("Failed to decode codeUploaded event", "err", err)
				continue
			} else if name == "" {
				log.Error("Err in parse name from @codeUploaded event")
				continue
			}

			// Lookup algorithm info from chain
			pc := lookupCodeInfo(client, name)
			if pc == nil {
				continue
			}
			if current, ok := getAlgorithmInfo(name); ok && current.code == pc.code && current.gas == pc.gas && current.itype == pc.itype && current.otype == pc.otype {
				log.Info("Upgrade algorithm is already active", "name", name)
				continue
			}

			err = ActivateAlgorithm(name, *pc)
			if err != nil {
				log.Error("Failed to activate upgrade algorithm", "name", name, "err", err)
				continue
			} else {
				goVerison := runtime.Version()
				log.Info("Activated upgrade algorithm", "name", name, "go", goVerison)
			}
		case err := <-coinErrCh:
			log.Error("Error while listening for coinbase logs", "err", err)
		case eventLog := <-coinLogCh:
			log.Info("Get Coinbase add event")
			vmap := make(map[string]interface{})
			err := CoinBaseABI.UnpackIntoMap(vmap, "CoinbaseAdded", eventLog.Data)
			if err != nil {
				log.Error("Failed to decode CoinbaseAdded event", "err", err)
			}
		}
	}

}

// Through Client call contract, get the infomation of @name algorithm
func lookupCodeInfo(client client, name string) *algoInfo {

	// Must equal to method in codestorage contract
	lookupFuncName := "getInfo"
	input, err := CodeStorageABI.Pack(lookupFuncName, name)
	if err != nil {
		log.Error("Failed to pack codeStorage getInfo input", "name", name, "err", err)
		return nil
	}
	msg := ethereum.CallMsg{
		To:   &common.CodeStorageAddress,
		Data: input,
	}
	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error("Failed to call codeStorage contract", "err", err)
		return nil
	}

	ci, err := CodeStorageABI.Unpack("getInfo", output)
	if err != nil {
		log.Error("Failed to unpack getInfo output from codeStorage contract", "err", err)
		return nil
	} else {
		log.Info("Loaded upgrade algorithm info", "name", name)
		// TODO exception handing
		return &algoInfo{
			code:  ci[0].(string),
			gas:   ci[1].(uint64),
			itype: ci[2].(string),
			otype: ci[3].(string),
		}
	}
}

// Check whether is callFunc in codestorage contract
func IsUpgradeAlgorithm(addr common.Address, funcSelector []byte) bool {
	if len(funcSelector) < 4 {
		return false
	}
	funcSelector = funcSelector[:4]
	return addr == common.CodeStorageAddress && bytes.Equal(CodeStorageABI.Methods["callFunc"].ID, funcSelector)
}

func bindCoinBaseEvent(client client, eventHash common.Hash) (ethereum.Subscription, chan types.Log, error) {
	query := ethereum.FilterQuery{
		Topics: [][]common.Hash{{eventHash}}, // Event hash
	}
	logCh := make(chan types.Log)
	// Subscribe to logs that meet FilterQuery,and logs will be stored in the logCh
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logCh)
	if err != nil {
		log.Error("Failed to subscribe to cryptoupgrade logs", "err", err)
		return nil, nil, err
	}
	return sub, logCh, nil
}
