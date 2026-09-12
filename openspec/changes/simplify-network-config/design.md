## Context
技术栈为 Go、yaml.v3、Docker Compose。公共 LoadConfig 被部署与 benchmark 共享，现有工作区已有修改，实施时保留这些改动。
## Goals / Non-Goals
集中手写配置和运行目录；兼容历史展开快照。不修改共识或清理运行数据。
## Decisions
使用可选 local 配置（nodeCount、outputDir、端口基址、signerAccount、extraArgs）在内存中展开节点。与 nodes 互斥，避免两套真相。加载不写文件；可复现测试 P2P key 在内存派生，只在 render 写入节点目录。账户 keystore/password 放 deployments/fixtures。render 快照保存实际输出路径及展开节点，独立于后续源配置修改。保留 genlocalnetwork 兼容旧自动化，迁移其资源默认位置。
## Risks / Trade-offs
公开确定性密钥仅用于本地实验。更改数量不会删除旧 datadir；用户先停止网络再 render 并以 --remove-orphans 启动。旧路径消费者需迁移到 local.yaml，历史结果不改写。
## Migration Plan
加入 compact reader，迁移配置和资源，更新默认入口及文档，运行网络和命令测试。回滚可使用历史展开 YAML。
