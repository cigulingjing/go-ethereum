## Context

技术栈：Go、ethereum/go-ethereum、Solidity、Clique 私有链、[wazero](https://github.com/tetratelabs/wazero)（纯 Go WASM runtime）、链下 TinyGo/WASI 用于把候选算法编成 `.wasm`。

当前 `cryptoupgrade` 的升级路径是：Solidity `uploadCodeVersion` 登记 gzip/base64 Go 源码 → 事件广播 → `activation.Service` 解码源码并执行 `go build -buildmode=plugin` → `pluginruntime` 在 Geth 进程内加载 `.so` → Solidity `callFunc` 反射调用导出符号。该路径把 Native 代码和节点本地编译差异同时放进宿主地址空间，无法支撑论文中的 WASM 字节码升级方案。

约束：最小侵入 Geth；`core/vm` 仍只走 `cryptoupgrade` facade；`callFunc` 调用形状不变；升级交易的 EVM 路径仍然只做登记和事件，不编译。

## Goals / Non-Goals

**Goals:**

- 把链上权威产物从 Go 源码改为 WASM bytecode，并以 `wasmHash` 标识同一份模块。
- 通过现有 CodeStorage 事件把字节码广播到各节点协处理器。
- 协处理器异步校验、编译、实例化 WASM，再按 Activation Block 选择版本。
- 用沙箱 WASM runtime 替换 Go plugin；调用仍从 Solidity `callFunc` 进入。
- 让现有 Lab1/Lab2/Lab3 实验入口能上传并调用 WASM 模块，形成可重复的冒烟路径。

**Non-Goals:**

- 不把 WASM 解释进 EVM，不替换 Precompiled Contract。
- 不在本 change 重跑正式 10 节点实验、不补采 stateRoot、不写 Lab4 密码迁移、不改论文 LaTeX。
- 不删除 `uploadCodeImmediate` ABI，但新路径不以 immediate 为合法升级模式。
- 不支持任意 host import、WASI 文件系统/网络，或不可信用户代码的完整安全证明。

## Decisions

### 1. 运行时选用 wazero，而不是 wasmer/wasmtime 或继续 Go plugin

选用 `github.com/tetratelabs/wazero`，因为纯 Go、无 CGO，符合 Geth 构建约束，并提供 `CompileModule`、fuel metering 和内存上限。

备选：wasmer-go / wasmtime-go 需要 CGO；继续 Go plugin 无法提供沙箱，也不能把字节码当作可广播权威产物。

### 2. 保留 CodeStorage 方法名，只替换载荷语义

保持 `uploadCode` / `uploadCodeVersion` / `callFunc` 名称和事件 `codeVersionUploaded(string,uint64,uint64)`，避免扩 ABI 面。`code` 字段改为 WASM bytecode 的编码（继续 gzip+base64，便于复用现有 calldata 通道）。协处理器解码后计算 `wasmHash = keccak256(rawWasm)`，并写入本地 metadata。

备选：新增 `uploadWasmVersion(bytes wasm, bytes32 wasmHash)`。语义更直观，但要同步改 ABI、dispatcher、实验打包和已有测试；本 change 用载荷替换换更小的 EVM 侵入。

### 3. 协处理器编译发生在 event/activation 路径，不在 EVM 交易路径

升级编排改为：

```text
event → wasmcodec.Decode → wasmhash.Verify
      → wasmruntime.Compile → repository.Save → wasmruntime.Instantiate
```

`uploadCodeVersion` 仍然只写 metadata 并发出事件。`compiler` package 从 `go build -buildmode=plugin` 改为 wazero AOT compile；`pluginruntime` 替换为 `wasmruntime`。

备选：在 `Dispatcher.Run` 内同步编译。这会把编译重新放进升级交易关键路径，违背异步准备。

### 4. Guest ABI 固定为 bytes-in / bytes-out，ABI 编解码留在宿主

WASM 模块必须导出 `execute(input_ptr: i32, input_len: i32) -> i32`。宿主把 `callFunc` 收到的 `input` 写入线性内存；模块返回指向 length-prefixed output 的指针；宿主再按 `otype` 校验并返回 EVM。

这样现有 `callcodec` 仍可做类型检查，但不再用 Go `plugin` 反射调用任意签名。链下用 TinyGo 把 `algorithm/go` 候选包成带 `execute` 导出的 `.wasm`，产物放 `cryptoupgrade/algorithm/wasm/`。

备选：为每个算法保留 Go 风格导出并在 WASM 里模拟反射。实现复杂，且 TinyGo/Rust 模块无法共用。

### 5. 沙箱默认拒绝 host import，并用 fuel/memory/compile timeout 限制资源

实例化配置：

- 不允许任意 WASI/filesystem/clock/random import；只注入协处理器提供的 `env.execute_abort`（可选）或零 import。
- `WithFuelCost` / 调用 fuel 上限与算法 metadata `gas` 对齐为运行时中止条件，失败不得拖垮 Geth。
- 内存页上限和 compile timeout 在 repository/runtime 配置中固定。
- 导出符号只允许 `execute` 以及模块名规范化后的可选别名。

备选：启用完整 WASI 以便直接跑 TinyGo 默认模块。该方案会重新引入时间、随机和 IO，破坏确定性。

### 6. 本地 workspace 从 src/so 改为 wasm/compiled

默认基础目录仍是 `./plugin`（环境变量 `GETH_CRYPTOUPGRADE_PLUGIN_DIR` 不变），子目录改为：

```text
<base>/wasm/<Name>-<version>.wasm
<base>/compiled/<Name>-<version>     # wazero compilation cache
<base>/algorithm_info.json
<base>/algorithm_versions.json
```

`src/` 与 `so/` 不再作为激活路径。旧 Go plugin 文件可留在磁盘，但运行时不得加载。

备选：继续把 `.wasm` 写进 `src/`、把 AOT cache 写进 `so/`。名称会误导后续实验和论文表述。

### 7. 调用时 fail-closed，不回退旧版本

`RunAt` 按区块高度选出目标版本后，若本地没有该版本的已编译 WASM 实例，必须返回 required-version-missing，不得执行旧模块。事件处理失败只影响本节点 prepared 状态，不回滚链上计划。

备选：未就绪时继续跑旧版本。这会在同一高度产生不同语义，正是 Activation Block 要避免的问题。

### 8. 实验入口改传 `.wasm`，旧 Go plugin JSON 不作为新结果

`experiments/cryptoupgrade` 的升级上传改为读取 `cryptoupgrade/algorithm/wasm/*.wasm`。Lab 命令名称保持，便于后续重跑；本 change 只要求冒烟通过，不重写正式结果文件。

## Risks / Trade-offs

- [Risk] TinyGo 产物依赖 WASI，和零 import 沙箱冲突。 → Mitigation: 为候选算法提供无 WASI 的 `execute` 包装；CI 用最小 `Add.wasm` 先打通，再批量编译其余算法。
- [Risk] WASM 调用比 Go plugin 慢，旧 Lab2 结论失效。 → Mitigation: 本 change 不沿用 Native-like 结论；效率数字留给后续正式重跑。
- [Risk] `string code` 承载二进制会增大 calldata。 → Mitigation: 继续 gzip；对超限字节码做 size limit，并在失败时拒绝登记。
- [Risk] wazero 版本差异导致编译缓存不兼容。 → Mitigation: metadata 记录 runtime 名称与版本；缓存 miss 时重新 CompileModule。
- [Risk] 旧实验脚本仍上传 `.go`。 → Mitigation: 解码阶段校验 magic `\0asm`，非 WASM 输入直接失败。

## Migration Plan

1. 增加 `internal/wasmcodec`、`internal/wasmruntime`，用 `Add.wasm` 单测打通 decode → compile → execute。
2. 修改 `activation.Service` 与 repository 路径，切断 compiler/pluginruntime 的 Go plugin 依赖。
3. 同步 CodeStorage 载荷校验、event 激活和 facade 导出。
4. 为 7 个候选算法生成 `.wasm`（至少 Add/Sha256 先可用），更新实验上传路径。
5. 跑 cryptoupgrade 单元测试和 experiments smoke；失败则保留新 runtime 在 internal，facade 暂不切换。

回滚：恢复 `activation` 对 `compiler` + `pluginruntime` 的注入，并继续接受 Go 源码载荷。链上若已写入 WASM 字节码，旧节点无法加载，因此测试网数据需按新格式重传。

## Open Questions

- TinyGo 是否作为仓库强制工具链，还是只提交预编译 `.wasm` 并由文档说明复现命令。
- `wasmHash` 是否写入链上合约存储，还是仅由节点从 `code` 字段计算。当前设计取后者，以保持 ABI 稳定。
- fuel 与 CodeStorage `gas` 是否 1:1 映射，还是使用独立 runtime fuel 上限。
