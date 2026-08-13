package cryptoupgrade

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	eventadapter "github.com/ethereum/go-ethereum/cryptoupgrade/internal/event"
)

// client 避免 ethclient 通过 core 反向导入 cryptoupgrade。
type client interface {
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

func runtimeEventService() *eventadapter.Service {
	return eventadapter.NewService(CodeStorageABI, common.CodeStorageAddress, codeUploaded, runtimeActivationService())
}

// BindCodeUploaded subscribes to asynchronous local activation events.
func BindCodeUploaded(client client) {
	runtimeEventService().Bind(context.Background(), client)
}

func handleCodeUploadedEvent(client client, eventLog types.Log) {
	_ = runtimeEventService().Handle(context.Background(), client, eventLog)
}
