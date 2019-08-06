#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run build-chart`

set -euo pipefail

source scripts/versioning.sh

if [ "$rel_version" == "edge" ]; then
  chart_version=0.0.1-$(date -u +"%Y.%m.%d.%H.%M.%S")-$git_version
else
  chart_version=$rel_version
fi

set -x

# Clean
rm -rf chart/dist

# Build
helm init --client-only
helm repo add brigade https://brigadecore.github.io/charts
helm dep up chart/brigdrake
mkdir chart/dist
helm package --version $chart_version -d chart/dist chart/brigdrake

# Update index
curl -o chart/dist/index.yaml https://raw.githubusercontent.com/lovethedrake/brigdrake/gh-pages/index.yaml
helm repo index --merge chart/dist/index.yaml chart/dist
