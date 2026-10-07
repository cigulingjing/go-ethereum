# UpgradeConsistencyProbe

```bash
make geth
docker build -f experiments/cryptoupgrade/scripts/docker/Dockerfile.lab -t cryptoupgrade-geth:lab .
# local.yaml 已暴露 8761..8780，并只给 node11..node20 设置实验 v2 准备条件
go run ./experiments/cryptoupgrade/scripts/render -config experiments/cryptoupgrade/config/local.yaml -geth "$(pwd)/build/bin/geth"
cd experiments/cryptoupgrade/output/network/local
docker compose up -d
cd /home/liuqi/project/go-ethereum
mkdir -p experiments/cryptoupgrade/output/results/upgrade-consistency

go run ./experiments/cryptoupgrade/scripts/upgradeconsistency \
  -rpc http://127.0.0.1:8761 -rpc-base 8761 -nodes 20 -activation-gap 100 \
  -call-interval 500ms -log-wait 3m \
  -key "$(cat experiments/cryptoupgrade/config/signer.key)" \
  -v1 experiments/cryptoupgrade/algorithm/go/wasm/consistency/upgrade_consistency_probe_v1.wasm \
  -v2 experiments/cryptoupgrade/algorithm/go/wasm/consistency/upgrade_consistency_probe_v2.wasm \
  -artifact experiments/cryptoupgrade/algorithm/contracts/out/UpgradeConsistencyProbe.sol/UpgradeConsistencyProbe.json \
  -plugin-root experiments/cryptoupgrade/output/network/local \
  -release-file experiments/cryptoupgrade/output/network/local/node20/plugin/release-v2 \
  -out experiments/cryptoupgrade/output/results/upgrade-consistency/run.json
python3 experiments/cryptoupgrade/scripts/upgradeconsistency/analyze.py \
  --input experiments/cryptoupgrade/output/results/upgrade-consistency/run.json \
  --out-dir experiments/cryptoupgrade/output/results/upgrade-consistency/analysis
```

The runner uses RPC only to submit the setup and probe transactions. Probe calls are submitted periodically; it does not poll their receipts. Per-node inclusion, execution, version, output, gas, status, and local state-root data are collected from each node's `plugin/stage_timing.jsonl`. The runner waits up to `-log-wait` for log events, then preserves missing entries as `pending`.

The run writes `run.json`, `preparation.csv`, `analysis/observations.csv`, `analysis/summary.csv`, `analysis/summary.json`, and `analysis/version_heatmap.png` under `experiments/cryptoupgrade/output/results/upgrade-consistency/`.

`GETH_CRYPTOUPGRADE_PREPARE_*` 默认未设置。node20 的屏障由 runner 在覆盖 H−3..H+3 样本后写入挂载的 plugin 目录；它不依赖 node20 先执行到 H 才释放。若 node20 因 `required-version-missing` 拒绝区块，脚本保留 pending/分支数据，分析不会把缺失样本算作一致。
