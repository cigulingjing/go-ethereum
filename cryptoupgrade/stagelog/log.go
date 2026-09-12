// Package stagelog 记录节点本地时间边界，不参与执行结果或共识判断。
package stagelog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/sirupsen/logrus"
)

const (
	FileEnv    = "GETH_CRYPTOUPGRADE_STAGE_LOG"
	DisableEnv = "GETH_CRYPTOUPGRADE_STAGE_LOG_DISABLE"
	NodeEnv    = "GETH_CRYPTOUPGRADE_NODE_ID"
)

type Fields = logrus.Fields
type contextKey struct{}

var (
	sequence    atomic.Uint64
	hostname, _ = os.Hostname()
	sessionID   = fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())
	output      struct {
		sync.Mutex
		path      string
		file      *os.File
		logger    *logrus.Logger
		lastError string
	}
)

func Path() string {
	if disabled, _ := strconv.ParseBool(os.Getenv(DisableEnv)); disabled {
		return ""
	}
	if path := strings.TrimSpace(os.Getenv(FileEnv)); path != "" {
		return path
	}
	if dir := strings.TrimSpace(os.Getenv("GETH_CRYPTOUPGRADE_PLUGIN_DIR")); dir != "" {
		return filepath.Join(dir, "stage_timing.jsonl")
	}
	return ""
}

func Enabled() bool { return Path() != "" }
func NewID() string { return fmt.Sprintf("%s-%d", sessionID, sequence.Add(1)) }

// With 复制元数据，避免批量和并发请求修改共享 map。
func With(ctx context.Context, fields Fields) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	merged := Metadata(ctx)
	for k, v := range fields {
		merged[k] = v
	}
	return context.WithValue(ctx, contextKey{}, merged)
}

func Metadata(ctx context.Context) Fields {
	result := Fields{}
	if ctx != nil {
		if fields, ok := ctx.Value(contextKey{}).(Fields); ok {
			for k, v := range fields {
				result[k] = v
			}
		}
	}
	return result
}

func Record(ctx context.Context, stage string, fields Fields) {
	RecordAt(ctx, stage, time.Now(), fields)
}

// RecordAt 在锁和文件 IO 之前保存时间；记录失败不能改变交易或升级行为。
func RecordAt(ctx context.Context, stage string, at time.Time, fields Fields) {
	path := Path()
	if path == "" {
		return
	}
	data := Metadata(ctx)
	for k, v := range fields {
		data[k] = v
	}
	data["stage"] = stage
	data["timestampUnixNano"] = strconv.FormatInt(at.UnixNano(), 10)
	data["sessionId"] = sessionID
	data["pid"] = os.Getpid()
	node := os.Getenv(NodeEnv)
	if node == "" {
		node = hostname
	}
	data["nodeId"] = node
	output.Lock()
	defer output.Unlock()
	if output.path != path || output.file == nil {
		if output.file != nil {
			_ = output.file.Close()
			output.file = nil
		}
		output.path = path
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err == nil {
			output.file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		}
		if err != nil {
			if message := err.Error(); message != output.lastError {
				log.Warn("Cannot open cryptoupgrade stage log", "path", path, "err", err)
				output.lastError = message
			}
			return
		}
		output.lastError = ""
		output.logger = logrus.New()
		output.logger.SetOutput(output.file)
		output.logger.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	}
	output.logger.WithFields(data).WithTime(at.UTC()).Info(stage)
}

// Close 供显式关闭或测试切换日志使用；无用户态缓冲需要额外刷新。
func Close() {
	output.Lock()
	defer output.Unlock()
	if output.file != nil {
		_ = output.file.Close()
		output.file = nil
	}
}

// ReceiveRPC 在解码后的请求进入 handler 时记录，包括批量请求的排队时间。
func ReceiveRPC(method, rpcID string) Fields {
	return ReceiveRPCAt(method, rpcID, time.Now())
}

func ReceiveRPCAt(method, rpcID string, at time.Time) Fields {
	if !Enabled() {
		return nil
	}
	switch method {
	case "eth_call", "eth_sendTransaction", "eth_sendRawTransaction", "eth_sendRawTransactionSync":
	default:
		return nil
	}
	fields := Fields{"requestId": NewID(), "rpcId": rpcID, "rpcMethod": method, "phase": "rpc"}
	RecordAt(nil, "rpc_received", at, fields)
	return fields
}
