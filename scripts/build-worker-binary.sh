#!/usr/bin/env bash

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `mallard run build-binary`

set -euo pipefail

source scripts/versioning.sh

base_package_name=github.com/lovethedrake/canard
ldflags="-w -X $base_package_name/pkg/version.version=$REL_VERSION -X $base_package_name/pkg/version.commit=$GIT_VERSION"

set -x

GOOS=linux GOARCH=amd64 go build -ldflags "$ldflags" -o /shared/bin/canard-linux-amd64 ./cmd/canard
