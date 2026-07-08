//go:build !linux && !darwin && !freebsd

package cryptoupgrade

import "fmt"

func lookupPluginFunction(_, _ string) (interface{}, error) {
	return nil, fmt.Errorf("go plugins are not supported on this platform")
}
