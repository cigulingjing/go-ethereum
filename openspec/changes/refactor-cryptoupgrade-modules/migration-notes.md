## 第 1 组：迁移前基线

### 基线取值方法

本记录以 Git `HEAD`（`28e1e3ba3`）中的根 package 实现作为“迁移前”基线，再与当前
工作区对照。原因是当前工作区已经完成 `internal` 模块拆分和
`cryptoupgrade/preload` → `cryptoupgrade/builtin` 迁移，直接只看当前文件会漏掉旧职责。
可使用以下命令复核：

```text
git show HEAD:cryptoupgrade/<file>
git diff HEAD -- cryptoupgrade
go list -f '{{.ImportPath}} -> {{join .Imports ", "}}' ./cryptoupgrade ./cryptoupgrade/internal/...
```

以下结论只覆盖当前仓库和当前可见 Git 历史；仓库外脚本、论文处理目录、人工复制文件及
其他不可见消费者无法由本仓库证明，均明确记为未知。

## 1.1 根 package API、Geth 导入点和测试

### 迁移前导出 API

按 `HEAD:cryptoupgrade/*.go` 复核，根 package 的导出面如下：

- ABI 与 CodeStorage：`CodeStorageABI`、`CodeStorageLogSink`、
  `IsCodeStorageCall`、`RequiredGasForCodeStorageCall`、
  `RunCodeStorageCall`、`IsUpgradeAlgorithm`。
- event 与生命周期：`ParseReceipt`、`BindCodeUploaded`、`ActivateAlgorithm`、`Store`。
- 动态/builtin 调用：`CallProcessor`、`RequiredGas`、`RequiredGasForCall`、`RunCall`、
  `PluginCompile`。
- `callFunc` codec：`ParseCall`、`UnpackCall`、`UnpackInput`、`PackOutput`。
- native precompile：`Precompile`、`Precompiles`、`PrecompileAddresses`、
  `PrecompileByName`，以及 `Precompile` 的 `Address`、`Name`、`InputTypes`、
  `OutputTypes`、`RequiredGas`、`Run` 方法。

其中 Geth 主流程实际使用的是较小子集；其余导出项仍属于迁移前兼容面，不能仅因主流程
没有直接调用而在重构中无记录删除。

### Geth 导入点

- `core/vm/contracts_cryptoupgrade.go`：使用 `Precompile`、`Precompiles`、
  `PrecompileAddresses`、`IsCodeStorageCall`、`RequiredGasForCodeStorageCall` 和
  `RunCodeStorageCall`，负责 native precompile 注册及 CodeStorage 特殊调用分派。
- `node/node.go:257`：节点关闭时调用 `Store` 持久化算法元数据。
- `cmd/geth/main.go:343`：节点启动后异步调用 `BindCodeUploaded`。
- `core/vm/contracts_cryptoupgrade_test.go`：直接使用 `Precompiles`、
  `PrecompileByName`、`Precompile` 和 `CodeStorageABI` 验证 EVM 集成。

上述三个生产导入点及 `core/vm` 集成测试在当前工作区相对 `HEAD` 无 diff，仍只导入
`github.com/ethereum/go-ethereum/cryptoupgrade`，没有绕过 facade 导入
`cryptoupgrade/internal`。

### 测试基线与当前补充

- 迁移前根包测试：`cryptoupgrade/path_test.go` 覆盖默认 plugin 路径、环境变量覆盖、
  目录创建和错误传播；`cryptoupgrade/precompile_test.go` 覆盖 registry、gas、ABI、
  算法 fixture 和错误输入。
- 迁移前 Geth 集成测试：`core/vm/contracts_cryptoupgrade_test.go` 覆盖注册、执行、
  gas、CALL/STATICCALL 和 CodeStorage 特殊分派。
- `HEAD` 中没有针对 event、CodeStorage 上传/激活分离、plugin loader、compiler 和
  codec 的独立测试，这是迁移前覆盖缺口。
- 当前工作区已补充 `cryptoupgrade/code_storage_test.go`、
  `cryptoupgrade/bind_event_test.go`、`cryptoupgrade/builtin_call_test.go`，以及
  `cryptoupgrade/internal/{sourcecodec,callcodec,repository,compiler,pluginruntime,activation,event,evm}/*_test.go`。
  这些是迁移验证证据，不反向冒充迁移前已有测试。

