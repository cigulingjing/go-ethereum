## Context

技术栈：Go、CGO、gcc/clang、PQMagic-std C 库、ethereum/go-ethereum native precompile。EvoCrypt 仍使用 wazero WASM，本设计不改那条路径。

当前仓库同时存在两条被混用的边界：

```text
EVM
 ├─ cryptoupgrade Precompile（固定地址 0x47–0x5a）
 │     现为纯 Go handler，禁止随机/plugin，尚未接入后量子算法
 └─ EvoCrypt / CodeStorage.callFunc
       上传 WASM → wazero 编译/实例化 → execute(ptr,len)
```

`dynamic-wasm` 把“纯 Go、禁止 CGO”写成了整条 cryptoupgrade 约束。正确范围只是：EvoCrypt 的 runtime 选 wazero，以避免 wasmer/wasmtime 的 CGO。预编译对照路径本来就编译进 Geth，与 secp256k1 一样可以使用 CGO。

`experiments/pqcc` 现有 `libs/*.a` 是 Emscripten wasm 目标文件，不能当作 Linux `.so` 链接进 Geth。必须另开一次原生编译。

## Goals / Non-Goals

**Goals:**

- 把运行时边界拆开：预编译 = 原生 CGO；EvoCrypt = WASM。
- 用 gcc/clang 产出 PQMagic 原生共享库，并通过 CGO 在 precompile handler 中调用。
- 先接入确定性后量子验签（ML-DSA-65 verify），作为 Lab2 对照的 native 入口。
- 保持现有 0x47–0x5a 预编译语义不变。

**Non-Goals:**

- 不把 EvoCrypt 改回 Go plugin / CGO dlopen。
- 不在本 change 把 KEM、SPHINCS-Alpha、Aigis 编进 precompile。
- 不修复 Emscripten wrapper 的 Aigis 验签问题，也不产出浏览器 wasm。
- 不重跑正式实验、不改论文 LaTeX。

## Decisions

### 1. 两条路径分属不同制品，禁止交叉

```text
cryptoupgrade Precompile          EvoCrypt
  编译进 geth                       链上 WASM 字节码
  CGO → native libpqmagic.so        wazero → execute()
  固定地址 CALL                     CodeStorage.callFunc
  升级需发新客户端                  Activation Block 切换版本
```

预编译 SHALL NOT 调用 `wasmruntime`。EvoCrypt SHALL NOT 为了跑 PQMagic 而去 `plugin.Open` 或 CGO 加载同一份 `.so`。

备选：让预编译也加载 WASM，数值上无法再当 Native 对照。否决。

### 2. 预编译用构建期 CGO 链接，而不是运行时 plugin.Open

Geth 已对 secp256k1 使用 CGO。后量子预编译采用同样模式：

```text
experiments/pqcc/native/   # gcc cmake，产出 ELF .so/.a
cryptoupgrade/internal/pqcnative/
    pqmagic.go             # // #cgo LDFLAGS: -lpqmagic
    pqmagic.c / pqmagic.h  # 薄封装：仅 verify（及可选 seeded keygen/sign）
cryptoupgrade/precompile.go
    注册 MlDsa65Verify 地址
```

`#cgo` 在 geth 链接期解析符号。这满足现有规范“预编译不得依赖运行时 plugin 加载”，同时允许 C 算法实现。

备选 A：运行时 `dlopen(libpqmagic.so)`。更像动态升级，但预编译对照应在客户端发布时固定。否决为本 change 默认。

备选 B：Go plugin 包一层 CGO。多一层 `plugin.Open`，且要求 geth 与 plugin 的 Go 版本一致。否决。

### 3. 原生编译与 wasm 编译目录隔离

`experiments/pqcc/pqmagic` 当前 CMake 把 `CMAKE_C_COMPILER` 钉成 `emcc`。原生构建必须使用独立目录和普通 gcc：

```bash
cmake -S experiments/pqcc/pqmagic -B experiments/pqcc/native-build \
  -DCMAKE_C_COMPILER=gcc \
  -DUSE_SM3=OFF -DUSE_SHAKE=ON \
  -DENABLE_KYBER=OFF -DENABLE_ML_KEM=OFF -DENABLE_AIGIS_ENC=OFF \
  -DENABLE_SPHINCS_A=OFF -DENABLE_TEST=OFF -DENABLE_BENCH=OFF \
  -DCMAKE_INSTALL_PREFIX=experiments/pqcc/native-prefix
```

安装产物只给 CGO 使用。现有 wasm 目标文件留在 `libs/`，互不覆盖。

### 4. 第一批只暴露 ML-DSA-65 verify

链上最需要、且不依赖 `randombytes` 的入口是验签。ABI：

```text
MlDsa65Verify(bytes pk, bytes message, bytes signature) → bool
地址：0x5b
```

只调用 `pqmagic_ml_dsa_65_std_verify_internal`。公钥/签名长度必须与 ML-DSA-65 常量一致，否则拒绝输入，不进入 C 层。

keygen/sign 依赖 RNG 或 coins。若后续需要，MUST 把 seed/coins 作为 ABI 输入，不得读 `/dev/urandom`。本 change 不注册这些入口。

备选：一次注册 Dilithium3 / SLH-DSA-SHAKE-192f。会扩大链接体积和测试面。先打通一条 NIST 标准验签。

### 5. CGO 失败必须 fail-closed

`CGO_ENABLED=0` 或缺少原生库时，`MlDsa65Verify` SHALL NOT 静默回退到 WASM 或空实现。构建标签：

- `cgo`：注册真实 handler
- `!cgo`：该地址不注册，或 `Run` 返回明确错误

现有纯 Go 预编译不受影响。

## Risks / Trade-offs

- [Risk] 当前 `libs/*.a` 被误链进 geth，链接器报 wasm 重定位错误。 → Mitigation: 独立 `native-build/`，CGO 只指向 native-prefix。
- [Risk] CGO 使部分 `CGO_ENABLED=0` CI 失败。 → Mitigation: PQC precompile 用 build tag 隔离；默认 Linux 实验镜像保持 CGO。
- [Risk] wrapper 的 `randombytes`/Aigis 缺陷污染 verify。 → Mitigation: verify 只走 `_verify_internal`，不经过现有 `pqcsign_wrapper.c`。
- [Risk] 原生 `.so` 不可移植，不能当 EvoCrypt 升级载荷。 → Mitigation: 这是预编译对照的预期代价；EvoCrypt 仍用 WASM。

## Migration Plan

1. 增加 native CMake 脚本并验证 `file` 输出为 ELF，而不是 `WebAssembly`。
2. 新增 `pqcnative` CGO 包与 ML-DSA-65 verify 单测（已知消息/签名向量）。
3. 注册 `0x5b` 并加入 `core/vm` precompile 映射测试。
4. 确认 `wasmruntime` 与 CodeStorage 路径无引用 `pqcnative`。

回滚：删除 `0x5b` 与 `pqcnative`，不影响已有 0x47–0x5a 和 EvoCrypt。

## Open Questions

- 实验镜像是否预装 `libpqmagic.so`，还是构建 geth 时静态链入 `.a`。倾向静态 `.a`，减少运行时 rpath。
- Dilithium3 / SLH-DSA verify 是否作为本 change 的可选后续任务，还是留给独立 change。
