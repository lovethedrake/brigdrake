#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run publish-worker-dind`

set -euo pipefail

source scripts/versioning.sh
source scripts/base-docker.sh

dockerd_logs=$(mktemp)

function dumpDockerdLogs {
  set +x
  echo "---------- Dumping dockerd logs ----------"
  cat $dockerd_logs
}

trap dumpDockerdLogs EXIT

set -x

dockerd \
  --host=unix:///var/run/docker.sock \
  --host=tcp://0.0.0.0:2375 \
  &> $dockerd_logs &

# Wait for the containerized dockerd to be ready
scripts/wupiao.sh localhost 2375 300

set +x # Don't let the value of $DOCKER_PASSWORD bleed into the logs!
docker login -u krancour -p $DOCKER_PASSWORD
set -x

scripts/docker-build.sh
scripts/docker-publish.sh
