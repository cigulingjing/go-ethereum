#!/bin/sh
set -eu

IMAGE="${IMAGE:-cryptoupgrade-geth:lab}"
CONFIG="${CONFIG:-experiments/cryptoupgrade/deployments/networks/local-2nodes.yaml}"
OUT="${OUT:-build/cryptoupgrade-networks/docker-check}"

if ! command -v docker >/dev/null 2>&1; then
  echo "skip: docker command not found"
  exit 0
fi

docker build -f experiments/cryptoupgrade/deployments/docker/Dockerfile.lab -t "$IMAGE" .

go run ./experiments/cryptoupgrade/bench/cmd/multinode \
  -mode render \
  -config "$CONFIG" \
  -out "$OUT" \
  -output-json "$OUT/render-result.json"

if docker compose version >/dev/null 2>&1; then
  docker compose -f "$OUT/docker-compose.yml" config >/dev/null
else
  echo "skip: docker compose plugin not found"
fi