## 1.2 旧文件到目标模块映射及依赖图

### 旧新映射

- `cryptoupgrade/algorithm_info.go` → `internal/model`（算法元数据值对象）+
  `internal/repository`（内存状态、JSON schema 和持久化）；根 facade 保留 `Store`。
- `cryptoupgrade/bind_event.go` → `internal/event`（订阅、解析、链上查询）；根 facade
  负责组装 activation 回调并保留 `BindCodeUploaded`。
- `cryptoupgrade/code_storage.go` → `internal/evm`（selector、gas、ABI dispatch）+
  `internal/activation`（解码、编译、保存、加载流程）；根 facade 保留 CodeStorage API。
- `cryptoupgrade/compressed.go` → `internal/sourcecodec`；当前根文件仅保留内部兼容转发。
- `cryptoupgrade/serialize.go` → `internal/callcodec`；当前根文件保留旧导出 codec 转发。
- `cryptoupgrade/path.go` → `internal/repository` 的 `Workspace` 路径解析；根 facade 保留
  现有运行时路径组装。
- `cryptoupgrade/plugin.go` → `internal/compiler`（plugin 构建）+
  `internal/pluginruntime`（加载、符号解析、反射调用）+ `internal/callcodec`
  （参数/返回值）+ `builtin` registry（编译进节点的实现）；根 facade 保留调用入口。
- `cryptoupgrade/plugin_loader.go`、`plugin_loader_unsupported.go` →
  `internal/pluginruntime/open_supported.go`、`open_unsupported.go`。
- `cryptoupgrade/init.go` → 根 facade 组装 + `internal/evm.MustCodeStorageABI` +
  `internal/repository` 初始化，不下沉为带全局副作用的基础模块。
- `cryptoupgrade/precompile.go` → `internal/evm` 的 registry/ABI/gas 边界；当前工作区已使用
  `internal/evm.Registry`，但具体 deterministic handler 仍在根文件，后续若移动只能保持
  地址、ABI、gas 和结果不变。
- `cryptoupgrade/preload/**` → `cryptoupgrade/builtin/**`，文件内容和
  `vote`/`vote2`/`elgmal` 子目录名保持；`preload/preload_algo.go` →
  `builtin/registry.go`。
- 新增的 `cryptoupgrade/activation.go` 是 facade 的依赖组装点，不对应一个迁移前旧文件。

### 当前依赖图

```text
core/vm ─┐
node ────┼─> cryptoupgrade facade
cmd/geth ┘          │
                    ├─> internal/event ──────────> internal/model
                    ├─> internal/evm ────────────> internal/model
                    ├─> internal/activation ─────> internal/model
                    ├─> internal/repository ─────> internal/model
                    ├─> internal/sourcecodec
                    ├─> internal/callcodec
                    ├─> internal/compiler
                    ├─> internal/pluginruntime
                    └─> builtin registry ─> vote ─> bls12381

bench/network ─> cryptoupgrade facade
cryptoupgrade runtime ─/─> bench/network（目标迁移后为 experiments/cryptoupgrade）
```

`internal/activation` 通过注入接口协调 codec、compiler、repository 和 loader，因此其 Go
import 只需要 `internal/model`；运行时流程仍是
`event → activation → sourcecodec/compiler/repository/pluginruntime`。`vote2` 与
`elgmal` 不在 builtin registry 依赖链中。

## 1.3 benchmark/network 重复源码压缩

当前共有 **9 处**实验私有实现，全部产生“gzip 字节 + `base64.StdEncoding`（standard、
带 padding）字符串，与 `internal/sourcecodec` 的线格式相同，但输入方式、附加统计和错误
包装不同：

- `bench/cmd/benchblake2b/main.go:619`、`benchcandidate/main.go:489`、
  `benchcall/main.go:578`、`upgradeflow/main.go:321`：`compressFile(path)`，
  `os.Open` + `io.Copy`，只返回 encoded string。
- `network/validate.go:270`：`compressFile(path)`，`os.ReadFile` + `Write`，只返回
  encoded string；写入失败时主动关闭 writer，错误不加上下文。
