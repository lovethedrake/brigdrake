#!/bin/sh

set -euox pipefail

docker build \
  --build-arg VERSION=$REL_VERSION \
  --build-arg COMMIT=$GIT_VERSION \
  -t $BASE_IMAGE_NAME:$GIT_VERSION \
  .
docker tag $BASE_IMAGE_NAME:$GIT_VERSION $BASE_IMAGE_NAME:$REL_VERSION
