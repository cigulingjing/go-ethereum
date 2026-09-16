# cryptoupgrade-performance-benchmarks delta

## ADDED Requirements

### Requirement: 单节点真实交易 Gas 对比基准

系统 SHALL 在单节点 Clique 网络中，对同一算法测量 WASM 动态升级方案与 Solidity 合约方案的真实交易 Gas 消耗，并以交易 receipt 的 `gasUsed` 作为唯一 Gas 数值来源。

#### Scenario: WASM 升级交易 Gas

- **WHEN** 算法通过真实交易调用 `CodeStorage.uploadCode` 上传 WASM 模块
- **THEN** 基准工具 SHALL 等待交易 receipt 并记录 `gasUsed` 作为 WASM 升级交易 Gas
- **AND** receipt 状态必须为成功，否则该算法实验失败

#### Scenario: Solidity 升级交易 Gas

- **WHEN** 等价算法通过真实交易部署 Solidity 合约
- **THEN** 基准工具 SHALL 等待部署交易 receipt 并记录 `gasUsed` 作为 Solidity 升级交易 Gas
- **AND** receipt 状态必须为成功且合约地址非空，否则该算法实验失败

#### Scenario: 执行交易 Gas 与输出一致性

- **WHEN** 两条路径均完成升级或部署
- **THEN** 基准工具 SHALL 先通过 `eth_call` 校验 `CodeStorage.callFunc` 与合约调用的逻辑输出一致
- **AND** 再分别以真实交易调用 `CodeStorage.callFunc` 与合约函数，记录两条执行交易 receipt 的 `gasUsed`
- **AND** 输出不一致时该算法实验失败，不输出部分对比数据

### Requirement: 真实交易 Gas 结果产物

系统 SHALL 输出机器可读 JSON 与文本摘要，覆盖全部选中算法。

#### Scenario: 结果字段完整

- **WHEN** 实验完成
- **THEN** 每个算法的结果 SHALL 包含 WASM 升级交易 gasUsed、Solidity 部署交易 gasUsed、WASM 执行交易 gasUsed、Solidity 执行交易 gasUsed、对应交易哈希与区块号
- **AND** JSON 结果 SHALL 记录网络配置、链 ID、节点规模与运行命令，保证实验可复现

#### Scenario: 算法覆盖

- **WHEN** 以默认算法集运行
- **THEN** 实验 SHALL 覆盖 Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit、SchnorrVerify、PolynomialMul 共 8 个算法