- `bench/cmd/benchrealchain/main.go:462`：`compressSource(path)`，读取整个文件并返回
  `sourceBytes + encoded`，带 `read/gzip/close` 错误上下文。
- `bench/cmd/benchexecutionefficiency/main.go:1066`：`compressSource(path)`，
  流式读取并用 `io.TeeReader` 统计原始字节，返回 `sourceBytes + encoded`。
- `bench/cmd/benchupgradeefficiency/main.go:605`：`compressSource(path)`，返回
  `sourceBytes + gzipBytes + encoded`。
- `bench/cmd/benchupgradelatency/main.go:1257`：`buildPayload(source []byte, ...)`
  内联压缩，除 encoded 外还记录 source/gzip/base64/calldata 字节数和 payload hash。

统一迁移时最终都复用 `internal/sourcecodec.EncodeFile`/`Encode` 的实现；但目标
`experiments/cryptoupgrade` 位于 `cryptoupgrade` 目录之外，受 Go `internal` 可见性约束，
不能直接导入该 package，因此实验代码必须通过根 facade 提供的共享源码编码入口转发。
各 benchmark 特有的长度、calldata 和 hash 指标继续在调用方根据原始字节与编码结果计算，
不能为了去重删除实验字段或改变 JSON schema。`cryptoupgrade/compressed.go` 已在根包内部
转发共享 codec，但尚未提供实验层可用的导出入口，上述 9 处仍待第 7.3 项替换。

## 1.4 `vote`、`vote2`、`elgmal` 用途与重命名结论

- `builtin/vote/` 是当前 registry 使用的电子投票实现。
  `builtin/registry.go:20-23` 注册 `VerifyDecryptionShareZKP`、`AggregateShare`、
  `VerifyBallotZKP`、`AggregateBallot`；`builtin/registry_test.go` 和
  `cryptoupgrade/builtin_call_test.go` 提供调用结果证据。
- `builtin/vote2/` 是另一套 ballot/vector-ZKP 实验实现。仓库搜索只发现目录内部类型和
  函数互调，以及 OpenSpec 对该名称的讨论；registry、其他 package、脚本和实验文档均未
  引用它。仓库没有给出可验证的协议名、证明系统名或版本含义。
- `builtin/elgmal/elgamal.go` 声明 `package ec_elgamal`，依赖 builtin BLS12-381，
  但没有被 registry 或其他 package 引用；仓库只在 OpenSpec 中讨论该目录名。`elgmal`
  很可能是拼写问题，但“看起来像 typo”不足以证明安全重命名。

决策：本次唯一可证安全的目录重命名是已完成的父目录
`cryptoupgrade/preload` → `cryptoupgrade/builtin`；保留 `vote`、`vote2` 和
`elgmal` 原名，不删除未注册实验。仓库外是否存在按这些旧路径访问的脚本未知，因此不能
以“仓库内无引用”推断外部无引用。将来若要语义重命名，须先取得实验/论文对应关系，并
提供显式旧新映射或兼容入口。

## 1.5 历史 results 引用证据与迁移决策

### 仓库内可验证证据

- 代码默认路径：`bench/cmd/benchexecutionefficiency/main.go:309` 和
  `bench/cmd/benchrealchain/main.go:298` 默认写入 `cryptoupgrade/results/...`；
  其他命令也允许通过输出参数写入该树。
- 脚本约束：`cryptoupgrade/check_layout.sh:31,46` 要求结果位于
  `cryptoupgrade/results`。
- 文档引用：`cryptoupgrade/docs/upgrade_latency.md` 和
  `cryptoupgrade/docs/real_chain_upgrade_vs_precompile.md` 包含旧默认路径及复现实例。
- OpenSpec 引用：`refactor-cryptoupgrade-layout` 的 proposal/design/spec/tasks 与
  `lab1-upgrade-latency/tasks.md` 明确引用旧结果树和具体 run 目录。
- 结果自引用：多个 `result.json`、`rounds.csv` 记录
  `/home/liuqi/project/paper/go-ethereum/cryptoupgrade/results/...` 绝对路径或旧相对路径，
  例如 `results/upgrade-latency/lab1-10nodes-all-20260812/result.json` 和
  `rounds.csv`。移动文件不能自动使这些历史字段有效，改写又会破坏原始运行快照。
