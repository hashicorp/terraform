#!/usr/bin/env bash
set -uo pipefail

# The actions/checkout action tries hard to fetch as little as
# possible, to the extent that even with "depth: 0" it fails to
# produce enough tag metadata for us to "describe" successfully.
# We'll therefore re-fetch the tags here to make sure we will
# select the most accurate version number.
git fetch origin --force --tags --quiet --unshallow
git log --tags --simplify-by-decoration --decorate-refs='refs/tags/v*' --pretty=format:'%h %<|(35)%S %ci' --max-count 15 --topo-order
set -e
RAW_VERSION=$(git describe --tags --match='v*' ${GITHUB_SHA})

echo "raw-version=${RAW_VERSION}" | tee -a "${GITHUB_OUTPUT}"