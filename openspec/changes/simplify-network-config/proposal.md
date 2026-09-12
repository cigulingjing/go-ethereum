## Why
每种节点规模维护逐节点 YAML 导致配置重复、阅读成本高，测试目录分散。

## What Changes
- 提供单一简洁配置，以 nodeCount 控制节点规模并集中运行目录。
- 自动展开节点、端口和测试 nodekey，保留旧式节点列表读取。
- **BREAKING**: 删除 deployments/networks 中旧配置，统一使用 local.yaml；测试密钥移动到 fixtures。

## Capabilities
### New Capabilities
- `compact-network-config`: 简洁节点规模配置与集中部署输出。
### Modified Capabilities
无。

## Impact
实验 network 包、命令默认路径、部署资源和说明文档。

## 非目标
不修改 Geth 共识与密码算法，不清理历史实验结果，不启动实验容器。
