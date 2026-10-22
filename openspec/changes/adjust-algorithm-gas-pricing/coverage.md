# `callFunc` 计价覆盖矩阵

| EVM 入口 | 现有 Gas 获取 | 执行入口 | 输入与版本 | 计数隔离要求 |
| --- | --- | --- | --- | --- |
| `CALL` | `runCryptoUpgradeCall` → `RequiredGasForCodeStorageCallAt` | `RunCodeStorageCallAtContext` | `callFunc(string,bytes)`；动态版本按区块高度解析 | 交易共享计数 |
| `CALLCODE` | 同上 | 同上 | 同上 | 交易共享计数 |
| `DELEGATECALL` | 同上 | 同上 | 同上 | 交易共享计数 |
| `STATICCALL` | 同上 | 同上，`readOnly=true` | 同上 | 交易共享计数 |
| 上传/升级/查询 | dispatcher 返回 `0` 算法执行 Gas | dispatcher 相应方法 | 不属于 `callFunc` | 不计数 |
| 原生 precompile | 各算法自身的 `RequiredGas` | 对应 precompile `Run` | ABI 与 `callFunc` 不同 | 不计数 |

`ParseCall` 解码 `callFunc` 的名称和算法输入字节；动态算法输入 ABI 类型来自版本元数据 `IType`，版本由 `getActiveAlgorithmVersionInfoAt` 选择。当前 `builtin.Lookup` 无注册算法；已经加载的 WASM 算法集合与版本取决于实验链上的上传、激活历史，不应从仓库算法文件名推断覆盖率。标定必须针对实际激活版本与 ABI 输入采集，再锁定到实验链配置；不可自动采用本地测量值。
