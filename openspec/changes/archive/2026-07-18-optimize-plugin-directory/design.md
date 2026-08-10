## 上下文

`cryptoupgrade/path.go` 当前基于 package-level `compressedPath` 推导源码和 plugin 路径，`cryptoupgrade/init.go` 同时定义 `compressedPath = "./plugin"` 和 `algoInfoPath = "./plugin/algorithm_info.json"`。运行时状态因此依赖进程工作目录，并且路径字符串分散在多个文件中。

该目录包含运行时制品，而不是静态源码资源：

- `src/<Algorithm>.go`：解压后的上传算法源码。
- `so/<Algorithm>.so`：已编译 Go plugin。
- `algorithm_info.json`：持久化算法元数据。

这些文件是节点本地执行制品。它们应对 operator 保持确定性，对测试保持隔离，并足够明确以便调试，同时保持当前开发流程使用的默认路径。

## 目标 / 非目标

**目标：**

- 将所有 cryptoupgrade plugin 制品路径推导集中到一个小型 API 中。
- 保持 `./plugin/src`、`./plugin/so` 和 `./plugin/algorithm_info.json` 作为默认布局。
- 支持部署和测试使用显式基础目录覆盖。
- 返回并传播目录创建错误。
- 增加路径解析、目录初始化和兼容行为测试。

**非目标：**

- 不改变 `CodeStorage` ABI、算法上传编码、plugin 编译 flags、gas 计费或 precompiled contract 行为。
- 不自动迁移或重写已有 plugin 制品文件。
- 不新增依赖。
- 不在本 change 中增加 geth CLI flag；如后续需要，可在此能力之上继续分层实现。

## 设计决策

1. 使用集中式 `pluginPaths` 解析器。

   `path.go` 应拥有一个小 struct 或等价 helper，用于表示基础目录、源码目录、shared-object 目录和算法信息路径。call sites 应向解析器请求路径，而不是分别拼接 `compressedPath` 和 `algoInfoPath`。

   备选方案：保留当前 package globals，只增加注释。该方案无法修复职责分散或错误处理问题，也会让后续改动继续脆弱。

2. 从进程级覆盖值一次性解析基础目录。

   解析器应在命名环境变量存在时使用它，否则默认使用 `./plugin`。相对值应在初始化时基于当前工作目录解析，以避免后续工作目录变化静默移动制品位置。

   备选方案：从 geth `--datadir` 推导目录。这对节点本地存储很有吸引力，但需要将 node 或 command 配置接入 `cryptoupgrade`，比本次清理需要的跨包修改更大。

3. 保持默认兼容，但让自定义位置显式。

   现有用户不做任何配置时，应继续使用启动工作目录下的 `plugin` 目录。设置覆盖值的用户会获得隔离目录；如果需要复用已有 `algorithm_info.json` 或制品，应自行复制。

   备选方案：自动从 `./plugin` 复制到覆盖路径。该方案会造成所有权不清晰，并可能在不兼容二进制之间复制过期已编译 plugin，因此排除。

4. 让初始化失败可见。

   `directoryInit` 应返回 error。`ActivateAlgorithm` 和 `Store` 应传播该错误。package 初始化应在非 missing 的加载错误发生时记录带路径上下文的日志，而不是忽略。

   备选方案：目录无法创建时在 package init 中 panic。该行为对只读代码路径过于激进，并使节点启动失败模式更难控制。

## 风险 / 权衡

- 现有测试可能假定 `./plugin` 相对路径 -> 保持默认布局，并为新的覆盖路径增加隔离临时目录测试。
- 环境变量配置必须在 package 初始化前设置，才能影响初始元数据加载 -> 文档化启动要求，并让测试聚焦解析器 helper。
- operator 可能让多个节点指向同一个 plugin 目录 -> 文档化 plugin 目录是节点本地的，不应在并发运行的节点之间共享。
- 相对覆盖路径仍可能依赖启动目录 -> 初始化时解析为绝对路径，并在测试/日志中暴露已解析路径。
