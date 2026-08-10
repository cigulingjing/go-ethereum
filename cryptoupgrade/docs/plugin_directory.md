# Cryptoupgrade 插件目录

Cryptoupgrade 将运行时插件产物存放在节点本地的插件目录中。

默认情况下，该目录相对于 geth 进程启动时的工作目录解析：

```text
./plugin/
  algorithm_info.json
  src/
    <Algorithm>.go
  so/
    <Algorithm>.so
```

默认配置保留现有开发目录布局。运行时在包初始化阶段解析一次基础目录，随后所有子路径均基于该已解析的基础目录派生。

## 覆盖默认路径

在启动 geth 之前设置 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`，可将产物放到指定位置：

```sh
GETH_CRYPTOUPGRADE_PLUGIN_DIR=/var/lib/geth-node-a/cryptoupgrade-plugin ./build/bin/geth ...
```

设置后，cryptoupgrade 使用：

```text
/var/lib/geth-node-a/cryptoupgrade-plugin/
  algorithm_info.json
  src/
  so/
```

相对路径的覆盖值会相对于 geth 启动时的工作目录解析。以服务方式部署时，建议使用绝对路径。

## 运维说明

- 将插件目录视为节点本地的运行时状态，不要在多个并发运行的节点之间共享同一插件目录。
- 在进程启动前设置 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`；geth 运行后再修改该环境变量不会迁移当前生效的路径。
- 运行时不会自动将 `./plugin` 中的文件复制或迁移到覆盖目录。如需复用，请手动复制 `algorithm_info.json` 及所需产物。
- 目录创建错误会通过算法激活或元数据存储路径返回，便于尽早暴露权限与路径冲突问题。
