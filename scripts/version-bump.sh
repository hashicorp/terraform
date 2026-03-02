#!/usr/bin/env bash
# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1


set -uo pipefail

function usage {
  cat <<-'EOF'
Usage: ./version-bump.sh <version>

Description:
  This script will update the version/VERSION file with the given version.
EOF
}

function update_version {
  VERSION="${1:-}"

  if [[ -z "$VERSION" ]]; then
    echo "missing at least one of [<version>] arguments"
    usage
    exit 1
  fi

  echo "$VERSION" > version/VERSION
}

update_version "$@"
exit $?
