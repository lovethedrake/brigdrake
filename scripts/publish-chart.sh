#!/bin/sh

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `drake run publish-chart`

set -euox

npm install -g gh-pages@2.0.1

set +x # Don't let the value of $GITHUB_TOKEN bleed into the logs!

cp -r /shared/chart/dist chart/dist

gh-pages --add -d chart/dist \
  -r https://drakeci:$GITHUB_TOKEN@github.com/lovethedrake/brigdrake.git \
  -u "Drake CI <drake@ci>" \
  -m "Add chart artifacts and update index.yaml"
