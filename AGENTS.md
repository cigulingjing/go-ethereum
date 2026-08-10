# AGENTS.md

## 1. 项目说明

本项目用于研究 **Ethereum/Geth 中密码学算法的动态升级机制**。

当前 Geth 对密码学算法的扩展主要依赖客户端源码修改或 Precompiled Contract。传统方式通常需要重新编译、部署并重启节点，升级成本较高，并可能造成服务中断。

本项目尝试设计一种 **无需停止节点即可升级密码学算法实现** 的机制，使密码学算法能够以模块化方式被 Geth 加载、调用和替换。

项目重点关注以下问题：

* 密码学算法的模块化实现与动态加载；
* Geth/EVM 与外部密码学实现之间的调用机制；
* 密码学算法版本管理与升级流程；
* 升级过程中的兼容性与安全性；
* Solidity、Precompiled Contract 与动态升级方案之间的性能差异；
* 不同方案在执行时间、Gas 消耗、升级时间和升级成本方面的对比。

### 开发原则

在修改代码时，应优先保证：

1. **最小侵入性**：尽量减少对 Geth 原有代码的修改。
2. **模块化**：新增功能应具有清晰的模块边界，避免与现有逻辑强耦合。
3. **兼容性**：除明确需要修改的行为外，不应改变 Geth 原有执行语义。
4. **可测试性**：新增功能应尽可能提供独立测试入口。
5. **实验可复现性**：涉及性能测试、升级测试的代码应保证测试过程可以重复执行。
6. **避免无关修改**：执行任务时仅修改与当前需求直接相关的代码，不进行无必要的重构。

---

## 2. 代码注释规则

代码注释应解释 **为什么这样设计**，而不是简单重复代码本身。

### 2.1 注释语言

* 普通说明使用中文。
* 技术名词、协议名、标准名、函数名、结构体名等保留英文。
* 不强制翻译已有的 Geth 英文注释。
* 修改 Geth 原有代码时，优先保持原文件已有的注释风格。

例如：

```go
// 加载指定版本的 crypto module，避免升级过程中修改正在执行的实例。
module := manager.Load(version)
```

不推荐：

```go
// 加载 module
module := manager.Load(version)
```

### 2.2 Go 注释规范

遵循 Go 官方注释习惯。

对于导出的 `function`、`struct`、`interface`、`const` 和 `variable`，注释应以对应标识符名称开头。

推荐：

```go
// CryptoManager manages dynamically loaded cryptographic modules.
type CryptoManager struct {
    ...
}
```

推荐：

```go
// LoadModule loads the cryptographic module for the specified version.
func LoadModule(version uint64) error {
    ...
}
```

不要写：

```go
// 这个函数负责加载密码模块。
func LoadModule(version uint64) error {
    ...
}
```

### 2.3 必须添加注释的场景

以下代码应添加必要注释：

* 核心架构设计；
* Geth 原有执行流程中的修改点；
* EVM 与密码学模块之间的调用边界；
* 动态加载、升级、版本切换相关逻辑；
* 并发控制；
* 缓存策略；
* 异常恢复机制；
* 不容易理解的状态变化；
* 为兼容 Geth 原有行为而增加的特殊处理；
* 与实验设计直接相关的测量或统计逻辑。

例如：

```go
// 使用读锁保证正在执行的交易继续使用当前 module。
// Upgrade 仅替换后续调用使用的 module，避免升级影响正在执行的交易。
m.mu.RLock()
module := m.current
m.mu.RUnlock()
```

### 2.4 不需要添加注释的场景

对于代码本身已经足够清晰的逻辑，不添加冗余注释。

不推荐：

```go
// version 加 1
version++

// 判断 err 是否不为空
if err != nil {
    return err
}
```

### 2.5 修改原有 Geth 代码时

如果修改 Geth 原有代码：

* 不删除原有有效注释；
* 尽量保持原有英文注释；
* 新增的重要逻辑可以使用中文说明；
* 对关键侵入点说明修改目的。

