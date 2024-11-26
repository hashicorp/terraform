#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1


set -uo pipefail

function usage {
  cat <<-'EOF'
Usage: ./changelog.sh <command> [<options>]

Description:
  This script will update CHANGELOG.md with the given version and date.

Commands:
  prepare <version> <date>
    prepare updates the first line in the CHANGELOG.md file with the
    given version and date.

    ./changelog.sh prepare 1.0.0 "November 1, 2021"

  cleanup <released-version> <next-version>
    cleanup prepends a new section to the CHANGELOG.md file with the given
    version and (Unreleased) as the date. If the released version contains a
    pre-release tag, the next version will replace the top line instead of
    inserting a new section.
EOF
}

function prepare {
  VERSION="${1:-}"
  DATE="${2:-}"

  if [[ -z "$VERSION" || -z "$DATE" ]]; then
    echo "missing at least one of [<version>, <date>] arguments"
    usage
    exit 1
  fi

  sed -i '' -e "1s/.*/## $VERSION ($DATE)/" CHANGELOG.md
}

function cleanup {
  RELEASED_VERSION="${1:-}"
  NEXT_VERSION="${2:-}"

  if [[ -z "$RELEASED_VERSION" || -z "$NEXT_VERSION" ]]; then
    echo "missing at least one of [<released-version>, <next-version>] arguments"
    usage
    exit 1
  fi

  if [[ "$RELEASED_VERSION" == *-* ]]; then
    # then we have a pre-release version, so we should replace the top line
    sed -i '' -e "1s/.*/## $NEXT_VERSION (Unreleased)/" CHANGELOG.md
  else
    sed -i '' -e "1s/^/## $NEXT_VERSION (Unreleased)\n\n/" CHANGELOG.md
  fi
}

function main {
  case "$1" in
    prepare)
      prepare "${@:2}"

      ;;
    cleanup)
      cleanup "${@:2}"

      ;;
    *)
      usage
      exit 1

      ;;
  esac
}

main "$@"
exit $?
