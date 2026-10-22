#!/usr/bin/env bash
set -euo pipefail

output_dir="${1:?usage: build-fixtures.sh OUTPUT_DIRECTORY}"
mkdir -p "$output_dir"

build() {
  go run ./cryptoupgrade/wasmtool/cmd/wasmbuild \
    -source "experiments/cryptoupgrade/algorithm/go/$1.go" \
    -out "$output_dir/$2.wasm" -function "$3" -itype "$4" -otype "$5"
}

build add add Add int256,int256 int256
build sha256 sha256 Sha256 bytes bytes
build blake2b blake2b Sum256 bytes bytes32
build pbkdf2_sha256 pbkdf2_sha256 Pbkdf2Sha256 bytes,bytes,uint256,uint256 bytes
build dh2048 dh2048 Dh2048Secret bytes,bytes bytes
build pedersen_commit pedersen_commit PedersenCommit bytes,bytes bytes
build schnorr_proof schnorr_proof SchnorrVerify bytes,bytes bool
build polynomial_mul polynomial_mul PolynomialMul 'uint256[],uint256[],uint256' 'uint256[]'
