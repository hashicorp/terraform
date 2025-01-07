#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1


set -uo pipefail

CHANGIE_VERSION="${CHANGIE_VERSION:-1.21.0}"

function usage {
  cat <<-'EOF'
Usage: ./changelog.sh <command> [<options>]

Description:
  This script will update CHANGELOG.md with the given version and date.

Commands:
  generate <release-type>
    generate will create a new section in the CHANGELOG.md file for the given release
    type. The release type should be one of "dev", "alpha", "release", or "patch".
    `dev`: will update the changelog with the latest unreleased changes.
    `alpha`: will generate a new section with an alpha version for today.
    `release`: will make the initial minor release for this branch.
    `patch`: will generate a new patch release
    

  nextminor
    Run this to get a new release branch for the next minor version.
EOF
}

function generate {
    RELEASE_TYPE="${1:-}"
    
    if [[ -z "$RELEASE_TYPE" ]]; then
        echo "missing <release-type> argument"
        usage
        exit 1
    fi
    
    case "$RELEASE_TYPE" in
  
        dev)
        LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
        COMPLETE_VERSION="$LATEST_VERSION-dev"

    
        npx -y changie@$CHANGIE_VERSION merge -u "## $LATEST_VERSION (Unreleased)"
        
        # If we have no changes yet, the changelog is empty now, so we need to add a header
        if [[ ! -s CHANGELOG.md ]]; then
            echo "## $LATEST_VERSION (Unreleased)" > CHANGELOG.md
            echo "" >> CHANGELOG.md
        fi
        ;;

        alpha)
        PRERELEASE_VERSION=$(date +"alpha%Y%m%d")
        LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
        HUMAN_DATE=$(date +"%B %d, %Y") # Date in Janurary 1st, 2022 format
        COMPLETE_VERSION="$LATEST_VERSION-$PRERELEASE_VERSION"

        npx -y changie@$CHANGIE_VERSION merge -u "## $COMPLETE_VERSION ($HUMAN_DATE)"
        ;;
        patch)
        COMPLETE_VERSION=$(npx -y changie@$CHANGIE_VERSION next patch)
        COMPLETE_VERSION=${COMPLETE_VERSION:1} # remove the v prefix
        npx -y changie@$CHANGIE_VERSION batch patch
        npx -y changie@$CHANGIE_VERSION merge
        ;;
        
        release)
        # This is the first release of the branch, releasing the new minor version
        COMPLETE_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
        # We currently keep a file that looks like this release to ensure the alphas and dev versions are generated correctly
        rm ./.changes/$COMPLETE_VERSION.md

        npx -y changie@$CHANGIE_VERSION batch $COMPLETE_VERSION
        npx -y changie@$CHANGIE_VERSION merge
        ;;

        *)
        echo "invalid <release-type> argument"
        usage
        exit 1
    
        ;;
    esac

    # Set version/VERSION to the to be released version
    echo "$COMPLETE_VERSION" > version/VERSION
    
    # Add footer to the changelog
    cat ./.changes/experiments.md >> CHANGELOG.md
    echo "" >> CHANGELOG.md
    cat ./.changes/previous-releases.md >> CHANGELOG.md
}

function nextminor {
    LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
    LATEST_VERSION=${LATEST_VERSION%.*} # Remove the patch version
    CURRENT_FILE_CONTENT=$(cat ./.changes/previous-releases.md)
    # Prepend the latest version to the previous releases
    echo "- [v$LATEST_VERSION](https://github.com/hashicorp/terraform/blob/v$LATEST_VERSION/CHANGELOG.md)" > ./.changes/previous-releases.md
    echo "$CURRENT_FILE_CONTENT" >> ./.changes/previous-releases.md

    NEXT_VERSION=$(npx -y changie@$CHANGIE_VERSION next minor)
    # Remove all existing per-release changelogs
    rm ./.changes/*.*.*.md
    # Remove all unreleased changes
    rm ./.changes/unreleased/*.yaml
    # Create a new empty version file for the next minor version
    touch ./.changes/$NEXT_VERSION.md
    
    generate "dev"
}

function main {
  case "$1" in
    generate)
    generate "${@:2}"

      ;;
    nextminor)
    nextminor "${@:2}"

      ;;
    *)
      usage
      exit 1

      ;;
  esac
}

main "$@"
exit $?
