#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1

set -uo pipefail

function usage {
  cat <<-'EOF'
Usage: ./equivalence-test.sh <command> [<args>] [<options>]

Description:
  This script will handle various commands related to the execution of the
  Terraform equivalence tests.

Commands:
  get_target_branch <version>
    get_target_branch returns the default target branch for a given Terraform
    version.

    target_branch=$(./equivalence-test.sh get_target_branch v1.4.3); target_branch=v1.4
    target_branch=$(./equivalence-test.sh get_target_branch 1.4.3); target_branch=v1.4

  download_equivalence_test_binary <version> <target> <os> <arch>
    download_equivalence_test_binary downloads the equivalence testing binary
    for a given version and places it at the target path.

    ./equivalence-test.sh download_equivalence_test_binary 0.3.0 ./bin/terraform-equivalence-testing linux amd64

  build_terraform_binary <target>
    download_terraform_binary builds the Terraform binary and places it at the
    target path.

    ./equivalence-test.sh build_terraform_binary ./bin/terraform
EOF
}

function download_equivalence_test_binary {
  VERSION="${1:-}"
  TARGET="${2:-}"
  OS="${3:-}"
  ARCH="${4:-}"

  if [[ -z "$VERSION" || -z "$TARGET" || -z "$OS" || -z "$ARCH" ]]; then
    echo "missing at least one of [<version>, <target>, <os>, <arch>] arguments"
    usage
    exit 1
  fi

  curl \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/hashicorp/terraform-equivalence-testing/releases" > releases.json

  ASSET="terraform-equivalence-testing_v${VERSION}_${OS}_${ARCH}.zip"
  ASSET_ID=$(jq -r --arg VERSION "v$VERSION" --arg ASSET "$ASSET" '.[] | select(.name == $VERSION) | .assets[] | select(.name == $ASSET) | .id' releases.json)

  mkdir -p zip
  curl -L \
    -H "Accept: application/octet-stream" \
    "https://api.github.com/repos/hashicorp/terraform-equivalence-testing/releases/assets/$ASSET_ID" > "zip/$ASSET"

  mkdir -p bin
  unzip -p "zip/$ASSET" terraform-equivalence-testing > "$TARGET"
  chmod u+x "$TARGET"
  rm -r zip
  rm releases.json
}

function build_terraform_binary {
  TARGET="${1:-}"

  if [[ -z "$TARGET" ]]; then
    echo "target argument"
    usage
    exit 1
  fi

  go build -o "$TARGET" ./
  chmod u+x "$TARGET"
}

function get_target_branch {
  VERSION="${1:-}"

  if [ -z "$VERSION" ]; then
    echo "missing <version> argument"
    usage
    exit 1
  fi


  # Split off the build metadata part, if any
  # (we won't actually include it in our final version, and handle it only for
  # completeness against semver syntax.)
  IFS='+' read -ra VERSION BUILD_META <<< "$VERSION"

  # Separate out the prerelease part, if any
  IFS='-' read -r BASE_VERSION PRERELEASE <<< "$VERSION"

  # Separate out major, minor and patch versions.
  IFS='.' read -r MAJOR_VERSION MINOR_VERSION PATCH_VERSION <<< "$BASE_VERSION"

  if [[ "$PRERELEASE" == *"alpha"* ]]; then
    TARGET_BRANCH=main
  else
    if [[ $MAJOR_VERSION = v* ]]; then
      TARGET_BRANCH=${MAJOR_VERSION}.${MINOR_VERSION}
    else
      TARGET_BRANCH=v${MAJOR_VERSION}.${MINOR_VERSION}
    fi
  fi

  echo "$TARGET_BRANCH"
}

function main {
  case "$1" in
    get_target_branch)
      if [ "${#@}" != 2 ]; then
        echo "invalid number of arguments"
        usage
        exit 1
      fi

      get_target_branch "$2"

      ;;
    download_equivalence_test_binary)
      if [ "${#@}" != 5 ]; then
        echo "invalid number of arguments"
        usage
        exit 1
      fi

      download_equivalence_test_binary "$2" "$3" "$4" "$5"

      ;;
    build_terraform_binary)
      if [ "${#@}" != 2 ]; then
        echo "invalid number of arguments"
        usage
        exit 1
      fi

      build_terraform_binary "$2"

      ;;
    *)
      echo "unrecognized command $*"
      usage
      exit 1

      ;;
  esac
}

main "$@"
exit $?