例如：

```go
// Dynamic crypto extension:
// 在执行标准 Precompiled Contract 前检查是否存在动态注册的实现。
// 未注册时保持 Geth 原有执行路径。
if contract := dynamicRegistry.Get(addr); contract != nil {
    return contract.Run(input)
}
```

### 2.6 TODO 注释

暂未实现但明确需要后续处理的内容统一使用：

```go
// TODO: 支持 module version rollback。
```

如果与实验有关，应说明实验目的：

```go
// TODO: 记录 module load latency，用于升级延迟实验。
```

避免没有上下文的信息：

```go
// TODO: fix
```

### 2.7 注释基本原则

生成或修改代码时遵循以下原则：

> **代码说明 What，注释解释 Why。**

如果注释只是把代码翻译成自然语言，则优先删除该注释。

对于复杂逻辑，应优先通过：

* 合理的函数拆分；
* 清晰的变量命名；
* 明确的数据结构；

提高代码可读性，而不是依赖大量注释解释复杂代码。

## 3. OpenSpec 编程规范

本项目使用 **OpenSpec** 管理功能设计、代码修改和实验任务。

涉及新增功能、架构调整、较大范围代码修改或实验能力扩展时，应优先通过 OpenSpec 创建 change，再根据 change 中的设计和任务进行实现。

### 3.1 基本工作流程

推荐遵循以下流程：

```text
需求
  ↓
创建 change
  ↓
proposal.md
  ↓
design.md
  ↓
tasks.md
  ↓
确认设计
  ↓
代码实现
  ↓
测试验证
  ↓
完成 change
```

Coding Agent 在开始较大开发任务前，应优先检查：

```text
openspec/
├── specs/
└── changes/
```

其中：

* `specs/`：描述项目当前已经确认的能力和系统行为；
* `changes/`：描述正在设计、开发或验证中的变更；
* `proposal.md`：说明为什么进行该变更以及需要解决什么问题；
* `design.md`：说明技术方案、模块关系和关键设计决策；
* `tasks.md`：将设计拆分为可以实际执行的开发任务。

代码实现应尽量与 `proposal.md`、`design.md` 和 `tasks.md` 保持一致。

如果实际实现过程中发现原设计存在问题，应优先更新对应 OpenSpec 文档，而不是直接偏离设计继续实现。

---

### 3.2 Change 命名规范

OpenSpec change 名称必须做到：

* 简短；
* 明确；
* 稳定；
* 能够直接表达变更目标；
* 不包含无必要的实现细节。

统一采用：

```text
<action>-<capability>
```

必要时可以采用：

```text
<action>-<capability>-<scope>
```

原则上控制在：

```text
2 ～ 4 个单词
```

使用：

```text
lowercase-kebab-case
```

例如：

```text
add-crypto-upgrade
add-module-loader
add-version-manager
add-upgrade-benchmark
optimize-module-call
fix-module-cache
refactor-crypto-registry
```

---

### 3.3 Action 命名

优先使用以下 action：

```text
add
fix
update
remove
refactor
optimize
```

含义分别为：

```text
add       新增能力
fix       修复已有问题
update    修改已有行为
remove    删除已有能力
refactor  重构实现但原则上不改变外部行为
optimize  性能或资源消耗优化
```

不要随意创造含义接近的新动词。

例如，不推荐：

```text
implement-module-loader
create-module-loader
support-module-loader
develop-module-loader
```

统一使用：

```text
add-module-loader
```

---

### 3.4 Capability 命名

`capability` 应描述系统提供的核心能力，而不是具体实现技术。

推荐：

```text
crypto-upgrade
module-loader
version-manager
crypto-registry
upgrade-benchmark
module-call
rollback
```

不推荐：

```text
go-plugin-loader
geth-dlopen-loader
grpc-crypto-module
solidity-crypto-test
```

除非某种具体技术本身就是研究对象，否则不要将：

* 编程语言；
* 框架名称；
* 第三方库；
* 临时实现方式；

