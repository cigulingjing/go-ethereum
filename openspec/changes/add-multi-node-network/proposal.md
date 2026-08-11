## Why

当前动态密码升级功能和 `lab2-execution-efficiency` 实验主要依赖本地 `geth --dev` 单节点环境，能够验证基础调用链路，但不能证明该机制在多节点私有链中的配置、部署、共识和升级传播行为。论文后续实验需要一个可复现的多机架构，用于在服务器上拉起多个 Geth 节点并形成私有链客户端网络。

本 change 解决从单机测试环境迁移到多节点私有链实验环境的问题，为后续多节点升级一致性、部署成本和网络化执行效率实验提供基础。

## What Changes

- 新增 YAML 配置载入能力，用于描述节点 ID、节点名称、IP/host、P2P/RPC 端口、数据目录、插件目录、账户/签名者和启动参数。
- 新增多节点私有链初始化流程，能够根据配置生成或校验 genesis、静态 peer、节点数据目录和启动命令。
- 新增 Docker 镜像打包与容器启动支持，能够在服务器上部署多个节点并组成可互联的 Geth 私有链网络。
- 新增建议的 Clique 共识私有链配置，使授权 signer 节点能够为私有链提供区块生产和共识认证。
- 保持现有动态密码升级、precompile、Solidity 对照实验的执行语义不变；多节点配置只扩展部署和运行环境。
- 提供基础验证入口，用于确认节点互联、Clique 出块、RPC 可访问和动态密码升级功能可在多节点网络中启动。

## Capabilities

### New Capabilities

- `multi-node-private-network`: 定义基于配置文件的多节点 Geth 私有链部署、Docker 打包、节点组网和 Clique 共识要求。

### Modified Capabilities

- None.

## Impact

- Affected code: 新增或扩展多节点配置解析、网络初始化脚本、Dockerfile/docker compose 配置、Clique genesis 生成或校验工具，以及必要的启动命令入口。
- Affected inputs: YAML 网络配置、节点账户/keystore、genesis 配置、静态 peer 列表、Docker build/run 参数。
- Affected systems: 本地开发环境、服务器 Docker 环境、多节点私有链运行目录和后续论文实验脚本。
- No breaking changes to Geth consensus defaults, EVM execution semantics, dynamic crypto upgrade activation, existing single-node experiments, Solidity candidate contracts, or native precompile behavior.
