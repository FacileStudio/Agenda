#!/usr/bin/env sh
set -eu
sed -n 's/^[[:space:]]*"version":[[:space:]]*"\([^"]*\)".*/v\1/p' "${1:-apps/client/package.json}" | head -n 1
