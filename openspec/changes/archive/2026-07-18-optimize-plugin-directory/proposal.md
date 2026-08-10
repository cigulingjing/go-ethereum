## 为什么

当前 cryptoupgrade 运行时将 plugin 源码、已编译 `.so` 文件和 `algorithm_info.json` 存放在硬编码的相对目录 `./plugin` 下。这会让行为依赖 geth 进程工作目录；对于多节点本地部署、service manager、测试和从不同路径启动的生产节点来说，这种行为比较脆弱。

目录布局应默认保持兼容，但运行时需要清晰的路径管理契约，使 operator 可以把 plugin 制品放在确定性的节点专属目录下。

## 变更内容

- 增加集中式 cryptoupgrade plugin 目录配置层，只解析一次基础目录，并从中推导 `src`、`so` 和 `algorithm_info.json` 路径。
- 未显式配置目录时，保留既有 `./plugin` 布局作为默认值。
- 允许通过适合 geth 启动和测试的进程级配置机制覆盖 plugin 基础目录。
- 强化目录初始化：返回并处理错误，而不是忽略 `os.MkdirAll` 失败。
- 保持算法上传、编译、加载、调用和元数据语义不变。
- 文档化默认布局、覆盖行为和迁移预期。

## 能力

### 新增能力

- `cryptoupgrade-plugin-directory-management`：定义 cryptoupgrade 如何解析、初始化和使用其运行时 plugin 制品目录。

### 修改能力

- 无。

## 影响

- 影响代码：`cryptoupgrade/path.go`、`cryptoupgrade/init.go`，以及假定全局 path constants 或忽略目录初始化错误的 call sites。
- 影响运行时行为：默认路径仍为 `./plugin`，但部署可以选择显式 plugin 基础目录。
- 测试：增加聚焦单元测试，覆盖默认解析、覆盖解析、路径推导、目录创建和错误传播。
- 依赖：不新增外部依赖。
