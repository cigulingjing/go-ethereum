## 1. 建立迁移基线

- [x] 1.1 记录 `cryptoupgrade` 根 package 当前导出 API、Geth 导入点及对应测试清单
- [x] 1.2 记录根目录文件到目标内部模块的迁移映射及当前 package 依赖图
- [x] 1.3 统计 benchmark 和 network 中重复的源码压缩实现及其输入输出差异
- [x] 1.4 确认 `vote`、`vote2`、`elgmal` 的引用和实验用途，并记录本次允许执行的重命名范围
- [x] 1.5 确认历史结果外部引用方式，选择整体迁移或只切换新结果默认路径

## 2. 拆分 codec 与共享类型

- [x] 2.1 创建 `cryptoupgrade/internal/model` 并迁移跨模块共享类型，保证不包含流程逻辑
- [x] 2.2 创建 `cryptoupgrade/internal/sourcecodec`，迁移 gzip/base64 源码编解码和输入校验
- [x] 2.3 为 `sourcecodec` 增加 round-trip、损坏输入、文件读写和大小边界测试
- [x] 2.4 创建 `cryptoupgrade/internal/callcodec`，迁移 `callFunc` ABI 输入输出编解码
- [x] 2.5 迁移现有 ABI 编解码测试并验证编码结果与迁移前一致
- [x] 2.6 在根 facade 保留必要的兼容入口，并删除已无调用的重复 codec 实现

## 3. 拆分 repository、compiler 与 plugin runtime

- [x] 3.1 创建 `cryptoupgrade/internal/repository`，迁移 plugin workspace 路径解析
- [x] 3.2 迁移算法元数据读写并保持 `algorithm_info.json` schema 和原子持久化行为
- [x] 3.3 增加默认路径、环境变量覆盖和元数据读写测试
- [x] 3.4 创建 `cryptoupgrade/internal/compiler`，以显式源码路径和输出路径封装 Go Plugin 编译
- [x] 3.5 增加编译成功、编译失败和原子发布不破坏旧制品的测试
- [x] 3.6 创建 `cryptoupgrade/internal/pluginruntime`，迁移 plugin 加载缓存、符号解析和调用逻辑
- [x] 3.7 增加 loader 缓存、缺失符号、无效 plugin 和并发访问测试

## 4. 引入升级编排与适配层

- [x] 4.1 创建 `cryptoupgrade/internal/activation` 并定义可注入的 codec、compiler、repository 和 loader 边界
- [x] 4.2 将 `ActivateAlgorithm` 流程迁入 activation service，并保持失败不影响节点存活
- [x] 4.3 增加激活成功及解码、编译、持久化、加载各阶段失败的单元测试
- [x] 4.4 创建 `cryptoupgrade/internal/event`，迁移 `codeUploaded` 订阅、解析和链上信息查询
- [x] 4.5 验证 event 只调用 activation service，且 upload receipt 不作为激活完成判据
- [x] 4.6 创建 `cryptoupgrade/internal/evm`，迁移 CodeStorage 调度、ABI 适配和 precompile registry
- [x] 4.7 更新根 facade 的组装与转发，保持 `core/vm`、`node`、`cmd/geth` 导入路径不变
- [x] 4.8 运行 CodeStorage、event activation、precompile 和 Geth 集成定向测试

## 5. 整理算法实现边界

- [x] 5.1 将 `cryptoupgrade/preload` 迁移为 `cryptoupgrade/builtin` 并更新内部 import
- [x] 5.2 将 builtin registry 与 plugin 候选源码目录解耦，保持现有算法调用结果不变
- [x] 5.3 根据任务 1.4 的结论处理明确安全的拼写或语义目录重命名
- [x] 5.4 更新算法接口检查、测试 fixture 和文档引用
- [x] 5.5 运行 builtin、动态候选算法和 native precompile 回归测试

## 6. 建立仓库级实验目录

- [x] 6.1 创建 `experiments/cryptoupgrade` 目标目录和说明文档
- [x] 6.2 将 `cryptoupgrade/bench/cmd` 迁移到 `experiments/cryptoupgrade/bench/cmd`
- [x] 6.3 将 `cryptoupgrade/network` 迁移到 `experiments/cryptoupgrade/network` 并更新 import
- [x] 6.4 将 Docker 资产和网络 YAML 迁移到 `experiments/cryptoupgrade/deployments`
- [x] 6.5 将实验文档迁移到 `experiments/cryptoupgrade/docs` 并修正命令示例
- [x] 6.6 按任务 1.5 的结论迁移或兼容历史 results，并切换新结果默认输出路径
- [x] 6.7 更新仓库内 Go、shell、Docker、YAML、测试和 OpenSpec 的旧路径引用

## 7. 分离 network 验证与 cryptoupgrade smoke

- [x] 7.1 从 network 验证中移除 Add 源码路径、CodeStorage ABI 和算法上传调用逻辑
- [x] 7.2 创建 `experiments/cryptoupgrade/smoke` 并迁移上传、激活等待和调用结果校验
- [x] 7.3 将 benchmark、smoke 和升级工具统一切换到共享源码压缩入口
- [x] 7.4 为纯 network 验证和 cryptoupgrade smoke 分别增加测试
- [x] 7.5 更新实验命令，使 network 验证和 smoke 可独立执行或顺序组合

## 8. 边界检查与最终验证

- [x] 8.1 更新布局检查，禁止 `cryptoupgrade` 运行时反向导入 `experiments/cryptoupgrade`
- [x] 8.2 增加检查以识别根目录实验资产、旧实验命令路径和重复源码压缩实现
- [x] 8.3 对所有迁移的 Go 文件运行 `gofmt` 并检查编辑文件的 linter diagnostics
- [x] 8.4 运行 `go test ./cryptoupgrade/... ./experiments/cryptoupgrade/...`
- [x] 8.5 运行关键实验命令帮助检查、多节点 network 验证和动态升级 smoke test
- [x] 8.6 验证默认 plugin 路径、环境变量覆盖、ABI、gas 和实验结果 schema 未发生变化
- [x] 8.7 更新本 change 的任务状态、旧新路径映射和最终验证记录
- [x] 8.8 运行 `openspec validate refactor-cryptoupgrade-modules`
