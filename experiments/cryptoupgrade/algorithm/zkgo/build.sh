#!/bin/bash
# 快照 pairing 后端源码，生成 testdata，并编译 WASI wasm。
# ZKGO_ALGOS 默认同时构建 BLS12-381 与 BN254。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
ZKGO="$(cd "$(dirname "$0")" && pwd)"
HARNESS="$ZKGO/_harness"
WASI_SDK_PATH="${WASI_SDK_PATH:-$HOME/.local/opt/wasi-sdk/wasi-sdk-24.0-x86_64-linux}"
BLST_VER="${BLST_VER:-v0.3.16}"
BLST_CFLAGS="-O3 -std=gnu11 -D__BLST_NO_ASM__ -D__BLST_NO_CPUID__"
MCL_SRC="${MCL_SRC:-/tmp/mcl}"
MCL_CXXFLAGS="-O3 -std=c++14 -DNDEBUG -DMCL_DONT_USE_XBYAK -DMCL_BINT_ASM=0 -DMCL_MSM=0 -DMCL_FP_BIT=256 -DMCL_FR_BIT=256"
MCL_CFLAGS="-O3 -std=gnu11"
WASM_CXXFLAGS="-O3 -std=c++14 -fno-exceptions -fno-rtti -fno-threadsafe-statics -DNDEBUG -DMCL_BINT_ASM=0 -DMCL_MSM=0 -DMCL_FP_BIT=256 -DMCL_FR_BIT=256 -DCYBOZU_DONT_USE_EXCEPTION -DCYBOZU_DONT_USE_STRING"
ALGOS="${ZKGO_ALGOS:-groth16_bls12381_verify groth16_bn254_verify}"

if [ ! -x "$WASI_SDK_PATH/bin/clang" ]; then
  echo "wasi-sdk clang not found at $WASI_SDK_PATH/bin/clang" >&2
  exit 1
fi

need_mcl() {
  case " $ALGOS " in
    *" groth16_bn254_verify "*) return 0 ;;
    *) return 1 ;;
  esac
}

ensure_mcl() {
  if [ -f "$MCL_SRC/src/fp.cpp" ] && [ -f "$MCL_SRC/include/mcl/bn_c256.h" ]; then
    return
  fi
  echo "cloning herumi/mcl into $MCL_SRC"
  git clone --depth 1 https://github.com/herumi/mcl.git "$MCL_SRC"
}

snapshot_blst() {
  local algo_dir="$ZKGO/groth16_bls12381_verify"
  local native="$algo_dir/native"
  local blst
  blst="$(go env GOPATH)/pkg/mod/github.com/supranational/blst@${BLST_VER}"
  if [ ! -f "$blst/src/server.c" ]; then
    echo "blst $BLST_VER not found; downloading module" >&2
    (cd "$ROOT" && go mod download github.com/supranational/blst@${BLST_VER})
  fi
  if [ ! -f "$blst/src/server.c" ]; then
    echo "missing $blst/src/server.c" >&2
    exit 1
  fi
  mkdir -p "$native/include" "$native/src"
  cp -a "$blst/bindings/blst.h" "$native/include/"
  cp -a "$blst/bindings/blst_aux.h" "$native/include/"
  find "$blst/src" -maxdepth 1 -type f \( -name '*.c' -o -name '*.h' -o -name '*.hpp' \) -exec cp -a {} "$native/src/" \;
  cat > "$algo_dir/SOURCE.txt" <<EOF
algorithm=groth16_bls12381_verify
curve=bls12-381
extracted_from=experiments/gnark/backend/groth16/bls12-381/verify.go
pairing=e(Ar,Bs)*e(Krs,-delta)*e(L_pub,-gamma)*e(alpha,-beta)=1
blst=$BLST_VER
flags=$BLST_CFLAGS
command=experiments/cryptoupgrade/algorithm/zkgo/build.sh
EOF
}

snapshot_mcl() {
  local algo_dir="$ZKGO/groth16_bn254_verify"
  local native="$algo_dir/native"
  mkdir -p "$native/include" "$native/src"
  cp -a "$MCL_SRC/include/mcl" "$native/include/"
  cp -a "$MCL_SRC/include/cybozu" "$native/include/"
  cp -a "$MCL_SRC/src/fp.cpp" "$native/src/"
  find "$MCL_SRC/src" -maxdepth 1 -type f \( -name '*.hpp' -o -name '*.h' \) -exec cp -a {} "$native/src/" \;
  cp -a "$MCL_SRC/src/xbyak" "$native/src/"
  cat > "$algo_dir/SOURCE.txt" <<EOF
algorithm=groth16_bn254_verify
curve=bn254
extracted_from=experiments/gnark/backend/groth16/bn254/verify.go
pairing=e(Ar,Bs)*e(Krs,-delta)*e(L_pub,-gamma)*e(alpha,-beta)=1
mcl=MCL_BN_SNARK1
flags=$MCL_CXXFLAGS
command=experiments/cryptoupgrade/algorithm/zkgo/build.sh
EOF
}

