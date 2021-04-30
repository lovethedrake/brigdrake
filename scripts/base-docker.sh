#!/bin/sh

set -euo pipefail

DOCKER_REGISTRY=${DOCKER_REGISTRY:-ghcr.io}
DOCKER_REGISTRY_NAMESPACE=${DOCKER_REGISTRY_NAMESPACE:-lovethedrake}

# Append a trailing slash if set
if [ "$DOCKER_REGISTRY" != "" ]; then
  DOCKER_REGISTRY=$DOCKER_REGISTRY/
fi

if [ "$DOCKER_REGISTRY_NAMESPACE" != "" ]; then
  DOCKER_REGISTRY_NAMESPACE=$DOCKER_REGISTRY_NAMESPACE/
fi

export base_image_name=${DOCKER_REGISTRY}${DOCKER_REGISTRY_NAMESPACE}brigdrake-worker
