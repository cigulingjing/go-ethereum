## Why

当前 `CodeStorage.uploadCode` 在交易执行路径中同步调用 `cryptoupgrade.ActivateAlgorithm`，导致升级交易不仅写入链上升级数据，还会直接触发本节点源码解压、Go plugin 编译和本地算法元数据更新。该行为把 EVM 合约调用与节点本地运行时升级强耦合，既破坏链上执行的可重复性，也让多节点升级流程难以统一由事件监听线程控制。

## What Changes

- **BREAKING**: `CodeStorage.uploadCode` 交易执行时不再调用 `ActivateAlgorithm`，只负责校验参数、更新链上可查询的算法信息，并发出 `codeUploaded` event。
- 新增事件驱动激活语义：节点链下专门线程订阅 `codeUploaded` event，解析算法名，调用 `getInfo` 查询链上算法元数据，再执行本地 `ActivateAlgorithm`。
- 保留 `CodeStorage.callFunc` 的调用语义：算法只有在本节点事件监听线程完成激活后才可被调用。
- 更新升级效率相关实验定义，避免继续把 upload receipt 直接等同于算法已激活。

## Capabilities

### New Capabilities

- `cryptoupgrade-event-activation`: 定义 `CodeStorage.uploadCode` 与节点本地算法激活解耦后的事件监听、事件解析、链上元数据读取和本地激活语义。

### Modified Capabilities

- `cryptoupgrade-performance-benchmarks`: 升级方案部署基准需要区分 upload transaction confirmation 与事件监听线程完成本地 activation，不再将 receipt 返回视为激活完成。

## Impact

- `cryptoupgrade/code_storage.go`: 删除 `uploadCode` 分支中直接调用 `ActivateAlgorithm` 的逻辑，保留事件发出和链上信息更新。
- `cryptoupgrade/bind_event.go`: 作为升级激活的唯一运行时入口，需要保证事件解析、`getInfo` 查询、去重和错误日志符合新语义。
- `cmd/geth` 或现有启动绑定点：确保节点启动后事件监听线程持续运行，负责处理 `codeUploaded` event。
- `cryptoupgrade` benchmark 和文档：需要按新流程等待本地 activation 后再验证 `callFunc`，并修正旧的“receipt 即激活”说明。
