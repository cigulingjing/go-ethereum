#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  printf 'layout check failed: %s\n' "$*" >&2
  exit 1
}

require_dir() {
  local dir="$1"
  [[ -d "$ROOT/$dir" ]] || fail "missing directory $dir"
}

require_absent() {
  local path="$1"
  local replacement="$2"
  [[ ! -e "$ROOT/$path" ]] || fail "$path must not exist; use $replacement"
}

require_dir cryptoupgrade
require_dir cryptoupgrade/algorithm/go
require_dir cryptoupgrade/algorithm/go/archive
require_dir cryptoupgrade/algorithm/contracts
require_dir cryptoupgrade/builtin
require_dir cryptoupgrade/results
require_dir experiments/cryptoupgrade/bench/cmd
require_dir experiments/cryptoupgrade/network
require_dir experiments/cryptoupgrade/smoke
require_dir experiments/cryptoupgrade/deployments/docker
require_dir experiments/cryptoupgrade/deployments/networks
require_dir experiments/cryptoupgrade/docs
require_dir experiments/cryptoupgrade/results

require_absent cryptoupgrade/cmd experiments/cryptoupgrade/bench/cmd
require_absent cryptoupgrade/bench experiments/cryptoupgrade/bench
require_absent cryptoupgrade/network experiments/cryptoupgrade/network
require_absent cryptoupgrade/docker experiments/cryptoupgrade/deployments/docker
require_absent cryptoupgrade/examples experiments/cryptoupgrade/deployments/networks
require_absent cryptoupgrade/docs experiments/cryptoupgrade/docs
require_absent cryptoupgrade/preload cryptoupgrade/builtin

if rg -n 'cryptoupgrade/algorithm/go' "$ROOT/cryptoupgrade/builtin" -g '*.go' >/dev/null; then
  rg -n 'cryptoupgrade/algorithm/go' "$ROOT/cryptoupgrade/builtin" -g '*.go' >&2
  fail "builtin runtime code must not import dynamic candidate source assets"
fi

while IFS= read -r file; do
  [[ -f "$file/main.go" ]] || fail "experiment command missing main.go: ${file#$ROOT/}"
done < <(find "$ROOT/experiments/cryptoupgrade/bench/cmd" -mindepth 1 -maxdepth 1 -type d | sort)

if find "$ROOT/experiments/cryptoupgrade" -maxdepth 1 -type f \( -name '*.json' -o -name '*.csv' -o -name '*.log' -o -name '*.png' -o -name '*.svg' -o -name '*.pdf' -o -name '*.tiff' \) | grep -q .; then
  fail "new experiment result files must live under experiments/cryptoupgrade/results"
fi

if rg -n 'go run \./cryptoupgrade/(cmd|bench/cmd)|(^|[^/])cryptoupgrade/(docker|examples/networks|docs)/' \
  "$ROOT/experiments/cryptoupgrade/docs" "$ROOT/experiments/cryptoupgrade/deployments" \
  -g '*.md' -g '*.sh' -g '*.yaml' -g '*.yml' -g 'Dockerfile*' >/dev/null; then
  rg -n 'go run \./cryptoupgrade/(cmd|bench/cmd)|(^|[^/])cryptoupgrade/(docker|examples/networks|docs)/' \
    "$ROOT/experiments/cryptoupgrade/docs" "$ROOT/experiments/cryptoupgrade/deployments" \
    -g '*.md' -g '*.sh' -g '*.yaml' -g '*.yml' -g 'Dockerfile*' >&2
  fail "documentation or deployment scripts still reference old experiment paths"
fi

if rg -n 'experiments/cryptoupgrade' "$ROOT/cryptoupgrade" -g '*.go' >/dev/null; then
  rg -n 'experiments/cryptoupgrade' "$ROOT/cryptoupgrade" -g '*.go' >&2
  fail "cryptoupgrade runtime must not import the experiment layer"
fi

if rg -n 'gzip\.NewWriter|func compressFile\b' "$ROOT/experiments/cryptoupgrade" -g '*.go' >/dev/null; then
  rg -n 'gzip\.NewWriter|func compressFile\b' "$ROOT/experiments/cryptoupgrade" -g '*.go' >&2
  fail "experiment code must use the cryptoupgrade source codec facade"
fi

printf 'cryptoupgrade layout check passed\n'
