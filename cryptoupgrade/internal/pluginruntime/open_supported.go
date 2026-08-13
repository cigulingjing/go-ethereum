//go:build linux || darwin || freebsd

package pluginruntime

import "plugin"

type pluginModule struct {
	plugin *plugin.Plugin
}

func (m pluginModule) Lookup(name string) (any, error) {
	return m.plugin.Lookup(name)
}

func openPlugin(path string) (Module, error) {
	loaded, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	return pluginModule{plugin: loaded}, nil
}

// Default 是进程级共享 plugin loader。
var Default = NewLoader(openPlugin)