- 未发现生产 Geth 运行时代码读取历史结果；这些引用属于实验代码、脚本、文档、
  OpenSpec 和结果自身。

仓库搜索无法观察仓库外论文绘图脚本、引用管理、人工命令或其他副本。其是否直接引用
`cryptoupgrade/results` **未知**，不作“存在”或“不存在”的猜测。

### 选择：保留旧历史结果，只切换新默认路径

- 历史 `cryptoupgrade/results/**` 原地只读保留，不整体移动、不重写 JSON/CSV、日志、
  图表、生成源码中的绝对路径或测量值。
- 新运行的默认输出从 `cryptoupgrade/results/<experiment>/<run-id>` 切换到
  `experiments/cryptoupgrade/results/<experiment>/<run-id>`；显式 `-out`、
  `-output-dir`、`-output-json` 仍按用户给定路径工作。
- 旧新映射是：
  - 历史：`cryptoupgrade/results/**` → `cryptoupgrade/results/**`（保留原位）。
  - 新默认：`cryptoupgrade/results/<experiment>/<run-id>` →
    `experiments/cryptoupgrade/results/<experiment>/<run-id>`。
- 后续第 6.6/6.7 项只更新新结果默认值、布局检查、脚本和文档示例；引用具体历史 run 的
  复现记录继续指向旧位置。该选择同时满足“不删除或改写历史数据”和外部引用未知时的
  最小破坏原则。

## 第 6-7 组：实验层实施映射

当前实验入口和部署资产使用以下路径：

- `cryptoupgrade/bench/cmd/**` → `experiments/cryptoupgrade/bench/cmd/**`
- `cryptoupgrade/network/**` → `experiments/cryptoupgrade/network/**`
- network 中的 Add 上传、receipt、异步激活等待和 `callFunc` 校验 →
  `experiments/cryptoupgrade/smoke/**`
- `cryptoupgrade/docker/**` → `experiments/cryptoupgrade/deployments/docker/**`
- `cryptoupgrade/examples/networks/**` →
  `experiments/cryptoupgrade/deployments/networks/**`
- `cryptoupgrade/docs/**` → `experiments/cryptoupgrade/docs/**`
- 新结果默认根目录：`cryptoupgrade/results/<experiment>/<run-id>` →
  `experiments/cryptoupgrade/results/<experiment>/<run-id>`

历史 `cryptoupgrade/results/**` 不在移动清单中，继续作为原始运行快照原地只读保留。
实验源码编码统一通过根 facade 的 `EncodeSource`/`EncodeSourceFile` 进入
`internal/sourcecodec`，实验 package 不直接导入 `cryptoupgrade/internal`。

## 最终验证记录

- `go test ./cryptoupgrade/... ./experiments/cryptoupgrade/...`：通过。
- `go test ./core/vm ./node ./cmd/geth`：通过，Geth 集成点仍只导入根 facade。
- `cryptoupgrade/check_layout.sh`：通过，已覆盖 runtime 反向依赖、旧实验目录和重复压缩实现。
- `cryptoupgrade/algorithm/check_interfaces.sh`：通过，Go/Solidity 候选接口保持一致。
- plugin 默认路径、环境变量覆盖、CodeStorage upload/gas、builtin 和 native precompile 定向测试：通过。
- multinode、升级延迟和执行效率命令帮助入口：通过，新配置和结果默认路径可见。
- 使用 `cryptoupgrade-geth:lab` 启动 2 节点 Clique 网络后，network validate 通过：
  chain ID 为 `11223344`，两个节点 peer count 均为 1，区块高度持续增长。
- 动态升级 smoke 通过：Add upload transaction 成功，bind-event 异步激活后
  `callFunc(100,100)` 返回 200；验证结果保存在
  `build/cryptoupgrade-networks/module-boundaries-smoke/{validate,smoke}.json`。
- 首次使用渲染后的 `network.yaml` 复现 smoke 时发现相对 `addSource` 随输出目录变化而失效；
  Render 现改为写入归一化配置快照，并增加快照重新加载测试。修复后使用渲染快照执行
  smoke 通过，临时容器及 Docker network 已清理。
