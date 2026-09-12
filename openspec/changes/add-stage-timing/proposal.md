## Why
当前激活日志缺少 RPC 接收、协处理器执行和主链打包时间，无法关联完整调用与升级链路。
## What Changes
- 新增独立 Logrus JSONL 阶段日志，记录纳秒时间和关联标识。
- 插桩 RPC 接收、协处理器进入/退出、交易主链落块、WASM 升级开始/载入完成。
- 区分请求、交易、执行尝试和升级版本；失败不记录成功就绪。
## Capabilities
### New Capabilities
- `stage-timing`: 节点调用与升级阶段可关联时间日志。
### Modified Capabilities
无。
## Impact
cryptoupgrade 日志模块、RPC、ethapi、EVM 调用边界、主链写入与激活服务；新增 logrus 依赖。
## 非目标
不调整 Gas、共识、算法结果，不重构实验采集脚本，不将 eth_call 视作交易。
