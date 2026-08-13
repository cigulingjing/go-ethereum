# Cryptoupgrade experiments

本目录保存动态密码升级研究的实验入口、网络部署输入、实验文档和新生成结果：

- `bench/cmd`：benchmark、多节点部署与升级流程命令；
- `network`：网络配置、渲染、RPC、chain ID、peer、出块和 Clique 状态验证；
- `smoke`：源码上传、receipt 等待、异步激活等待和 `callFunc` 结果校验；
- `deployments/docker`：实验镜像和 Docker 检查脚本；
- `deployments/networks`：可复现的网络 YAML、node key 和测试账户输入；
- `docs`：当前实验操作及结果说明；
- `results`：新实验结果的默认根目录。

实验代码只通过 `github.com/ethereum/go-ethereum/cryptoupgrade` facade 使用运行时能力。
历史 `cryptoupgrade/results` 是只读运行快照，仍保留原路径且不应改写。
