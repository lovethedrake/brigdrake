#!/bin/sh

set -euox pipefail

base_package_name=github.com/lovethedrake/brigdrake

docker build \
  --build-arg VERSION=$rel_version \
  --build-arg COMMIT=$git_version \
  -t $base_image_name:$git_version \
  .
docker tag $base_image_name:$git_version $base_image_name:$rel_version
