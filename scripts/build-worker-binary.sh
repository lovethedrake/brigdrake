#!/usr/bin/env bash

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run build-worker-binary`

set -euo pipefail

source scripts/versioning.sh

base_package_name=github.com/lovethedrake/brigdrake
ldflags="-w -X $base_package_name/pkg/version.version=$rel_version -X $base_package_name/pkg/version.commit=$git_version"

set -x

GOOS=linux GOARCH=amd64 go build -ldflags "$ldflags" -o /shared/bin/brigdrake-worker-linux-amd64 ./cmd/brigdrake-worker