build_vectors() {
  local algo="$1"
  local algo_dir="$ZKGO/$algo"
  local out="$algo_dir/testdata"
  mkdir -p "$out"
  if [ -f "$out/vk.bin" ] && [ -f "$out/proof.bin" ] && [ -f "$out/public.bin" ] && [ "${FORCE_VECTORS:-}" != "1" ]; then
    echo "reusing vectors $out"
    return
  fi
  echo "generating testdata for $algo"
  (cd "$algo_dir/gen" && go run . "$out")
}

build_wasm_bls() {
  local algo_dir="$ZKGO/groth16_bls12381_verify"
  local native="$algo_dir/native"
  local out="$algo_dir/groth16_bls12381_verify.wasm"
  "$WASI_SDK_PATH/bin/clang" \
    --target=wasm32-wasi \
    --sysroot="$WASI_SDK_PATH/share/wasi-sysroot" \
    $BLST_CFLAGS \
    -DGROTH16_VERIFY_FN=groth16_bls12381_verify \
    -I"$native/include" \
    -I"$native/src" \
    -mexec-model=reactor \
    -Wl,--no-entry \
    -Wl,--export=execute \
    -Wl,--export-memory \
    -Wl,--initial-memory=16777216 \
    -Wl,--max-memory=16777216 \
    -Wl,-z,stack-size=2097152 \
    "$native/src/server.c" \
    "$native/verify.c" \
    "$HARNESS/execute.c" \
    -o "$out"
  echo "wrote $out ($(wc -c < "$out") bytes)"
}

host_check_bn254() {
  local algo_dir="$ZKGO/groth16_bn254_verify"
  local native="$algo_dir/native"
  local testdata="$algo_dir/testdata"
  local work
  work="$(mktemp -d)"
  gcc $MCL_CFLAGS -c "$native/verify.c" -I"$native/include" -o "$work/verify.o"
  gcc $MCL_CFLAGS -c "$native/check.c" -o "$work/check.o"
  g++ $MCL_CXXFLAGS -c "$native/src/fp.cpp" -I"$native/include" -I"$native/src" -o "$work/fp.o"
  g++ -o "$work/check" "$work/fp.o" "$work/verify.o" "$work/check.o" -lstdc++
  "$work/check" "$testdata/vk.bin" "$testdata/public.bin" "$testdata/proof.bin"
  rm -rf "$work"
}

build_wasm_bn254() {
  local algo_dir="$ZKGO/groth16_bn254_verify"
  local native="$algo_dir/native"
  local out="$algo_dir/groth16_bn254_verify.wasm"
  local work
  work="$(mktemp -d)"
  "$WASI_SDK_PATH/bin/clang++" \
    --target=wasm32-wasi \
    --sysroot="$WASI_SDK_PATH/share/wasi-sysroot" \
    $WASM_CXXFLAGS \
    -I"$native/include" \
    -I"$native/src" \
    -c "$native/src/fp.cpp" \
    -o "$work/fp.o"
  "$WASI_SDK_PATH/bin/clang" \
    --target=wasm32-wasi \
    --sysroot="$WASI_SDK_PATH/share/wasi-sysroot" \
    $MCL_CFLAGS \
    -I"$native/include" \
    -c "$native/verify.c" \
    -o "$work/verify.o"
  "$WASI_SDK_PATH/bin/clang" \
    --target=wasm32-wasi \
    --sysroot="$WASI_SDK_PATH/share/wasi-sysroot" \
    $MCL_CFLAGS \
    -DGROTH16_VERIFY_FN=groth16_bn254_verify \
    -c "$HARNESS/execute.c" \
    -o "$work/execute.o"
  "$WASI_SDK_PATH/bin/clang++" \
    --target=wasm32-wasi \
    --sysroot="$WASI_SDK_PATH/share/wasi-sysroot" \
    $WASM_CXXFLAGS \
    -mexec-model=reactor \
    -Wl,--no-entry \
    -Wl,--export=execute \
    -Wl,--export-memory \
    -Wl,--initial-memory=16777216 \
    -Wl,--max-memory=16777216 \
    -Wl,-z,stack-size=2097152 \
    "$work/fp.o" "$work/verify.o" "$work/execute.o" \
    -o "$out"
  rm -rf "$work"
  echo "wrote $out ($(wc -c < "$out") bytes)"
}

if need_mcl; then
  ensure_mcl
fi

for algo in $ALGOS; do
  echo "=== $algo ==="
  case "$algo" in
    groth16_bls12381_verify)
      echo "snapshot blst"
      snapshot_blst
      echo "vectors"
      build_vectors "$algo"
      echo "wasm"
      build_wasm_bls
      ;;
    groth16_bn254_verify)
      echo "snapshot mcl"
      snapshot_mcl
      echo "vectors"
      build_vectors "$algo"
      echo "host check"
      host_check_bn254
      echo "wasm"
      build_wasm_bn254
      ;;
    *)
      echo "unknown algorithm $algo" >&2
      exit 1
      ;;
  esac
done

echo "zkgo archive ready"
