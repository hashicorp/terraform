#!/usr/bin/env bash

set -uo pipefail

function usage {
  cat <<-'EOF'
Usage: ./changelog.sh <version> <date>

Description:
  This script will update the first line in the CHANGELOG.md file with the given
  version and date.
EOF
}

function update_changelog {
  VERSION="${1:-}"
  DATE="${2:-}"

  if [[ -z "$VERSION" || -z "$DATE" ]]; then
    echo "missing at least one of [<version>, <date>] arguments"
    usage
    exit 1
  fi

  sed -i '' -e "1s/.*/## $VERSION ($DATE)/" CHANGELOG.md
}

update_changelog "$@"
exit $?
