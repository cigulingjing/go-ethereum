# pqcgo

Solidity 不适合对照的后量子验签归档。每个算法一个目录：

```text
<pqcgo>/<algorithm>/
  native/           # CGO 使用的源码快照与 libverify.a
  <algorithm>.wasm
  testdata/         # pk.bin message.bin signature.bin
```

重建：

```bash
experiments/cryptoupgrade/algorithm/pqcgo/build.sh
go run ./experiments/pqc_verify
```

四个算法也可一次跑完：`go run ./experiments/cryptoupgrade/bench/cmd/benchpqccgowasm`。
