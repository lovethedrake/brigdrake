#!/usr/bin/env bash

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run publish-worker-binary`

set -euox pipefail

source scripts/versioning.sh

go get -u github.com/tcnksm/ghr

set +x

echo "Publishing binary for commit $FULL_GIT_VERSION"

ghr -t $GITHUB_TOKEN -u lovethedrake -r brigdrake -c $FULL_GIT_VERSION -delete $REL_VERSION /shared/bin/
