## Why

`cryptoupgrade` 根包同时承担 Geth/EVM 适配、升级编排、源码编解码、plugin 编译加载及路径持久化，实验网络与 Docker 资产也位于同一目录，导致运行时、算法资产和部署实验边界不清。现有目录初步整理已经完成，需要进一步建立可约束依赖方向的模块边界。

## What Changes

- 将根包整理为稳定 facade，并按 EVM 集成、升级编排、源码制品、编译、运行时加载和持久化拆分内部职责。
- 区分动态候选算法与编译进节点的 builtin 算法。
- 将 benchmark、network、Docker、YAML、文档和结果迁入仓库级 `experiments/cryptoupgrade`。
- 将密码升级 smoke test 与通用网络验证分离，统一源码压缩能力。
- 分阶段迁移引用并增加布局和依赖检查。

## Capabilities

### New Capabilities

- `cryptoupgrade-module-boundaries`: 定义运行时模块、算法资产、实验部署的目录边界、依赖方向和迁移兼容要求。

### Modified Capabilities

- None.

## Impact

影响 `cryptoupgrade` 根包及子目录、Geth 集成引用、实验命令、网络配置、Docker 资产、文档和测试。保留根 package API，并保持 ABI、gas、plugin 数据目录及实验结果格式兼容。

## 非目标

- 不改变密码算法、升级协议、EVM 执行语义、ABI、gas 或共识行为。
- 不删除历史实验结果，不引入新的构建系统或外部依赖。