写入 change 名称。

---

### 3.5 禁止过长命名

禁止将需求描述直接转换为 change 名称。

例如，不推荐：

```text
add-dynamic-crypto-module-loading-for-geth
add-go-plugin-based-cryptographic-algorithm-upgrade
add-geth-evm-dynamic-precompile-contract-manager
```

推荐分别简化为：

```text
add-module-loader
add-crypto-upgrade
add-precompile-manager
```

Change 名称只用于标识一个变更主题。

具体：

* 使用什么技术；
* 修改哪些模块；
* 为什么这样设计；
* 如何实现；

应写入 `proposal.md` 或 `design.md`，而不是塞入 change 名称。

---

### 3.6 一个 Change 只解决一个核心问题

一个 change 应具有明确的单一目标。

例如：

```text
add-module-loader
```

可以包含：

* module 加载；
* module 生命周期；
* module 缓存；
* 必要的错误处理。

但如果同时需要增加：

* module 动态加载；
* 版本管理；
* 升级性能实验；
* rollback 机制；

应根据实际独立性拆分为：

```text
add-module-loader
add-version-manager
add-upgrade-benchmark
add-upgrade-rollback
```

避免：

```text
add-module-loader-version-manager-benchmark-rollback
```

---

### 3.7 Change 名称禁止包含的信息

除非确有必要，change 名称中禁止包含：

```text
具体文件名
具体函数名
具体 struct 名
开发人员姓名
日期
版本号
实验编号
无意义技术栈堆叠
```

不推荐：

```text
modify-contract.go
fix-loadModule-function
add-crypto-v2
add-test-0808
add-go-geth-plugin-module
```

---

### 3.8 实验类 Change

论文实验相关任务统一使用：

```text
add-<experiment>
```

或：

```text
benchmark-<target>
```

推荐：

```text
add-call-benchmark
add-upgrade-benchmark
benchmark-crypto-call
benchmark-upgrade-latency
```

实验 change 应在设计中明确：

* 实验目的；
* 自变量；
* 对比方案；
* 测量指标；
* 测量方法；
* 输出数据格式；
* 实验环境；
* 重复次数。

禁止只实现 benchmark 代码而不说明实验指标和实验目的。

---

### 3.9 Coding Agent 使用 OpenSpec 的规则

Coding Agent 执行开发任务时应遵循：

1. 如果用户明确指定已有 change，优先读取该 change，再开始修改代码。
2. 不得在没有必要的情况下创建新的 change。
3. 不得为同一个需求重复创建多个语义相同的 change。
4. 创建 change 前，应检查 `openspec/changes/` 是否已经存在对应任务。
5. change 名称必须遵守本节命名规范。
6. 禁止自动生成冗长的 change 名称。
7. `proposal.md` 重点描述 **Why / What**。
8. `design.md` 重点描述 **How**。
9. `tasks.md` 必须拆分为可执行、可验证的任务。
10. 代码实现完成后，应根据实际完成情况更新 `tasks.md`。
11. 如果实现与设计发生明显偏差，应同步修改 OpenSpec，而不是让文档与代码长期不一致。
12. 不为了形式而创建 OpenSpec；简单 bug、注释修改、格式调整等小型修改可以直接完成。

---

### 3.10 Change 命名判断原则

创建 change 名称时，首先回答：

> **这次修改最终为系统增加、修改或修复了什么能力？**

然后使用该能力作为名称主体。

例如：

```text
需求：
使用 Go Plugin 实现密码算法动态加载。

关注点：
Go Plugin 是实现方式，
动态加载密码算法才是系统能力。

推荐：
add-module-loader

不推荐：
add-go-plugin-crypto-loader
```

再例如：

```text
需求：
为密码算法增加版本选择和版本切换机制。

推荐：
add-version-manager

不推荐：
add-crypto-module-version-switching-mechanism
```

因此，change 命名遵循：

> **命名表达能力，Design 表达实现。**

这是本项目 OpenSpec change 命名的核心原则。
