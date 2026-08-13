## Context

当前 `cryptoupgrade` 根 package 被 `core/vm`、`node` 和 `cmd/geth` 直接导入，根目录中的文件却同时承担 CodeStorage 调度、event 订阅、ABI 编解码、源码解压、Go plugin 编译、动态加载、路径解析、元数据持久化和 native precompile 实现。`cryptoupgrade/network`、`bench`、`docker`、`examples`、`docs`、`results` 则属于论文实验与部署层，但和运行时代码共享同一顶层目录。

现有 `refactor-cryptoupgrade-layout` 已完成算法资产和实验命令的初步整理，本 change 在其基础上进一步约束 package 职责和依赖方向。迁移必须保持根 package API、CodeStorage ABI、gas、plugin 目录及实验数据兼容。

技术栈保持为 Go、ethereum/go-ethereum、Go Plugin、Docker、Docker Compose 和 YAML，不增加外部依赖。

## Goals / Non-Goals

**Goals:**

- 保留 `github.com/ethereum/go-ethereum/cryptoupgrade` 作为 Geth 唯一稳定集成入口。
- 将 EVM/event 适配、升级编排、源码制品编解码、编译、plugin 运行时及持久化拆分为单一职责 package。
- 明确动态候选算法与编译进节点的 builtin 算法边界。
- 将 benchmark、network、Docker、YAML、实验文档和结果移入仓库级实验目录。
- 建立单向依赖和静态布局检查，防止实验部署逻辑回流运行时。
- 分阶段迁移并保持现有调用语义和实验可复现性。

**Non-Goals:**

- 不改变 CodeStorage ABI、event topic、升级完成判据、gas、precompile 地址或共识行为。
- 不重新设计算法函数签名、版本协议或 `algorithm_info.json` schema。
- 不合并动态 plugin 算法与 native precompile 的实现和计费模型。
- 不删除或改写历史实验数据。
- 不把实验网络扩展为通用 Geth 部署框架。

## Decisions

### 1. 根 package 作为 facade，具体实现进入 internal packages

目标布局为：

```text
cryptoupgrade/
├── doc.go
├── export.go
├── internal/
│   ├── model/           # 跨模块领域类型，不包含流程
│   ├── evm/             # CodeStorage、ABI 和 native precompile 适配
│   ├── event/           # codeUploaded 订阅与解析
│   ├── activation/      # 一次升级的应用层编排
│   ├── sourcecodec/     # Go 源码 gzip/base64 编解码与校验
│   ├── callcodec/       # callFunc 输入输出 ABI 编解码
│   ├── compiler/        # Go 源码到 plugin 制品
│   ├── pluginruntime/   # plugin 加载、缓存、符号解析与调用
│   └── repository/      # workspace 路径和算法元数据持久化
├── algorithm/
│   ├── go/
│   ├── go/archive/
│   └── contracts/
└── builtin/
    ├── bls12381/
    ├── vote/
    └── ...
```

`core/vm`、`node` 和 `cmd/geth` 继续只导入根 package。`export.go` 使用薄转发或 type alias 暴露现有 API，内部 package 不构成项目公共 API。相比直接暴露 `cryptoupgrade/runtime/*`，`internal` 能在编译期阻止实验代码绕过 facade 依赖具体实现。

### 2. event 适配与升级编排分离

`internal/event` 只负责订阅、解析 `codeUploaded` 并取得链上信息，随后调用 `activation.Service`。`activation.Service` 按以下顺序编排升级：

```text
event → sourcecodec.Decode → compiler.Compile
      → repository.Save → pluginruntime.Load
```

运行时调用不通过 event package。`uploadCode` 仍不得在 EVM 交易路径同步编译，交易 receipt 也不代表本地激活完成。

备选方案是让 `bind_event.go` 继续直接调用压缩、编译和路径函数；该方案文件较少，但无法隔离协议适配和升级生命周期，也不利于独立测试失败恢复。

### 3. 区分源码制品编码与调用 ABI 编码

现有 `compressed.go` 对应源码传输制品，迁入 `sourcecodec`；现有 `serialize.go` 实际处理 `callFunc` ABI，迁入 `callcodec`。源码文件压缩 API 由 `sourcecodec` 统一提供，benchmark 和 smoke test 不再维护私有 `compressFile` 副本。

不使用笼统的 `serialize` package，因为 gzip/base64 源码打包与 EVM ABI 编解码具有不同的格式兼容和安全边界。

### 4. 编译器不拥有持久化路径和元数据

`compiler` 接收明确的源码路径、输出路径和构建上下文，只负责执行 `go build -buildmode=plugin`、校验构建结果并原子替换输出。`repository` 负责解析 `GETH_CRYPTOUPGRADE_PLUGIN_DIR`、管理 `src/`、`so/` 和 `algorithm_info.json`。

该边界允许独立测试编译命令，同时防止编译器逐步吸收版本、缓存和元数据职责。默认 plugin 路径及已有环境变量语义保持不变。

### 5. 动态候选算法和 builtin 算法分开

