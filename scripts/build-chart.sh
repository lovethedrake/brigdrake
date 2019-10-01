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

# Clean
rm -rf chart/build
rm -rf chart/dist

mkdir chart/build
mkdir chart/dist

# Copy
cp -R chart/brigdrake/ chart/build/brigdrake/

# Set version info
sed -i "s/^appVersion:.*/appVersion: $app_version/" chart/build/brigdrake/Chart.yaml
sed -i "s/^    tag:.*/    tag: $app_version/" chart/build/brigdrake/values.yaml

# Make sure helm and repos containing dependencies are in good working order
helm init --client-only
helm repo add brigade https://brigadecore.github.io/charts

# Build!
helm dep up chart/build/brigdrake
helm package --version $chart_version -d chart/dist chart/build/brigdrake

# Update index
curl -o chart/dist/index.yaml https://raw.githubusercontent.com/lovethedrake/brigdrake/gh-pages/index.yaml
helm repo index --merge chart/dist/index.yaml chart/dist
