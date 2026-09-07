## MODIFIED Requirements

### Requirement: 三组密码算法对照

系统 SHALL 支持同一密码算法的三组实验对照。

#### Scenario: Solidity 纯合约实现

- **WHEN** 选择 Solidity 对照组
- **THEN** 密码算法 SHALL 由 Solidity 合约实现
- **AND** 不应通过预编译合约或升级方案绕过算法主体执行

#### Scenario: 升级方案实现

- **WHEN** 选择升级方案对照组
- **THEN** 密码算法 SHALL 以 WASM bytecode 上传到 `CodeStorage`
- **AND** 各节点协处理器 SHALL 编译并在 WASM runtime 中执行该模块
- **AND** 调用 SHALL 通过 `CodeStorage.callFunc` 进入
- **AND** 系统 MUST NOT 再把 Go plugin 动态编译作为升级对照路径

#### Scenario: 预编译合约实现

- **WHEN** 选择预编译合约对照组
- **THEN** 密码算法 SHALL 在 geth 客户端侧以 native precompile 路径执行
- **AND** 调用接口 SHALL 与其他对照组保持语义等价

## ADDED Requirements

### Requirement: 升级实验上传 WASM 模块

系统 SHALL 让升级延迟、执行效率和升级稳定性实验入口上传 `cryptoupgrade/algorithm/wasm` 中的 WASM 模块，而不是 Go 源码。

#### Scenario: 实验提交 WASM 升级交易

- **WHEN** 基准工具为某候选算法发起升级
- **THEN** 工具 SHALL 读取对应 `.wasm` 文件并作为 `uploadCode` / `uploadCodeVersion` 的 `code` 载荷
- **AND** 完成判据 SHALL 仍为节点 `callFunc` 返回期望输出
- **AND** 工具 MUST NOT 上传 `.go` 源码作为升级产物

#### Scenario: 记录 WASM 身份

- **WHEN** 完成一轮升级或调用基准
- **THEN** 结果 SHALL 记录算法名、版本、`wasmHash` 和 Activation Block
- **AND** 旧 Go plugin JSON/CSV MUST NOT 被写成新正式结果
