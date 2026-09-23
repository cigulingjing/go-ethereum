#!/bin/bash
# 将 PQMagic 源码快照到 pqcgo/<algorithm>/native/，并生成 testdata 与 WASI wasm。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
PQMAGIC="$ROOT/experiments/pqcc/pqmagic"
HARNESS="$(cd "$(dirname "$0")/_harness" && pwd)"
PQCGo="$(cd "$(dirname "$0")" && pwd)"
BUILD="${PQC_BUILD_DIR:-$PQCGo/_build}"
WASI_SDK_PATH="${WASI_SDK_PATH:-$HOME/.local/opt/wasi-sdk/wasi-sdk-24.0-x86_64-linux}"

ALGOS="${PQC_ALGOS:-aigis_sig2_verify dilithium3_verify ml_dsa_65_verify slh_dsa_shake_192f_verify}"

if [ ! -x "$WASI_SDK_PATH/bin/clang" ]; then
  echo "wasi-sdk clang not found at $WASI_SDK_PATH/bin/clang" >&2
  exit 1
fi

mkdir -p "$BUILD"

copy_files() {
  local dest="$1"
  shift
  mkdir -p "$dest"
  local src
  for src in "$@"; do
    if [ -f "$src" ]; then
      cp -a "$src" "$dest/"
    elif [ -d "$src" ]; then
      mkdir -p "$dest/$(basename "$src")"
      cp -a "$src/." "$dest/$(basename "$src")/"
    else
      echo "missing source $src" >&2
      exit 1
    fi
  done
}

write_pqmagic_config() {
  local dest="$1"
  local ns="$2"
  mkdir -p "$dest/include"
  cat > "$dest/include/pqmagic_config.h" <<EOF
#ifndef PQMAGIC_CONFIG_H
#define PQMAGIC_CONFIG_H
#define GLOBAL_NAMESPACE(s) ${ns}_##s
#endif
EOF
}

# 每个算法的符号加前缀，避免四个实现链进同一进程时 randombytes 冲突。
native_cflags() {
  local ns="$1"
  echo -n "-O3 -std=gnu11 -DUSE_SHAKE -Drandombytes=${ns}_randombytes"
}

snapshot_aigis() {
  local dest="$PQCGo/aigis_sig2_verify/native"
  rm -rf "$dest"
  copy_files "$dest" \
    "$PQMAGIC/sig/aigis-sig/std/ntt.c" \
    "$PQMAGIC/sig/aigis-sig/std/packing.c" \
    "$PQMAGIC/sig/aigis-sig/std/poly.c" \
    "$PQMAGIC/sig/aigis-sig/std/polyvec.c" \
    "$PQMAGIC/sig/aigis-sig/std/reduce.c" \
    "$PQMAGIC/sig/aigis-sig/std/rounding.c" \
    "$PQMAGIC/sig/aigis-sig/std/sign.c" \
    "$PQMAGIC/sig/aigis-sig/std/config.h" \
    "$PQMAGIC/sig/aigis-sig/std/params.h" \
    "$PQMAGIC/sig/aigis-sig/std/api.h" \
    "$PQMAGIC/sig/aigis-sig/std/sign.h" \
    "$PQMAGIC/sig/aigis-sig/std/packing.h" \
    "$PQMAGIC/sig/aigis-sig/std/poly.h" \
    "$PQMAGIC/sig/aigis-sig/std/polyvec.h" \
    "$PQMAGIC/sig/aigis-sig/std/ntt.h" \
    "$PQMAGIC/sig/aigis-sig/std/reduce.h" \
    "$PQMAGIC/sig/aigis-sig/std/rounding.h" \
    "$PQMAGIC/hash/keccak/fips202.c" \
    "$PQMAGIC/hash/keccak/fips202.h" \
    "$PQMAGIC/utils/randombytes.c" \
    "$PQMAGIC/utils/randombytes.h"
  mkdir -p "$dest/utils" "$dest/hash/keccak" "$dest/include"
  cp -a "$PQMAGIC/utils/randombytes.h" "$dest/utils/"
  cp -a "$PQMAGIC/hash/keccak/fips202.h" "$dest/hash/keccak/"
  write_pqmagic_config "$dest" aigis_sig2_verify
  cp -a "$HARNESS/verify.c" "$dest/"
}

