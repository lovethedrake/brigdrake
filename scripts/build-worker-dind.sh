#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `mallard run build-dind

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

scripts/docker-build.sh
