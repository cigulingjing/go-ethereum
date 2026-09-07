## 1. WASM runtime 基础

- [x] 1.1 在 Go module 中加入 wazero 依赖，并确认 Geth 构建不引入 CGO
- [x] 1.2 新增 `cryptoupgrade/internal/wasmcodec`，将 gzip/base64 载荷解码为原始 WASM 字节，并校验 `\0asm` magic 与 size limit
- [x] 1.3 为 wasmcodec 增加合法模块、Go 源码、损坏编码和超限输入测试
- [x] 1.4 新增 `cryptoupgrade/internal/wasmruntime`，封装 CompileModule、零 import 实例化、fuel/memory 上限和 `execute` 导出调用
- [x] 1.5 用最小 `Add.wasm` fixture 测试 compile、execute、非法 import、fuel 耗尽和缺失导出

## 2. 本地制品路径

- [x] 2.1 将 repository workspace 从 `src/` + `so/` 改为 `wasm/` + `compiled/`，并保留 `GETH_CRYPTOUPGRADE_PLUGIN_DIR` 与 `./plugin` 默认值
- [x] 2.2 按 `<Name>-<version>.wasm` 生成版本路径，停止把激活路径解析到 `.go` / `.so`
- [x] 2.3 在 metadata 中记录 `wasmHash`、runtime 名称和版本；旧 Go plugin 记录不得视为 prepared
- [x] 2.4 更新 repository 路径、目录创建失败和元数据加载测试

## 3. 升级编排切换

- [x] 3.1 让 `activation.Service` 注入 wasmcodec + wasmruntime，流程改为 decode → hash → compile → persist → instantiate
- [x] 3.2 保持 `uploadCode` / `uploadCodeVersion` 只登记 metadata 并发出事件，EVM 路径不编译 WASM
- [x] 3.3 事件处理失败时记录原因且不回滚链上计划，不标记 prepared
- [x] 3.4 更新 activation 成功/失败阶段测试，覆盖非 WASM 载荷和编译失败
- [x] 3.5 从 facade 组装中移除 Go plugin compiler/loader 的激活依赖

## 4. Solidity 调用与 fail-closed

- [x] 4.1 让 `callFunc` / `RunAt` 把 ABI `input` 交给 WASM `execute`，并按 `otype` 返回 bytes
- [x] 4.2 当区块高度已到达 Activation Block 但本地缺少目标 WASM 实例时，返回 required-version-missing，不得执行旧版本
- [x] 4.3 当区块高度尚未到达 Activation Block 且旧版本 prepared 时，继续执行旧 WASM 模块
- [x] 4.4 增加调用路径测试：成功执行、未就绪 fail-closed、非法模块不可调用

## 5. 候选 WASM 模块

- [x] 5.1 定义 guest ABI 包装约定，并在 `cryptoupgrade/algorithm/wasm` 放入至少 `add.wasm` 与 `sha256.wasm`
- [x] 5.2 记录其余 5 个算法的链下编译命令；能编译的补齐 `.wasm`，不能编译的在文档标明缺口
- [x] 5.3 增加静态检查：升级实验使用的模块存在、magic 合法、导出 `execute`

## 6. 实验入口

- [x] 6.1 修改升级延迟、执行效率和升级稳定性实验，改为读取 `.wasm` 作为 `code` 载荷
- [x] 6.2 在实验 JSON 中记录 `wasmHash` 与 Activation Block；禁止把旧 Go plugin 结果目录标为新正式数据
- [x] 6.3 跑通至少 Add 的单节点或 2 节点 smoke：上传 WASM、异步编译、`callFunc` 成功

## 7. 验证

- [x] 7.1 运行 `cryptoupgrade` 包测试，确认不再依赖 `go build -buildmode=plugin` 完成激活
- [x] 7.2 运行与 CodeStorage、event activation、callFunc 相关的定向测试
- [x] 7.3 运行 `openspec validate --changes dynamic-wasm` 并修复规格或任务不一致
