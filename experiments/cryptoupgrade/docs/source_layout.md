# cryptoupgrade 源码目录布局

`cryptoupgrade` 目录按职责分层：

```text
cryptoupgrade/
├── *.go                       # geth 直接导入的运行时根包
├── algorithm/                 # Go plugin 候选和 Solidity 对照算法
├── builtin/                   # 编译进节点并由 registry 直接调用的算法
├── internal/                  # codec、编译、加载、持久化和激活内部实现
└── results/                   # 原地只读保留的历史实验结果

experiments/cryptoupgrade/
├── bench/cmd/                 # benchmark、升级流程和实验网络命令
├── network/                   # 网络配置、渲染和网络健康验证
├── smoke/                     # 上传、异步激活等待和调用校验
├── deployments/              # Docker 和网络 YAML 部署输入
├── docs/                      # 操作文档和结果解读
└── results/                   # 新实验结果默认输出
```

运行时 plugin 制品目录不属于源码布局迁移范围。节点仍通过 `GETH_CRYPTOUPGRADE_PLUGIN_DIR` 或默认 `./plugin` 解析 `src`、`so` 和 `algorithm_info.json`。

## 检查命令

```shell
bash cryptoupgrade/check_layout.sh
```

该检查会确认：

- 实验命令位于 `experiments/cryptoupgrade/bench/cmd`；
- 算法资产位于 `cryptoupgrade/algorithm/go`、`cryptoupgrade/algorithm/go/archive` 和 `cryptoupgrade/algorithm/contracts`；
- builtin 实现位于 `cryptoupgrade/builtin`，且不依赖动态候选源码目录；
- 旧的 `cryptoupgrade/cmd` 目录不存在；
- 运行时不反向导入 `experiments/cryptoupgrade`；
- 文档和脚本不再把旧实验路径作为当前入口；
- 新实验结果写入 `experiments/cryptoupgrade/results`，历史 `cryptoupgrade/results` 保持只读。
