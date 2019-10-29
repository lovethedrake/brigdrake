#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run build-chart`

set -euo pipefail

source scripts/versioning.sh

if [ "$rel_version" == "edge" ] || [ "$rel_version" == "unstable" ]; then
  chart_version=0.0.1-$(date -u +"%Y.%m.%d.%H.%M.%S")-$git_version
  app_version=$git_version
else
  # Strip away the leading "v" from $rel_version
  chart_version=$(echo $rel_version | cut -c 2-)
  app_version=$rel_version
fi

set -x

# Set version info
sed -i "s/^appVersion:.*/appVersion: $app_version/" chart/brigdrake/Chart.yaml
sed -i "s/^    tag:.*/    tag: $app_version/" chart/brigdrake/values.yaml

# Make sure helm and repos containing dependencies are in good working order
helm init --client-only
helm repo add brigade https://brigadecore.github.io/charts

# Build!
helm dep up chart/brigdrake
mkdir -p /shared/chart/dist
helm package --version $chart_version -d /shared/chart/dist chart/brigdrake

# Update index
curl -o /shared/chart/dist/index.yaml https://raw.githubusercontent.com/lovethedrake/brigdrake/gh-pages/index.yaml
helm repo index --merge /shared/chart/dist/index.yaml /shared/chart/dist
