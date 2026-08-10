## 上下文

geth 的 native precompile 通过 `core/vm.PrecompiledContract` 暴露，`contracts.go` 按分叉规则维护地址到实现的映射，`RunPrecompiledContract` 统一完成 gas 扣减、tracing hook 和执行。当前仓库还存在 `core/vm/contracts_cryptoupgrade.go`，它把 `common.CodeStorageAddress` 上的 `CodeStorage` ABI 调用接入动态上传/plugin 执行路径。

`cryptoupgrade/algorithm` 中的候选算法文件是 `package main`，用于 `go build -buildmode=plugin`，不能直接被 `core/vm` import。部分函数还会读取系统随机数，例如 `RandomBytes`、`ShamirSplit`、`SchnorrProve`，以及空 seed 下的密钥生成函数；这类行为不能进入共识相关的 precompile 执行路径。

## 目标 / 非目标

**目标：**

- 将选定的 deterministic `cryptoupgrade` 算法作为 native precompile 注册到当前客户端。
- 保持现有 Ethereum precompile、`Blake2bSum256Address` 和 `CodeStorageAddress` 行为不变。
- 为每个新增 precompile 定义稳定地址、名称、ABI 输入类型、ABI 输出类型和 deterministic gas 规则。
- 让 precompile 可通过普通 EVM `CALL`、`CALLCODE`、`DELEGATECALL` 和 `STATICCALL` 进入现有 `RunPrecompiledContract` 路径。
- 增加直接 precompile 测试和 EVM call-path 测试，验证输出、gas 扣减和错误行为。

**非目标：**

- 不把运行时上传的 Go plugin 动态转换成 precompile。
- 不在 precompile 中执行任何依赖系统随机数、文件系统、编译器或 plugin loader 的逻辑。
- 不新增第三方依赖。
- 不替换现有 `CodeStorage` 上传、查询和调用接口。

## 设计决策

1. 使用保留的 cryptoupgrade 地址范围。

   新增算法 precompile 使用 `common.CodeStorageAddress`、`common.MutiVoucherAddress`、`common.CoinBaseAddress` 和 `common.Blake2bSum256Address` 之后的连续地址，例如从 `0x47` 开始。这样可以避免与 Ethereum 标准低地址 precompile、现有 cryptoupgrade 系统地址和未来内置地址发生冲突。

   备选方案：复用 `CodeStorageAddress` 并通过 method selector dispatch。已拒绝，因为面向用户的目标是 native precompile 行为，precompile 身份应基于地址，而不是基于 ABI method。

2. 将 VM 集成保持为 `PrecompiledContract` wrapper。

   `core/vm` 应用小型实现包装 cryptoupgrade 算法入口，提供 `RequiredGas`、`Run` 和 `Name`，然后把生成的 map 合并到当前 chain rules 返回的激活 precompile 集合中。这样可以保留 geth 正常的 gas 计费和 tracing 行为，并把 `isCryptoUpgradeCall` 限制在既有 `CodeStorage` special case 上。

   备选方案：在 `EVM.Call` 中增加类似 `isCryptoUpgradeCall` 的另一个特殊分支。已拒绝，因为普通 precompile 已经具备正确的 dispatch 路径。

3. 在 `cryptoupgrade/algorithm` 外增加可 import 的 native 算法注册表。

   由于 plugin 源文件是 `package main`，共享 deterministic 实现应放在可 import package 或根级 cryptoupgrade 文件中，并带有地址、算法名、输入 ABI 类型、输出 ABI 类型、gas rule 和 handler 等 registry records。现有 plugin 候选可以继续保留在 `cryptoupgrade/algorithm` 中；实现可以按需抽取共享代码，或为选定 deterministic 函数增加薄 native wrapper。

   备选方案：precompile 运行时编译并加载 `cryptoupgrade/algorithm` 文件作为 plugin。已拒绝，因为 precompile 执行必须确定、快速，并且独立于本地编译器/plugin 状态。

4. 每个 precompile 使用直接 ABI payload。

   每个算法地址精确标识一个入口。调用输入仅为 ABI 编码参数，不包含 method selector，也不包含 `callFunc(string,bytes)` 包装。输出使用与 cryptoupgrade serialization helper 相同的 `accounts/abi` 类型处理，返回 ABI 编码值。

   备选方案：要求每个 precompile 都使用 `CodeStorage.callFunc` ABI 形状。已拒绝，因为地址已经选择算法，precompile 接口不应再依赖动态算法名。

5. 仅暴露确定性、有界入口。

   hash、调用方提供 IV 的 AES-CBC、有界 iteration/key length 的 PBKDF2、公钥派生、使用调用方提供 key material 的签名、验证、Pedersen commit/verify、Schnorr public/verify、DH public/secret、Shamir recover 和简单算术等 deterministic 函数可以注册。生成随机数的函数不注册，除非实现拒绝随机路径并要求显式 deterministic input。

6. 执行前确定性计算 gas。

   每个 registry entry 定义 `RequiredGas(input []byte) uint64` 规则。简单函数可以使用 base plus per-word cost。PBKDF2 和模运算等参数敏感算法必须只解码 gas 所需字段，或回退到保守 malformed-input cost。溢出必须饱和为 `math.MaxUint64`，让普通 out-of-gas 行为在 `Run` 前发生。

## 风险 / 权衡

- 随机或环境相关算法可能导致共识非确定性 -> 只注册 deterministic 入口，并拒绝 random fallback 输入。
- CPU-heavy 算法可能 gas 定价过低 -> gas 公式按算法定义，测试覆盖代表性高成本输入、格式错误输入和 out-of-gas 路径。
- plugin 候选和 native precompile 实现可能漂移 -> 在实际可行时抽取共享 deterministic 逻辑，并用 fixture 测试比较 precompile 输出和选定参考实现。
- 地址可能与未来 precompile 冲突 -> 所有 cryptoupgrade 算法地址都集中声明在 `common` 中，并测试其与激活 precompile map 的唯一性。
- VM 依赖增长 -> `core/vm` 只 import 小型 cryptoupgrade registry/wrapper 接口；算法专属代码留在 VM 文件之外。

## 迁移计划

1. 增加 central address constants 和 deterministic cryptoupgrade precompile entries 的 registry。
2. 增加 VM wrapper，并将 cryptoupgrade precompile map 合并进激活 precompile 集合。
3. 增加 registry uniqueness、gas、ABI decode/encode 和 output fixtures 的单元测试。
4. 增加 EVM call-path 测试，确保正常 precompile dispatch 能到达新算法。

回滚方式是移除 cryptoupgrade precompile map 合并和地址注册；既有 `CodeStorage` 行为不受影响。

## 开放问题

- 初始算法子集应在实现时根据 deterministic behavior 和 manageable gas formulas 最终确定。
- 如果 benchmark 需要产生随机数的操作，应使用显式 deterministic test seed，或继续留在 native precompile 集合之外。
