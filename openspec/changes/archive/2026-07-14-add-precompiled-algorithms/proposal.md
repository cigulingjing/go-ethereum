## 为什么

`cryptoupgrade/algorithm` 已经包含一组可用于升级和基准实验的密码算法实现，但这些实现目前还不能像 geth 内置预编译合约一样直接通过 EVM `CALL` 访问。将这些算法接入当前客户端的 precompile 路径，可以补齐性能基准中的“预编译合约实现”对照组，并提供与 Solidity/升级方案语义等价的 native 执行入口。

## 变更内容

- 为选定的 `cryptoupgrade/algorithm` 实现增加 native precompiled contract 绑定。
- 将 cryptoupgrade precompile 注册到激活的 geth precompile 集合中，并且不替换已有 Ethereum precompile。
- 为每个暴露的算法入口定义确定性的输入/输出 ABI 处理和确定性 gas 计费。
- 增加通过 EVM precompile 路径调用新 precompile 地址的测试，并将输出与底层算法行为对比。
- 保留既有 plugin 上传/调用流程；新增 precompile 是额外执行路径。

## 能力

### 新增能力

- `cryptoupgrade-precompiled-algorithms`：覆盖将 cryptoupgrade 候选算法暴露为 geth native precompiled contracts 的能力，包括地址注册、调用语义、gas 计费和验证。

### 修改能力

- 无。

## 影响

- 影响代码：`core/vm/contracts.go`、`core/vm/contracts_cryptoupgrade.go`、`core/vm/contracts_test.go`，以及按需新增的 cryptoupgrade precompile adapter 代码。
- 影响算法：`cryptoupgrade/algorithm` 下选定的确定性且适合 benchmark 的入口。
- 影响规划制品：`openspec/changes/add-precompiled-algorithms`。
- 不新增依赖。