`algorithm/` 保留为上传实验使用的源码资产，不作为 Geth 运行时 package 依赖。现有 `preload/` 改名为 `builtin/`，表达这些实现编译进节点、由 registry 直接调用，而不是运行时临时预加载。

`precompile.go` 的注册与 gas/ABI 适配归 `internal/evm`；可复用的算法计算实现可逐步迁入 `builtin`，但本 change 不强制合并与 plugin 候选重复的算法代码。`vote`/`vote2` 和 `elgmal` 的语义重命名单独记录，只有在调用关系和实验对应关系明确后执行。

### 6. 实验与部署资产移出 cryptoupgrade 运行时目录

仓库级目标布局为：

```text
experiments/cryptoupgrade/
├── bench/cmd/
├── network/                 # 配置、genesis、compose、peer 和网络健康验证
├── smoke/                   # 上传、激活和调用算法的端到端验证
├── deployments/
│   ├── docker/
│   └── networks/
├── docs/
└── results/
```

依赖方向固定为：

```text
Geth → cryptoupgrade facade → cryptoupgrade/internal
experiments/cryptoupgrade → cryptoupgrade facade
cryptoupgrade →/ experiments/cryptoupgrade
```

这里的 `→/` 表示禁止依赖。实验 Go 代码仍属于当前 Go module，无需建立第二个 module。

备选方案是使用 `cryptoupgrade/experiments`，迁移较小但仍会把部署资产表现为运行时功能的一部分；使用仓库级目录更符合系统部署层定位。

### 7. 网络健康验证和 cryptoupgrade smoke test 分开

`network` 只验证配置、RPC、chain ID、peer、出块和容器状态。Add 源码上传、激活等待及 `callFunc` 校验迁入 `smoke`，由实验命令按需组合两类验证。

这样网络库不再依赖 `cryptoupgrade.CodeStorageABI`，也不会硬编码 `cryptoupgrade/algorithm/go/add.go`。网络验证结果和 smoke test 结果仍可合并到同一次实验输出中。

### 8. 采用分阶段兼容迁移

先新增内部 package 和根 facade，再逐个迁移实现；确认调用方只依赖 facade 后，才删除根目录旧实现。实验目录迁移时同步更新仓库内命令、fixture 和文档，不长期保留重复命令 wrapper。

历史 `cryptoupgrade/results` 保持只读；迁移时原样移动或保留位置映射，新运行默认写入 `experiments/cryptoupgrade/results/<experiment>/<run-id>`。

## Risks / Trade-offs

- [Risk] 拆分 Go package 后出现 import cycle。→ 共享数据结构下沉到 `internal/model`，编排层只向基础能力单向依赖，根 facade 负责组装。
- [Risk] 根 facade 转发遗漏导致 Geth 行为变化。→ 迁移前建立导出 API 清单，并对 `core/vm`、`node`、`cmd/geth` 运行定向测试。
- [Risk] Go Plugin 对 package path 和构建上下文敏感。→ 不改变动态候选源码接口和 module path，使用现有构建上下文执行回归测试。
- [Risk] 移动实验入口破坏脚本和论文命令。→ 全量更新仓库内引用，记录旧新路径映射，并运行主要实验命令的帮助和 smoke test。
- [Risk] 历史结果包含旧绝对路径。→ 不改写结果内容，文档说明其为历史运行快照。
- [Trade-off] `internal` 增加 package 数量。→ 以可测试边界和编译期依赖约束换取少量组装代码。

## Migration Plan

1. 记录现有根 package 导出 API、内部调用和测试基线。
2. 新增 `model`、`sourcecodec`、`callcodec`、`repository`、`compiler` 和 `pluginruntime`，迁移实现并保留根 package 转发。
3. 引入 `activation.Service`，再迁移 event 和 EVM 适配；验证异步激活语义不变。
4. 将 `preload` 迁为 `builtin` 并更新内部 import，暂不处理含义不明确的算法重命名。
5. 创建 `experiments/cryptoupgrade`，迁移 benchmark、network、Docker、YAML、docs 和 results。
6. 将 cryptoupgrade smoke 从 network 验证拆出，统一源码压缩调用。
7. 更新代码、脚本、OpenSpec 和文档引用，增加布局及禁止反向依赖检查。
8. 运行格式化、单元测试、`go test ./cryptoupgrade/... ./experiments/cryptoupgrade/...`、相关集成 smoke test 和 OpenSpec 校验。

回滚时按阶段保留根 facade：内部 package 迁移可逐模块恢复旧转发实现；实验目录迁移可恢复旧路径而不改变运行时数据。任何阶段均不得自动删除历史结果。

## Open Questions

- `vote` 与 `vote2` 应按协议、证明系统还是论文实验名称重命名，需要结合两套实现的实际用途另行确认。
- `preload/elgmal` 是保留的未完成实验还是应删除/修正为 `elgamal`，需在实现对应阶段先确认引用和实验价值。
- 历史结果是整体移动并保留路径映射，还是仅修改新结果默认路径；实现前应根据论文脚本的外部引用情况决定。
