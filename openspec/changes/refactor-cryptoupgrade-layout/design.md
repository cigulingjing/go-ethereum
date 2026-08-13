## Context

`cryptoupgrade/` 现在包含三类差异很大的内容：

- 被 geth 主流程直接导入的运行时根包，例如 `plugin.go`、`code_storage.go`、`path.go`、`precompile.go`。
- 研究资产和实验资产，例如 `algorithm/go`、`algorithm/contracts`、`preload`、`network`、`examples`。
- 可执行实验入口、文档和结果，例如 `cmd/bench*`、`cmd/multinode`、`cmd/upgradeflow`、`docs`、`results`。

`core/vm`、`node` 和 `cmd/geth` 已经直接导入 `github.com/ethereum/go-ethereum/cryptoupgrade`。因此目录整理必须保留根包作为稳定集成入口，不能为了归类整洁而让 geth 侧大范围改 import。已有 `cryptoupgrade-plugin-directory-management` 规范还定义了节点运行时 plugin 制品目录；本 change 只整理仓库源码布局，不改变该运行时目录语义。

## Goals / Non-Goals

**Goals:**

- 建立 `cryptoupgrade/` 源码布局规范，明确 runtime、算法资产、实验工具、网络环境、文档和结果归档的边界。
- 保留 `github.com/ethereum/go-ethereum/cryptoupgrade` 根包作为 EVM、node shutdown 和 geth event binding 的稳定入口。
- 将可执行实验命令按实验目的分组，降低旧 `cryptoupgrade/cmd` 下命令随实验增长而失控的风险。
- 将实验配置、实验中间资产和结果归档分开，保证实验可复现且不会污染运行时路径。
- 更新默认路径、文档和测试，使目录迁移后现有实验仍能用默认参数运行。

**Non-Goals:**

- 不改变 CodeStorage ABI、EVM 调用语义、gas 计费、precompile 地址或算法注册语义。
- 不修改运行时 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`、默认 `./plugin`、`src`、`so` 和 `algorithm_info.json` 解析规则。
- 不重新设计算法接口命名；该部分由 `refactor-algorithm-interface` 和对应规范约束。
- 不删除历史实验结果；需要迁移时保留原始数据并记录路径变化。
- 不引入新的构建系统或外部依赖。

## Decisions

1. 保留 `cryptoupgrade` 根包作为运行时 API 层。

   根目录继续放置被 geth 主流程直接调用的文件，包括 CodeStorage 调度、ABI 编解码、动态 plugin 调用、预编译注册和路径解析。备选方案是迁移到 `cryptoupgrade/runtime` 并更新所有导入；该方案会让 `core/vm`、`node`、`cmd/geth` 产生不必要的侵入式修改，且对论文实验能力没有直接收益。

2. 使用目录职责而不是实验编号组织源码。

   目标布局按职责划分：

   | 目录 | 职责 |
   | --- | --- |
   | `cryptoupgrade/` | 运行时根包与 geth 集成入口 |
   | `cryptoupgrade/algorithm/` | Go plugin 候选与 Solidity 对照算法 |
   | `cryptoupgrade/preload/` | 预置算法实现和预加载注册 |
   | `cryptoupgrade/bench/` | benchmark、升级流程和实验命令入口 |
   | `cryptoupgrade/network/` | 多节点实验网络配置解析、渲染和验证库 |
   | `cryptoupgrade/examples/` | 可复现实验配置和静态输入样例 |
   | `cryptoupgrade/docs/` | 实验说明、操作文档和结果解读 |
   | `cryptoupgrade/results/` | 已产生的实验输出和原始数据 |
   | `cryptoupgrade/docker/` | cryptoupgrade 实验用容器资产 |

   备选方案是按 `lab1`、`lab2` 组织所有代码。该方案便于论文章节对应，但会复制通用网络和 benchmark 逻辑，后续跨实验复用成本更高。

3. 将 `cryptoupgrade/cmd/*` 迁移到 `cryptoupgrade/bench/cmd/*`。

   `benchcall`、`benchblake2b`、`benchcandidate`、`benchrealchain`、`benchupgradeefficiency`、`benchexecutionefficiency`、`benchupgradelatency`、`upgradeflow` 都属于研究实验入口；`multinode` 是实验网络辅助命令，也应和实验 tooling 放在同一层入口下。保留命令包名为 `main`，仅更新路径和文档。备选方案是继续堆叠在 `cryptoupgrade/cmd`，但它不能表达这些命令和运行时根包的边界。

4. 保持算法资产现有二级结构。

   `cryptoupgrade/algorithm/go`、`cryptoupgrade/algorithm/go/archive` 和 `cryptoupgrade/algorithm/contracts` 已经被算法接口规范约束，本 change 不再重命名这些路径。`algorithm/check_interfaces.sh` 继续作为算法一致性检查入口，可以在迁移后更新引用。

5. 保留 `results` 中的历史结果并允许新旧路径共存过渡。

   历史 JSON、CSV、图片和日志是论文实验依据，不应为了目录重整删除。迁移实现可以移动到 `cryptoupgrade/results/<experiment>/<run-id>` 的稳定结构，也可以保留旧 run 目录并只更新新实验默认输出路径；无论采用哪种方式，都必须在文档中记录。

6. 用静态路径检查防止布局回退。

   实现阶段应增加或复用一个轻量检查入口，验证不再新增 `cryptoupgrade/cmd/*` 实验命令、根目录不出现新的实验结果文件、默认路径仍指向规范目录。备选方案是只依赖 review；该方案容易在后续新增实验时回退到旧目录习惯。

## Risks / Trade-offs

- [Risk] `go test ./...` 或用户脚本仍引用旧 `cryptoupgrade/cmd/*` 路径。→ 迁移时使用 `rg` 全量更新仓库内引用，并在文档中给出新命令路径。
- [Risk] 历史结果中的绝对路径无法自动改写。→ 历史结果保留原始记录，新结果写入新默认目录；文档说明旧结果路径是历史运行环境快照。
- [Risk] 将 `multinode` 放入 benchmark 工具层可能弱化它作为网络辅助命令的语义。→ 在 `bench/cmd/multinode` 文档中明确其职责是实验网络渲染、初始化和验证，不是 geth 节点入口。
- [Risk] 根包继续包含多个文件，看起来没有彻底分层。→ 该权衡换取对 geth 集成点的最小侵入；根包只保留运行时 API 和集成边界，不放实验命令或结果资产。

## Migration Plan

1. 迁移实验命令目录：将 `cryptoupgrade/cmd/*` 移动到 `cryptoupgrade/bench/cmd/*`，保持各命令 flag 和输出 schema 不变。
2. 更新代码、测试、OpenSpec、文档和脚本中的命令路径引用。
3. 更新 benchmark 默认路径：算法输入继续指向 `cryptoupgrade/algorithm/*`，网络配置继续指向 `cryptoupgrade/examples/*`，结果默认写入 `cryptoupgrade/results/<experiment>`。
4. 补充路径布局检查，覆盖命令目录、算法目录和结果目录的基本规则。
5. 运行相关命令包测试、`go test ./cryptoupgrade/...` 和 `openspec validate refactor-cryptoupgrade-layout`。
6. 如果迁移后发现外部脚本依赖旧路径，优先更新项目内文档；除非明确需要兼容外部调用，不新增重复 wrapper 命令。
