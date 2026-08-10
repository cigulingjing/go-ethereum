## 1. 路径解析器

- [x] 1.1 在 `cryptoupgrade/path.go` 中增加集中式 plugin path resolver，包含 base、source、shared-object 和 metadata paths。
- [x] 1.2 保留默认 `./plugin` 布局，同时确定性解析启动工作目录位置。
- [x] 1.3 增加进程级 plugin 基础目录覆盖，并在代码中记录所选环境变量名称。
- [x] 1.4 更新 `gofilePath` 和 `sofilePath`，使算法专属路径来自 resolver。

## 2. 运行时集成

- [x] 2.1 将 `algoInfoPath` 和 `compressedPath` 的直接使用替换为 resolver-backed helper。
- [x] 2.2 修改目录初始化，使其返回 `os.MkdirAll` 错误。
- [x] 2.3 在写入源码/plugin 前，从 `ActivateAlgorithm` 传播目录初始化错误。
- [x] 2.4 在写入元数据前，从 `Store` 传播目录初始化错误。
- [x] 2.5 package 初始化应容忍缺失元数据，同时为非缺失加载错误记录带路径上下文的日志。

## 3. 测试和文档

- [x] 3.1 增加默认路径推导单元测试。
- [x] 3.2 增加绝对和相对覆盖路径推导单元测试。
- [x] 3.3 增加目录创建和错误传播单元测试。
- [x] 3.4 增加或更新 cryptoupgrade 文档，说明默认布局、覆盖用法、节点本地隔离和迁移预期。

## 4. 验证

- [x] 4.1 对修改的 Go 文件运行 `gofmt` 和 `goimports`。
- [x] 4.2 对 cryptoupgrade package 运行目标测试。
- [x] 4.3 如果实现改动超出 path helper 并影响共享运行时行为，运行 `go run ./build/ci.go test -short`。
- [x] 4.4 运行 `openspec validate optimize-plugin-directory`。
