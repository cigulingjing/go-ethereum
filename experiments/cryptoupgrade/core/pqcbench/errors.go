package pqcbench

import "errors"

// 可识别错误：实验入口据此跳过，而不是把另一条路径的结果冒充过来。
var (
	ErrCGODisabled      = errors.New("cgo-disabled")
	ErrMissingArtifacts = errors.New("missing-artifacts")
	ErrOutputMismatch   = errors.New("output-mismatch")
)
