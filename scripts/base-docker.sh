#!/bin/sh

set -eo pipefail

set +u # Some vars may be unbound-- that's ok here

if [ "$DOCKER_REGISTRY" == "" ]; then
  docker_registry=""
else
  docker_registry=$DOCKER_REGISTRY/
fi

if [ "$DOCKER_REGISTRY_NAMESPACE" == "" ]; then
  docker_registry_namespace=""
else
  docker_registry_namespace=$DOCKER_REGISTRY_NAMESPACE/
fi

set -u

export base_image_name=${docker_registry}${docker_registry_namespace}brigdrake-worker
