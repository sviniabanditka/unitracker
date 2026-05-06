#!/usr/bin/env bash
set -euo pipefail

# Builds the Tracker image and pushes it to the private registry.
# Usage: ./deploy/build-push.sh [tag]   (default tag: latest)

TAG="${1:-latest}"
REGISTRY="${REGISTRY:-registry.sviniabanditka.com}"
IMAGE="${REGISTRY}/tracker:${TAG}"
VERSION="$(git rev-parse --short HEAD 2>/dev/null || echo dev)"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

docker build \
  --platform linux/amd64 \
  --build-arg "VERSION=${VERSION}" \
  -f "${REPO_ROOT}/deploy/Dockerfile" \
  -t "${IMAGE}" \
  "${REPO_ROOT}"

docker push "${IMAGE}"
echo "pushed ${IMAGE} (version=${VERSION})"
