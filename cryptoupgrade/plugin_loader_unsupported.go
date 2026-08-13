//go:build !linux && !darwin && !freebsd

package cryptoupgrade

import "github.com/ethereum/go-ethereum/cryptoupgrade/internal/pluginruntime"

func lookupPluginFunction(pluginPath, funName string) (interface{}, error) {
	return pluginruntime.Default.LookupActive(pluginPath, funName)
}
