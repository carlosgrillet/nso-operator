#!/usr/bin/env bash

set -euo pipefail

CLUSTER_NAME="kind"
IMAGES=(
  "carlosgrillet/cisco-nso:6.1.19-prod"
  "carlosgrillet/cisco-nso:6.1.19-build"
)

log() { echo "🔍 [load-images] $*" >&2; }
error() { echo "❌ [load-images] ERROR: $*" >&2; exit 1; }

command -v kind >/dev/null || error "kind not found. Install: https://kind.sigs.k8s.io/"

if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  error "Cluster '${CLUSTER_NAME}' not found. Create with: kind create cluster"
fi

for img in "${IMAGES[@]}"; do
  IMG_NAME="${img%:*}"
  IMG_TAG="${img##*:}"

  if ! docker image inspect "$img" >/dev/null 2>&1; then
    log "📥 Pulling $img (not found locally)..."
    docker pull "$img" || error "Failed to pull $img"
  fi

  if docker exec "${CLUSTER_NAME}-control-plane" crictl images | grep -q "${IMG_NAME}.*${IMG_TAG}"; then
    log "✅ Already loaded: $img"
    continue
  fi

  log "🚚 Loading $img into cluster '${CLUSTER_NAME}'..."
  kind load docker-image "$img" &
done
wait

log "✅ All images loaded successfully."
