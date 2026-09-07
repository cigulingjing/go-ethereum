# Experiment results

新实验默认写入本目录下的 `<experiment>/<run-id>`。

历史结果继续只读保留在 `cryptoupgrade/results`；该历史目录中的 JSON、CSV、日志、图表、
生成源码、绝对路径和测量值均不迁移、不重写。

dynamic-wasm 之后的新正式结果必须在 JSON 中记录 `wasmHash` 和 `activationBlock`。
缺少这些字段的旧 Go plugin 结果目录只作为历史对照保留，不作为 WASM 路径的正式数据引用。
