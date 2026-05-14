#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME="${1:-hub.shiyak-office.com/storage/cubefs-dashboard}"
IMAGE_TAG="${2:-v1.0.0}"

docker build \
  --no-cache \
  --platform linux/amd64 \
  -t "${IMAGE_NAME}:${IMAGE_TAG}" \
  --load \
  .

echo "image built: ${IMAGE_NAME}:${IMAGE_TAG}"
