#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$ROOT/../../.." && pwd)"

build_module_from_go() {
  local source_path="$1"
  local function_name="$2"
  local input_types="$3"
  local output_types="$4"
  local output_path="$5"
  (
    cd "$REPO_ROOT"
    go run ./cryptoupgrade/wasmtool/cmd/wasmbuild \
      -source "$source_path" \
      -function "$function_name" \
      -itype "$input_types" \
      -otype "$output_types" \
      -out "$output_path"
  )
}

build_module_from_go "cryptoupgrade/algorithm/go/add.go" "Add" "int256,int256" "int256" "$ROOT/add.wasm"
build_module_from_go "cryptoupgrade/algorithm/go/sha256.go" "Sha256" "bytes" "bytes" "$ROOT/sha256.wasm"
build_module_from_go "cryptoupgrade/algorithm/go/blake2b.go" "Sum256" "bytes" "bytes32" "$ROOT/blake2b.wasm"
build_module_from_go "cryptoupgrade/algorithm/go/pbkdf2_sha256.go" "Pbkdf2Sha256" "bytes,bytes,uint256,uint256" "bytes" "$ROOT/pbkdf2_sha256.wasm"
build_module_from_go "cryptoupgrade/algorithm/go/dh2048.go" "Dh2048Secret" "bytes,bytes" "bytes" "$ROOT/dh2048.wasm"
build_module_from_go "cryptoupgrade/algorithm/go/pedersen_commit.go" "PedersenCommit" "bytes,bytes" "bytes" "$ROOT/pedersen_commit.wasm"
build_module_from_go "cryptoupgrade/algorithm/go/schnorr_proof.go" "SchnorrVerify" "bytes,bytes" "bool" "$ROOT/schnorr_proof.wasm"
