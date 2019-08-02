#!/bin/sh

set -euox pipefail

docker push $base_image_name:$git_version
docker push $base_image_name:$rel_version
