#!/usr/bin/env bash
# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./equivalence-test.sh <command> [<args>]

Description:
  This script handles commands related to Terraform equivalence tests.

Commands:
  get_target_branch <version>
    Returns the default target branch for a given Terraform version.

    Examples:
      target_branch=$(./equivalence-test.sh get_target_branch v1.4.3)   # v1.4
      target_branch=$(./equivalence-test.sh get_target_branch 1.4.3)    # v1.4
      target_branch=$(./equivalence-test.sh get_target_branch 1.5.0-alpha20240101)  # main

  download_equivalence_test_binary <version> <target> <os> <arch>
    Downloads the equivalence testing binary for a given version and writes it
    to the target path.

    Example:
      ./equivalence-test.sh download_equivalence_test_binary 0.3.0 ./bin/terraform-equivalence-testing linux amd64

  build_terraform_binary <target>
    Builds the Terraform binary and writes it to the target path.

    Example:
      ./equivalence-test.sh build_terraform_binary ./bin/terraform
EOF
}

require_command() {
  local cmd="${1:?missing command name}"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
}

cleanup() {
  if [[ -n "${TMP_DIR:-}" && -d "${TMP_DIR:-}" ]]; then
    rm -rf "$TMP_DIR"
  fi
}

download_equivalence_test_binary() {
  local version="${1:-}"
  local target="${2:-}"
  local os="${3:-}"
  local arch="${4:-}"

  if [[ -z "$version" || -z "$target" || -z "$os" || -z "$arch" ]]; then
    echo "error: missing at least one required argument: <version> <target> <os> <arch>" >&2
    usage
    exit 1
  fi

  require_command curl
  require_command jq
  require_command unzip
  require_command mktemp

  TMP_DIR="$(mktemp -d)"
  trap cleanup EXIT

  local normalized_version="$version"
  if [[ "$normalized_version" != v* ]]; then
    normalized_version="v${normalized_version}"
  fi

  local releases_json="${TMP_DIR}/releases.json"
  local asset_name="terraform-equivalence-testing_${normalized_version}_${os}_${arch}.zip"
  local asset_id
  local zip_path="${TMP_DIR}/${asset_name}"

  curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/hashicorp/terraform-equivalence-testing/releases" \
    -o "$releases_json"

  asset_id="$(jq -r --arg version "$normalized_version" --arg asset "$asset_name" '
    .[]
    | select(.name == $version)
    | .assets[]
    | select(.name == $asset)
    | .id
  ' "$releases_json")"

  if [[ -z "$asset_id" || "$asset_id" == "null" ]]; then
    echo "error: could not find release asset '$asset_name' for version '$normalized_version'" >&2
    exit 1
  fi

  mkdir -p "$(dirname "$target")"

  curl -fsSL \
    -H "Accept: application/octet-stream" \
    "https://api.github.com/repos/hashicorp/terraform-equivalence-testing/releases/assets/${asset_id}" \
    -o "$zip_path"

  unzip -p "$zip_path" terraform-equivalence-testing > "$target"
  chmod u+x "$target"
}

build_terraform_binary() {
  local target="${1:-}"

  if [[ -z "$target" ]]; then
    echo "error: missing <target> argument" >&2
    usage
    exit 1
  fi

  require_command go

  mkdir -p "$(dirname "$target")"
  go build -o "$target" ./
  chmod u+x "$target"
}

get_target_branch() {
  local version="${1:-}"

  if [[ -z "$version" ]]; then
    echo "error: missing <version> argument" >&2
    usage
    exit 1
  fi

  local normalized_version="$version"
  normalized_version="${normalized_version#v}"

  local version_without_build="${normalized_version%%+*}"
  local base_version="${version_without_build%%-*}"
  local prerelease=""

  if [[ "$version_without_build" == *-* ]]; then
    prerelease="${version_without_build#*-}"
  fi

  local major_version=""
  local minor_version=""
  local patch_version=""

  IFS='.' read -r major_version minor_version patch_version <<< "$base_version"

  if [[ -z "$major_version" || -z "$minor_version" || -z "$patch_version" ]]; then
    echo "error: invalid version format '$version'; expected semver like 1.4.3 or v1.4.3" >&2
    exit 1
  fi

  if [[ -n "$prerelease" && "$prerelease" == *alpha* ]]; then
    echo "main"
  else
    echo "v${major_version}.${minor_version}"
  fi
}

main() {
  if [[ "$#" -lt 1 ]]; then
    echo "error: missing command" >&2
    usage
    exit 1
  fi

  case "$1" in
    get_target_branch)
      if [[ "$#" -ne 2 ]]; then
        echo "error: invalid number of arguments for get_target_branch" >&2
        usage
        exit 1
      fi
      get_target_branch "$2"
      ;;
    download_equivalence_test_binary)
      if [[ "$#" -ne 5 ]]; then
        echo "error: invalid number of arguments for download_equivalence_test_binary" >&2
        usage
        exit 1
      fi
      download_equivalence_test_binary "$2" "$3" "$4" "$5"
      ;;
    build_terraform_binary)
      if [[ "$#" -ne 2 ]]; then
        echo "error: invalid number of arguments for build_terraform_binary" >&2
        usage
        exit 1
      fi
      build_terraform_binary "$2"
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      echo "error: unrecognized command: $*" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
