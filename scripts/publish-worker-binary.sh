#!/usr/bin/env bash

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `mallard run publish-binary`

set -euox pipefail

source scripts/versioning.sh

go get -u github.com/tcnksm/ghr

set +x

echo "Publishing binary for commit $FULL_GIT_VERSION"

ghr -t $GITHUB_TOKEN -u lovethedrake -r canard -c $FULL_GIT_VERSION -delete $REL_VERSION /shared/bin/
