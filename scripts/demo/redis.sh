#!/usr/bin/env bash

# AVOID INVOKING THIS SCRIPT DIRECTLY -- USE `mallard run redis`

set -euox pipefail

redis-cli set foo bar

redis-cli get foo