snapshot_dilithium() {
  local dest="$PQCGo/dilithium3_verify/native"
  rm -rf "$dest"
  copy_files "$dest" \
    "$PQMAGIC/sig/dilithium/std/sign.c" \
    "$PQMAGIC/sig/dilithium/std/packing.c" \
    "$PQMAGIC/sig/dilithium/std/polyvec.c" \
    "$PQMAGIC/sig/dilithium/std/poly.c" \
    "$PQMAGIC/sig/dilithium/std/ntt.c" \
    "$PQMAGIC/sig/dilithium/std/reduce.c" \
    "$PQMAGIC/sig/dilithium/std/rounding.c" \
    "$PQMAGIC/sig/dilithium/std/symmetric-shake.c" \
    "$PQMAGIC/sig/dilithium/std/config.h" \
    "$PQMAGIC/sig/dilithium/std/params.h" \
    "$PQMAGIC/sig/dilithium/std/api.h" \
    "$PQMAGIC/sig/dilithium/std/sign.h" \
    "$PQMAGIC/sig/dilithium/std/packing.h" \
    "$PQMAGIC/sig/dilithium/std/poly.h" \
    "$PQMAGIC/sig/dilithium/std/polyvec.h" \
    "$PQMAGIC/sig/dilithium/std/ntt.h" \
    "$PQMAGIC/sig/dilithium/std/reduce.h" \
    "$PQMAGIC/sig/dilithium/std/rounding.h" \
    "$PQMAGIC/sig/dilithium/std/symmetric.h" \
    "$PQMAGIC/hash/keccak/fips202.c" \
    "$PQMAGIC/hash/keccak/fips202.h" \
    "$PQMAGIC/utils/randombytes.c" \
    "$PQMAGIC/utils/randombytes.h"
  mkdir -p "$dest/utils" "$dest/hash/keccak" "$dest/include"
  cp -a "$PQMAGIC/utils/randombytes.h" "$dest/utils/"
  cp -a "$PQMAGIC/hash/keccak/fips202.h" "$dest/hash/keccak/"
  write_pqmagic_config "$dest" dilithium3_verify
  cp -a "$HARNESS/verify.c" "$dest/"
}

snapshot_mldsa() {
  local dest="$PQCGo/ml_dsa_65_verify/native"
  rm -rf "$dest"
  copy_files "$dest" \
    "$PQMAGIC/sig/ml_dsa/std/sign.c" \
    "$PQMAGIC/sig/ml_dsa/std/packing.c" \
    "$PQMAGIC/sig/ml_dsa/std/polyvec.c" \
    "$PQMAGIC/sig/ml_dsa/std/poly.c" \
    "$PQMAGIC/sig/ml_dsa/std/ntt.c" \
    "$PQMAGIC/sig/ml_dsa/std/reduce.c" \
    "$PQMAGIC/sig/ml_dsa/std/rounding.c" \
    "$PQMAGIC/sig/ml_dsa/std/symmetric-shake.c" \
    "$PQMAGIC/sig/ml_dsa/std/config.h" \
    "$PQMAGIC/sig/ml_dsa/std/params.h" \
    "$PQMAGIC/sig/ml_dsa/std/api.h" \
    "$PQMAGIC/sig/ml_dsa/std/sign.h" \
    "$PQMAGIC/sig/ml_dsa/std/packing.h" \
    "$PQMAGIC/sig/ml_dsa/std/poly.h" \
    "$PQMAGIC/sig/ml_dsa/std/polyvec.h" \
    "$PQMAGIC/sig/ml_dsa/std/ntt.h" \
    "$PQMAGIC/sig/ml_dsa/std/reduce.h" \
    "$PQMAGIC/sig/ml_dsa/std/rounding.h" \
    "$PQMAGIC/sig/ml_dsa/std/symmetric.h" \
    "$PQMAGIC/hash/keccak/fips202.c" \
    "$PQMAGIC/hash/keccak/fips202.h" \
    "$PQMAGIC/utils/randombytes.c" \
    "$PQMAGIC/utils/randombytes.h"
  mkdir -p "$dest/utils" "$dest/hash/keccak" "$dest/include"
  cp -a "$PQMAGIC/utils/randombytes.h" "$dest/utils/"
  cp -a "$PQMAGIC/hash/keccak/fips202.h" "$dest/hash/keccak/"
  write_pqmagic_config "$dest" ml_dsa_65_verify
  cp -a "$HARNESS/verify.c" "$dest/"
}

