#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run build-and-publish-worker-dood`

set -euo pipefail

source scripts/versioning.sh
source scripts/base-docker.sh

scripts/docker-build.sh
scripts/docker-publish.sh
