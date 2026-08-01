# Cryptoupgrade Plugin Directory

Cryptoupgrade stores runtime plugin artifacts in a node-local plugin directory.

By default, the directory is resolved from the geth process startup working directory:

```text
./plugin/
  algorithm_info.json
  src/
    <Algorithm>.go
  so/
    <Algorithm>.so
```

The default preserves the existing development layout. The runtime resolves the base directory once during package initialization, then derives all child paths from that resolved base.

## Override

Set `GETH_CRYPTOUPGRADE_PLUGIN_DIR` before starting geth to place artifacts somewhere explicit:

```sh
GETH_CRYPTOUPGRADE_PLUGIN_DIR=/var/lib/geth-node-a/cryptoupgrade-plugin ./build/bin/geth ...
```

With this setting, cryptoupgrade uses:

```text
/var/lib/geth-node-a/cryptoupgrade-plugin/
  algorithm_info.json
  src/
  so/
```

Relative override values are resolved against the geth startup working directory. Prefer absolute paths for service deployments.

## Operational Notes

- Treat the plugin directory as node-local runtime state. Do not share one plugin directory between concurrently running nodes.
- Set `GETH_CRYPTOUPGRADE_PLUGIN_DIR` before process startup; changing it after geth is running does not move the active paths.
- The runtime does not automatically copy or migrate files from `./plugin` to an override directory. Copy `algorithm_info.json` and any required artifacts manually if reuse is needed.
- Directory creation errors are returned through algorithm activation or metadata storage paths, which helps surface permission and path-conflict problems early.
