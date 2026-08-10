#!/usr/bin/env bash
set -euo pipefail

ROOT="${ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
GO_DIR="$ROOT/cryptoupgrade/algorithm/go"
GO_ARCHIVE_DIR="$GO_DIR/archive"
SOL_DIR="$ROOT/cryptoupgrade/algorithm/contracts"

status=0

fail() {
    printf 'ERROR: %s\n' "$*" >&2
    status=1
}

extract_go_entries() {
    perl -ne 'print "$1\n" if /^func\s+([A-Z][A-Za-z0-9_]*)\s*\(/' "$1"
}

extract_sol_entries() {
    perl -0ne 'while (/function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)\s*([^{};]*?)\{/sg) { my ($name, $tail) = ($1, $2); print "$name\n" if $tail =~ /\b(?:external|public)\b/ }' "$1"
}

has_contract() {
    local file=$1
    local contract=$2
    grep -Eq "\\b(abstract[[:space:]]+)?contract[[:space:]]+$contract\\b" "$file"
}

join_lines() {
    local IFS=','
    printf '%s' "$*"
}

go_files=(
    add.go
    blake2b.go
    dh2048.go
    pbkdf2_sha256.go
    pedersen_commit.go
    schnorr_proof.go
    sha256.go
    archive/aes_cbc.go
    archive/ed25519.go
    archive/random_bytes.go
    archive/shamir.go
)

declare -A expected_go=(
    [add.go]=Add
    [blake2b.go]=Sum256
    [dh2048.go]=Dh2048Secret
    [pbkdf2_sha256.go]=Pbkdf2Sha256
    [pedersen_commit.go]=PedersenCommit
    [schnorr_proof.go]=SchnorrVerify
    [sha256.go]=Sha256
    [archive/aes_cbc.go]=AesCBCEncrypt
    [archive/ed25519.go]=Ed25519Verify
    [archive/random_bytes.go]=RandomBytes
    [archive/shamir.go]=ShamirRecover
)

declare -A sol_file=(
    [add.go]=Add.sol
    [blake2b.go]=Blake2b.sol
    [dh2048.go]=Dh2048.sol
    [pbkdf2_sha256.go]=Pbkdf2Sha256.sol
    [pedersen_commit.go]=PedersenCommit.sol
    [schnorr_proof.go]=SchnorrProof.sol
    [sha256.go]=Sha256.sol
)

declare -A archive_go=(
    [archive/aes_cbc.go]=1
    [archive/ed25519.go]=1
    [archive/random_bytes.go]=1
    [archive/shamir.go]=1
)

declare -A sol_contract=(
    [add.go]=AddContract
    [blake2b.go]=Blake2b
    [dh2048.go]=Dh2048
    [pbkdf2_sha256.go]=Pbkdf2Sha256Contract
    [pedersen_commit.go]=PedersenCommitContract
    [schnorr_proof.go]=SchnorrProof
    [sha256.go]=Sha256Contract
)

declare -A sol_entry=(
    [add.go]=Add
    [blake2b.go]=Sum256
    [dh2048.go]=Dh2048Secret
    [pbkdf2_sha256.go]=Pbkdf2Sha256
    [pedersen_commit.go]=PedersenCommit
    [schnorr_proof.go]=SchnorrVerify
    [sha256.go]=Sha256
)

declare -A semantic_notes=(
    [add.go]='一致：int256 加法；Solidity 使用 checked arithmetic，溢出会 revert，Go 侧在 ABI int256 有效域内等价。'
    [blake2b.go]='一致：BLAKE2b-256；Solidity 实现限单块输入 <=128 bytes，Go 实现支持更长输入。'
    [dh2048.go]='一致：RFC 3526 2048-bit group 中 peerPublicKey^privateKey mod p；无效 scalar 或 group element 均返回空结果。'
    [pbkdf2_sha256.go]='一致：PBKDF2-HMAC-SHA256；Solidity 为链上可执行性限制 iterations <=10000、keyLength <=1024。'
    [pedersen_commit.go]='一致：g^message * h^blinding mod p；Go 显式 mod q，Solidity 依赖 subgroup order，结果等价。'
    [schnorr_proof.go]='一致：验证 g^s == t*y^c，challenge 为 sha256("cryptoupgrade-schnorr" || y || t || message)。'
    [sha256.go]='一致：SHA-256 digest，Solidity 通过内置 sha256 opcode 返回同一 32-byte 结果。'
)

if [[ ! -d "$GO_DIR" ]]; then
    fail "Go algorithm directory not found: $GO_DIR"
