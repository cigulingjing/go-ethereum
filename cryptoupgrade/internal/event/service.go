// Package event adapts CodeStorage logs to the local activation service.
package event

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/cryptoupgrade/internal/model"
	"github.com/ethereum/go-ethereum/log"
)

// Client is the minimal chain client needed by the event adapter.
type Client interface {
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// Activator is the only application-service dependency used by event handling.
type Activator interface {
	Activate(ctx context.Context, name string, info model.AlgorithmInfo) error
}

// Service subscribes to CodeStorage events and delegates activation.
type Service struct {
	contractABI abi.ABI
	address     common.Address
	topic       common.Hash
	activator   Activator
}

// NewService creates a CodeStorage event adapter.
func NewService(contractABI abi.ABI, address common.Address, topic common.Hash, activator Activator) *Service {
	return &Service{
		contractABI: contractABI,
		address:     address,
		topic:       topic,
		activator:   activator,
	}
}

// Bind listens until the subscription ends and keeps activation failures local to the listener.
func (s *Service) Bind(ctx context.Context, client Client) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{s.address},
		Topics:    [][]common.Hash{{s.topic}},
	}
	logCh := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(ctx, query, logCh)
	if err != nil {
		log.Error("Failed to subscribe to cryptoupgrade logs", "err", err)
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-sub.Err():
			if !ok {
				log.Info("Stopped listening for cryptoupgrade logs")
				return
			}
			log.Error("Error while listening for cryptoupgrade log", "err", err)
		case eventLog, ok := <-logCh:
			if !ok {
				return
			}
			if err := s.Handle(ctx, client, eventLog); err != nil {
				log.Error("Failed to activate upgrade algorithm", "err", err)
			}
		}
	}
}

// Handle parses one codeUploaded log, queries getInfo and delegates activation.
func (s *Service) Handle(ctx context.Context, client Client, eventLog types.Log) error {
	if s == nil || s.activator == nil {
		return errors.New("event service activator is not configured")
	}
	var name string
	if err := s.contractABI.UnpackIntoInterface(&name, "codeUploaded", eventLog.Data); err != nil {
		return fmt.Errorf("decode codeUploaded event: %w", err)
	}
	name = model.NormalizeAlgorithmName(name)
	if name == "" {
		return errors.New("decode codeUploaded event: empty algorithm name")
	}
	info, err := s.GetInfo(ctx, client, name)
	if err != nil {
		return err
	}
	if err := s.activator.Activate(ctx, name, info); err != nil {
		return fmt.Errorf("activate algorithm %s: %w", name, err)
	}
	log.Info("Activated upgrade algorithm", "name", name)
	return nil
}

// GetInfo queries the authoritative CodeStorage metadata for an algorithm.
func (s *Service) GetInfo(ctx context.Context, client Client, name string) (model.AlgorithmInfo, error) {
	input, err := s.contractABI.Pack("getInfo", name)
	if err != nil {
		return model.AlgorithmInfo{}, fmt.Errorf("pack CodeStorage getInfo input: %w", err)
	}
	msg := ethereum.CallMsg{To: &s.address, Data: input}
	output, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return model.AlgorithmInfo{}, fmt.Errorf("call CodeStorage getInfo: %w", err)
	}
	values, err := s.contractABI.Unpack("getInfo", output)
	if err != nil {
		return model.AlgorithmInfo{}, fmt.Errorf("unpack CodeStorage getInfo output: %w", err)
	}
	if len(values) != 4 {
		return model.AlgorithmInfo{}, fmt.Errorf("CodeStorage getInfo returned %d values", len(values))
	}
	code, codeOK := values[0].(string)
	gas, gasOK := values[1].(uint64)
	inputTypes, inputOK := values[2].(string)
	outputTypes, outputOK := values[3].(string)
	if !codeOK || !gasOK || !inputOK || !outputOK {
		return model.AlgorithmInfo{}, errors.New("CodeStorage getInfo returned unexpected ABI types")
	}
	return model.AlgorithmInfo{Code: code, Gas: gas, IType: inputTypes, OType: outputTypes}, nil
}
