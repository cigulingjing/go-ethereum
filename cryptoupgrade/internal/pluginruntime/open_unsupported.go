//go:build !linux && !darwin && !freebsd

package pluginruntime

import "fmt"

func openPlugin(string) (Module, error) {
	return nil, fmt.Errorf("go plugins are not supported on this platform")
}

// Default 是进程级共享 plugin loader。
var Default = NewLoader(openPlugin)