fi
if [[ ! -d "$GO_ARCHIVE_DIR" ]]; then
    fail "Go archive directory not found: $GO_ARCHIVE_DIR"
fi
if [[ ! -d "$SOL_DIR" ]]; then
    fail "Solidity algorithm directory not found: $SOL_DIR"
fi

declare -A seen_go=()
while IFS= read -r path; do
    file=${path#"$GO_DIR/"}
    seen_go[$file]=1
    if [[ -z "${expected_go[$file]:-}" ]]; then
        fail "unexpected Go algorithm file: $file"
        continue
    fi
    if [[ -n "${sol_file[$file]:-}" && "$file" == archive/* ]]; then
        fail "$file has Solidity counterpart and should stay in $GO_DIR"
    fi
    if [[ -z "${archive_go[$file]:-}" && -z "${sol_file[$file]:-}" ]]; then
        fail "$file has no Solidity counterpart and should be listed under archive/"
    fi
    mapfile -t entries < <(extract_go_entries "$path")
    if [[ ${#entries[@]} -ne 1 ]]; then
        fail "$file should expose exactly one Go function, got ${#entries[@]}: $(join_lines "${entries[@]}")"
        continue
    fi
    if [[ "${entries[0]}" != "${expected_go[$file]}" ]]; then
        fail "$file exposes ${entries[0]}, expected ${expected_go[$file]}"
    fi
done < <(find "$GO_DIR" -maxdepth 2 -type f -name '*.go' | sort)

for file in "${go_files[@]}"; do
    if [[ -z "${seen_go[$file]:-}" ]]; then
        fail "missing Go algorithm file: $file"
    fi
done

declare -A expected_sol_files=()
for file in "${go_files[@]}"; do
    if [[ -n "${sol_file[$file]:-}" ]]; then
        expected_sol_files[${sol_file[$file]}]=$file
    fi
done

while IFS= read -r path; do
    file=$(basename "$path")
    if [[ "$file" == "BigMod.sol" ]]; then
        mapfile -t helper_entries < <(extract_sol_entries "$path")
        if [[ ${#helper_entries[@]} -ne 0 ]]; then
            fail "BigMod.sol should not expose public/external functions: $(join_lines "${helper_entries[@]}")"
        fi
        continue
    fi
    go_file=${expected_sol_files[$file]:-}
    if [[ -z "$go_file" ]]; then
        fail "unexpected Solidity algorithm file: $file"
        continue
    fi
    contract=${sol_contract[$go_file]}
    if ! has_contract "$path" "$contract"; then
        fail "$file should declare contract $contract"
    fi
    mapfile -t entries < <(extract_sol_entries "$path")
    if [[ ${#entries[@]} -ne 1 ]]; then
        fail "$file should expose exactly one Solidity function, got ${#entries[@]}: $(join_lines "${entries[@]}")"
        continue
    fi
    if [[ "${entries[0]}" != "${sol_entry[$go_file]}" ]]; then
        fail "$file exposes ${entries[0]}, expected ${sol_entry[$go_file]}"
    fi
done < <(find "$SOL_DIR" -maxdepth 1 -type f -name '*.sol' | sort)

for file in "${go_files[@]}"; do
    if [[ -n "${sol_file[$file]:-}" && ! -f "$SOL_DIR/${sol_file[$file]}" ]]; then
        fail "missing Solidity counterpart for $file: ${sol_file[$file]}"
    fi
done

printf '\nGo/Solidity 算法入口统计\n'
printf '%-28s %-18s %-22s %-18s %s\n' 'Go 文件' 'Go 入口' 'Solidity 文件' 'Solidity 入口' '结论'
printf '%-28s %-18s %-22s %-18s %s\n' '---' '---' '---' '---' '---'
for file in "${go_files[@]}"; do
    if [[ -n "${sol_file[$file]:-}" ]]; then
        printf '%-28s %-18s %-22s %-18s %s\n' "$file" "${expected_go[$file]}" "${sol_file[$file]}" "${sol_entry[$file]}" '匹配'
    else
        printf '%-28s %-18s %-22s %-18s %s\n' "$file" "${expected_go[$file]}" '-' '-' 'archive，仅保留 Go 实现'
    fi
done

printf '\n操作逻辑判断\n'
for file in "${go_files[@]}"; do
    if [[ -n "${semantic_notes[$file]:-}" ]]; then
        printf -- '- %s / %s: %s\n' "$file" "${sol_file[$file]}" "${semantic_notes[$file]}"
    fi
done

exit "$status"