snapshot_slh() {
  local dest="$PQCGo/slh_dsa_shake_192f_verify/native"
  rm -rf "$dest"
  copy_files "$dest" \
    "$PQMAGIC/sig/slh_dsa/std/address.c" \
    "$PQMAGIC/sig/slh_dsa/std/fors.c" \
    "$PQMAGIC/sig/slh_dsa/std/merkle.c" \
    "$PQMAGIC/sig/slh_dsa/std/sign.c" \
    "$PQMAGIC/sig/slh_dsa/std/utils.c" \
    "$PQMAGIC/sig/slh_dsa/std/utilsx1.c" \
    "$PQMAGIC/sig/slh_dsa/std/wots.c" \
    "$PQMAGIC/sig/slh_dsa/std/wotsx1.c" \
    "$PQMAGIC/sig/slh_dsa/std/hash_shake.c" \
    "$PQMAGIC/sig/slh_dsa/std/thash_shake_simple.c" \
    "$PQMAGIC/sig/slh_dsa/std/api.h" \
    "$PQMAGIC/sig/slh_dsa/std/params.h" \
    "$PQMAGIC/sig/slh_dsa/std/config.h" \
    "$PQMAGIC/hash/keccak/fips202.c" \
    "$PQMAGIC/hash/keccak/fips202.h" \
    "$PQMAGIC/utils/randombytes.c" \
    "$PQMAGIC/utils/randombytes.h"
  mkdir -p "$dest/params" "$dest/utils" "$dest/hash/keccak" "$dest/include"
  cp -a "$PQMAGIC/sig/slh_dsa/std/params/params-slh-dsa-shake-192f.h" "$dest/params/"
  write_pqmagic_config "$dest" slh_dsa_shake_192f_verify
  local h
  for h in "$PQMAGIC/sig/slh_dsa/std/"*.h; do
    cp -a "$h" "$dest/"
  done
  cp -a "$PQMAGIC/utils/randombytes.h" "$dest/utils/"
  cp -a "$PQMAGIC/hash/keccak/fips202.h" "$dest/hash/keccak/"
  cp -a "$HARNESS/slh_cgo_mode.h" "$dest/"
  cp -a "$HARNESS/verify.c" "$dest/"
}

algo_defs() {
  local algo="$1"
  case "$algo" in
    aigis_sig2_verify)
      echo "-DAIGIS_SIG_MODE=2 -DPQC_HAS_CTX=1 -DPQC_VERIFY_FN=aigis_sig2_verify -DPQC_ALG_NAME=aigis_sig2_verify"
      ;;
    dilithium3_verify)
      echo "-DDILITHIUM_MODE=3 -DPQC_VERIFY_FN=dilithium3_verify -DPQC_ALG_NAME=dilithium3_verify"
      ;;
    ml_dsa_65_verify)
      echo "-DML_DSA_MODE=65 -DPQC_HAS_CTX=1 -DPQC_VERIFY_FN=ml_dsa_65_verify -DPQC_ALG_NAME=ml_dsa_65_verify"
      ;;
    slh_dsa_shake_192f_verify)
      echo "-DSLH_DSA_MODE=slh-dsa-shake-192f -DTHASH=simple -DSLH_DSA_HASH_MODE_NAMESPACE=shake_192f -DPQC_USE_API_H=1 -DPQC_VERIFY_FN=slh_dsa_shake_192f_verify -DPQC_ALG_NAME=slh_dsa_shake_192f_verify"
      ;;
    *)
      echo "unknown algorithm $algo" >&2
      exit 1
      ;;
  esac
}

includes_for() {
  local dest="$1"
  echo -n "-I$dest -I$dest/utils -I$dest/hash/keccak -I$PQMAGIC -I$PQMAGIC/include -I$PQMAGIC/utils"
}

