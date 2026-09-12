## ADDED Requirements

### Requirement: 多节点 Solidity 合约升级延迟实验

系统 SHALL 提供多节点 Solidity 合约部署/升级延迟实验入口，用于测量等价 Solidity 密码算法合约在多节点网络中的部署完成耗时和部署 Gas 消耗。

#### Scenario: 载入多节点 Solidity 实验配置

- **WHEN** 执行 Solidity 合约升级延迟实验入口并提供多节点 YAML 配置
- **THEN** 实验入口 SHALL 载入归一化网络配置、节点角色、RPC URL、chain ID 和 consensus period
- **AND** SHALL 支持选择 sender 节点和参与观测的目标节点
- **AND** SHALL 在没有显式 sender 时选择第一个 signer 节点作为部署交易发送节点

#### Scenario: 执行多算法 Solidity 部署实验

- **WHEN** 用户选择一个或多个 Solidity 密码算法 fixture
- **THEN** 实验入口 SHALL 为每个算法编译对应 Solidity 合约并提交一笔合约部署交易
- **AND** SHALL 至少支持 Add、Sha256、Blake2bSum256、Pbkdf2Sha256、Dh2048Secret、PedersenCommit 和 SchnorrVerify 中可在当前环境成功编译和验证的算法
- **AND** 每个算法 SHALL 保留独立的部署交易 hash、合约地址、receipt status、receipt gas used 和节点级观测结果

#### Scenario: 观测所有目标节点的部署完成

- **WHEN** Solidity 合约部署交易 hash 已返回
- **THEN** 实验入口 SHALL 并发轮询每个目标节点的部署交易 receipt
- **AND** SHALL 在每个节点首次返回成功 receipt 后通过该节点执行一次等价合约调用校验
- **AND** 只有 receipt 成功且校验调用返回期望输出时，该节点 SHALL 被标记为 completed
- **AND** 全网升级完成耗时 SHALL 取最晚 completed 节点时间减去部署交易提交开始时间

#### Scenario: 输出可合并的 Solidity 原始数据

- **WHEN** Solidity 合约升级延迟实验完成或失败
- **THEN** 实验入口 SHALL 输出 round-level JSON/CSV 和 node-level JSON/CSV
- **AND** round-level CSV SHALL 包含算法、合约源码、ABI 输入输出类型、部署交易 hash、合约地址、部署 gas used、目标节点数、完成节点数和 `submit_to_all_complete_ms`
- **AND** node-level CSV SHALL 包含每个节点的 receipt 可见时间、校验调用耗时、submit-to-receipt、receipt-to-complete 和 submit-to-complete 耗时
