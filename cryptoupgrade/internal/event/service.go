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
	ActivateVersion(ctx context.Context, name string, info model.AlgorithmVersionInfo) error
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
		Topics:    [][]common.Hash{{s.topic, versionedTopic(s.contractABI)}},
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

// Handle parses one CodeStorage upgrade log, queries chain metadata and delegates activation.
func (s *Service) Handle(ctx context.Context, client Client, eventLog types.Log) error {
	if s == nil || s.activator == nil {
		return errors.New("event service activator is not configured")
	}
	if len(eventLog.Topics) > 0 && eventLog.Topics[0] == versionedTopic(s.contractABI) {
		return s.handleVersioned(ctx, client, eventLog)
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

func (s *Service) handleVersioned(ctx context.Context, client Client, eventLog types.Log) error {
	values, err := s.contractABI.Unpack("codeVersionUploaded", eventLog.Data)
	if err != nil {
		return fmt.Errorf("decode codeVersionUploaded event: %w", err)
	}
	if len(values) != 3 {
		return fmt.Errorf("codeVersionUploaded returned %d values", len(values))
	}
	name, ok := values[0].(string)
	if !ok {
		return fmt.Errorf("codeVersionUploaded name has type %T, want string", values[0])
	}
	version, ok := values[1].(uint64)
	if !ok {
		return fmt.Errorf("codeVersionUploaded version has type %T, want uint64", values[1])
	}
	activationBlock, ok := values[2].(uint64)
	if !ok {
		return fmt.Errorf("codeVersionUploaded activationBlock has type %T, want uint64", values[2])
	}
	name = model.NormalizeAlgorithmName(name)
	if name == "" {
		return errors.New("decode codeVersionUploaded event: empty algorithm name")
	}
	info, err := s.GetVersionInfo(ctx, client, name, version)
	if err != nil {
		return err
	}
	if info.ActivationBlock != activationBlock {
		return fmt.Errorf("CodeStorage version activation block mismatch for %s v%d: event=%d info=%d", name, version, activationBlock, info.ActivationBlock)
	}
	if err := s.activator.ActivateVersion(ctx, name, info); err != nil {
		return fmt.Errorf("activate algorithm %s version %d at block %d: %w", name, version, activationBlock, err)
	}
	log.Info("Activated upgrade algorithm version", "name", name, "version", version, "activationBlock", activationBlock)
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

// GetVersionInfo queries the authoritative CodeStorage metadata for one algorithm version.
func (s *Service) GetVersionInfo(ctx context.Context, client Client, name string, version uint64) (model.AlgorithmVersionInfo, error) {
	input, err := s.contractABI.Pack("getVersionInfo", name, version)
	if err != nil {
		return model.AlgorithmVersionInfo{}, fmt.Errorf("pack CodeStorage getVersionInfo input: %w", err)
	}
	msg := ethereum.CallMsg{To: &s.address, Data: input}
	output, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return model.AlgorithmVersionInfo{}, fmt.Errorf("call CodeStorage getVersionInfo: %w", err)
	}
	values, err := s.contractABI.Unpack("getVersionInfo", output)
	if err != nil {
		return model.AlgorithmVersionInfo{}, fmt.Errorf("unpack CodeStorage getVersionInfo output: %w", err)
	}
	if len(values) != 6 {
		return model.AlgorithmVersionInfo{}, fmt.Errorf("CodeStorage getVersionInfo returned %d values", len(values))
	}
	code, codeOK := values[0].(string)
	gas, gasOK := values[1].(uint64)
	inputTypes, inputOK := values[2].(string)
	outputTypes, outputOK := values[3].(string)
	storedVersion, versionOK := values[4].(uint64)
	activationBlock, activationOK := values[5].(uint64)
	if !codeOK || !gasOK || !inputOK || !outputOK || !versionOK || !activationOK {
		return model.AlgorithmVersionInfo{}, errors.New("CodeStorage getVersionInfo returned unexpected ABI types")
	}
	return model.AlgorithmVersionInfo{
		AlgorithmInfo:   model.AlgorithmInfo{Code: code, Gas: gas, IType: inputTypes, OType: outputTypes},
		Version:         storedVersion,
		ActivationBlock: activationBlock,
	}, nil
}

func versionedTopic(contractABI abi.ABI) common.Hash {
	if event, ok := contractABI.Events["codeVersionUploaded"]; ok {
		return event.ID
	}
	return common.Hash{}
}