build_native_lib() {
  local algo="$1"
  local dest="$PQCGo/$algo/native"
  local objdir="$BUILD/$algo/native-obj"
  local ns
  ns="$(echo "$algo" | tr -c 'A-Za-z0-9' '_')"
  rm -rf "$objdir"
  mkdir -p "$objdir"
  local cfile
  for cfile in "$dest"/*.c; do
    gcc $(native_cflags "$ns") $(algo_defs "$algo") $(includes_for "$dest") -c "$cfile" -o "$objdir/$(basename "$cfile" .c).o"
  done
  ar rcs "$dest/libverify.a" "$objdir"/*.o
  cat > "$PQCGo/$algo/SOURCE.txt" <<EOF
algorithm=$algo
source=experiments/pqcc/pqmagic
built=$(date -u +%Y-%m-%dT%H:%M:%SZ)
command=experiments/cryptoupgrade/algorithm/pqcgo/build.sh
EOF
}

build_vectors() {
  local algo="$1"
  local dest="$PQCGo/$algo/native"
  local out="$PQCGo/$algo/testdata"
  local ns
  ns="$(echo "$algo" | tr -c 'A-Za-z0-9' '_')"
  mkdir -p "$out"
  if [ -f "$out/pk.bin" ] && [ "${FORCE_VECTORS:-}" != "1" ]; then
    echo "reusing vectors $out"
    return
  fi
  gcc $(native_cflags "$ns") $(algo_defs "$algo") $(includes_for "$dest") \
    "$dest"/*.c \
    "$HARNESS/gen_vectors.c" \
    -o "$BUILD/$algo/gen_vectors"
  "$BUILD/$algo/gen_vectors" "$out"
}

build_wasm() {
  local algo="$1"
  local dest="$PQCGo/$algo/native"
  local out="$PQCGo/$algo/$algo.wasm"
  mkdir -p "$(dirname "$out")"
  local srcs=()
  local cfile
  for cfile in "$dest"/*.c; do
    case "$(basename "$cfile")" in
      randombytes.c) continue ;;
    esac
    srcs+=("$cfile")
  done
  "$WASI_SDK_PATH/bin/clang" \
    --target=wasm32-wasi \
    --sysroot="$WASI_SDK_PATH/share/wasi-sysroot" \
    -O3 -std=gnu11 -DUSE_SHAKE \
    $(algo_defs "$algo") \
    $(includes_for "$dest") \
    -mexec-model=reactor \
    -Wl,--no-entry \
    -Wl,--export=execute \
    -Wl,--export-memory \
    -Wl,--initial-memory=4194304 \
    -Wl,--max-memory=16777216 \
    -Wl,-z,stack-size=1048576 \
    "${srcs[@]}" \
    "$HARNESS/randombytes_wasi.c" \
    "$HARNESS/execute.c" \
    -o "$out"
  echo "wrote $out ($(wc -c < "$out") bytes)"
}

copy_existing_aigis_artifacts() {
  local model="$ROOT/cryptoupgrade/internal/model/aigis_sig2"
  if [ -f "$model/aigis_sig2_verify.wasm" ]; then
    cp -a "$model/aigis_sig2_verify.wasm" "$PQCGo/aigis_sig2_verify/aigis_sig2_verify.wasm"
  fi
  if [ -f "$model/pk.bin" ]; then
    mkdir -p "$PQCGo/aigis_sig2_verify/testdata"
    cp -a "$model/pk.bin" "$model/message.bin" "$model/signature.bin" "$PQCGo/aigis_sig2_verify/testdata/"
  fi
}

for algo in $ALGOS; do
  echo "=== snapshot $algo ==="
  mkdir -p "$PQCGo/$algo/testdata" "$BUILD/$algo"
  case "$algo" in
    aigis_sig2_verify) snapshot_aigis ;;
    dilithium3_verify) snapshot_dilithium ;;
    ml_dsa_65_verify) snapshot_mldsa ;;
    slh_dsa_shake_192f_verify) snapshot_slh ;;
  esac
  echo "=== native lib $algo ==="
  build_native_lib "$algo"
  if [ "$algo" = aigis_sig2_verify ]; then
    copy_existing_aigis_artifacts
  fi
  echo "=== vectors $algo ==="
  build_vectors "$algo"
  echo "=== wasm $algo ==="
  build_wasm "$algo"
done

echo "pqcgo archive ready under $PQCGo"
