#!/bin/sh

set -euox pipefail

set +x # Don't let the value of $DOCKER_PASSWORD bleed into the logs!
echo $DOCKER_PASSWORD | docker login $DOCKER_REGISTRY -u $DOCKER_USER --password-stdin
set -x

docker push $BASE_IMAGE_NAME:$GIT_VERSION
docker push $BASE_IMAGE_NAME:$REL_VERSION
