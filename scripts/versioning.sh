#!/bin/sh

set -euo pipefail

export FULL_GIT_VERSION=$(git rev-parse HEAD)
export GIT_VERSION=$(git describe --always --abbrev=7 --dirty --match=NeVeRmAtCh)
REL_VERSION=$(git tag --list 'v*' --points-at HEAD | tail -n 1)

if [ "$REL_VERSION" == "" ]; then
  git_branch=$(git rev-parse --abbrev-ref HEAD)
  if [ "$git_branch" == "master" ]; then
    REL_VERSION=edge
  else
    REL_VERSION=unstable
  fi
fi

export REL_VERSION
