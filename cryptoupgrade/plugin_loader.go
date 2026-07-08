//go:build linux || darwin || freebsd

package cryptoupgrade

import (
	"fmt"
	"plugin"
)

func lookupPluginFunction(pluginPath string, funName string) (interface{}, error) {
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("open plugin %s: %w", pluginPath, err)
	}
	fn, err := p.Lookup(funName)
	if err != nil {
		return nil, fmt.Errorf("find symbol %s in plugin %s: %w", funName, pluginPath, err)
	}
	return fn, nil
}
