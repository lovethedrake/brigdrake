#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `mallard run build-and-publish-dood`

set -euo pipefail

source scripts/versioning.sh
source scripts/base-docker.sh

scripts/docker-build.sh
scripts/docker-publish.sh
