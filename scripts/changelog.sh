#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: BUSL-1.1


set -uo pipefail

CHANGIE_VERSION="${CHANGIE_VERSION:-1.21.0}"
SEMVER_VERSION="${SEMVER_VERSION:-7.6.3}"

function usage {
  cat <<-'EOF'
Usage: ./changelog.sh <command> [<options>]

Description:
  This script will update CHANGELOG.md with the given version and date.

Commands:
  generate <release-type>
    generate will create a new section in the CHANGELOG.md file for the given release
    type. The release type should be one of "dev", "alpha", "rc", "release", or "patch".
    `dev`: will update the changelog with the latest unreleased changes.
    `alpha`: will generate a new section with an alpha version for today.
    `beta`: will generate a new beta release.
    `rc`: will generate a new rc release.
    `release`: will make the initial minor release for this branch.
    `patch`: will generate a new patch release

  nextminor
    This function expects the current branch to be main. Run it if you want to set main to the next
    minor version.
    
  firstbeta
    This function is expected to be run on the branch of the last minor release. It will make sure
    that backports work properly
  
EOF
}

function generate {
    RELEASE_TYPE="${1:-}"
    
    if [[ -z "$RELEASE_TYPE" ]]; then
        echo "missing <release-type> argument"
        usage
        exit 1
    fi
    
    FOOTER_FILE='footer.md'
    
    case "$RELEASE_TYPE" in
  
        dev)
        FOOTER_FILE='footer-with-experiments.md'
        LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)

        # Check if we already released this version already
        if git tag -l "v$LATEST_VERSION" | grep -q "v$LATEST_VERSION"; then
            LATEST_VERSION=$(npx -y semver@$SEMVER_VERSION -i patch $LATEST_VERSION)
        fi

        COMPLETE_VERSION="$LATEST_VERSION-dev"

        npx -y changie@$CHANGIE_VERSION merge -u "## $LATEST_VERSION (Unreleased)"
        
        # If we have no changes yet, the changelog is empty now, so we need to add a header
        if [[ ! -s CHANGELOG.md ]]; then
            echo "## $LATEST_VERSION (Unreleased)" > CHANGELOG.md
            echo "" >> CHANGELOG.md
        fi
        ;;

        alpha)
        FOOTER_FILE='footer-with-experiments.md'
        PRERELEASE_VERSION=$(date +"alpha%Y%m%d")
        LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
        HUMAN_DATE=$(date +"%B %d, %Y") # Date in Janurary 1st, 2022 format
        COMPLETE_VERSION="$LATEST_VERSION-$PRERELEASE_VERSION"

        npx -y changie@$CHANGIE_VERSION merge -u "## $COMPLETE_VERSION ($HUMAN_DATE)"
        ;;

        beta)
        LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
        # We need to check if this is the first RC of the version
        BETA_NUMBER=$(git tag -l "v$LATEST_VERSION-beta*" | wc -l)
        BETA_NUMBER=$((BETA_NUMBER + 1))
        HUMAN_DATE=$(date +"%B %d, %Y") # Date in Janurary 1st, 2022 format
        COMPLETE_VERSION="$LATEST_VERSION-beta$BETA_NUMBER"

        npx -y changie@$CHANGIE_VERSION merge -u "## $COMPLETE_VERSION ($HUMAN_DATE)"
        ;;
        
        rc)
        LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
        # We need to check if this is the first RC of the version
        RC_NUMBER=$(git tag -l "v$LATEST_VERSION-rc*" | wc -l)
        RC_NUMBER=$((RC_NUMBER + 1))
        HUMAN_DATE=$(date +"%B %d, %Y") # Date in Janurary 1st, 2022 format
        COMPLETE_VERSION="$LATEST_VERSION-rc$RC_NUMBER"

        npx -y changie@$CHANGIE_VERSION merge -u "## $COMPLETE_VERSION ($HUMAN_DATE)"
        ;;

        patch)
        COMPLETE_VERSION=$(npx -y changie@$CHANGIE_VERSION next patch)
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
    cat ./.changes/$FOOTER_FILE >> CHANGELOG.md
    echo "" >> CHANGELOG.md
    cat ./.changes/previous-releases.md >> CHANGELOG.md
}

# This function expects the current branch to be main. Run it if you want to set main to the next
# minor version.
function nextminor {
    # Prepend the latest version to the previous releases
    LATEST_VERSION=$(npx -y changie@$CHANGIE_VERSION latest -r --skip-prereleases)
    LATEST_VERSION=${LATEST_VERSION%.*} # Remove the patch version
    CURRENT_FILE_CONTENT=$(cat ./.changes/previous-releases.md)
    echo "- [v$LATEST_VERSION](https://github.com/hashicorp/terraform/blob/v$LATEST_VERSION/CHANGELOG.md)" > ./.changes/previous-releases.md
    echo "$CURRENT_FILE_CONTENT" >> ./.changes/previous-releases.md

    NEXT_VERSION=$(npx -y changie@$CHANGIE_VERSION next minor)
    # Remove all existing per-release changelogs
    rm ./.changes/*.*.*.md
    # Remove all unreleased changes
    rm ./.changes/unreleased/*.yaml
    # Remove all backported changes
    rm ./.changes/backported/*.yaml
    # Create a new empty version file for the next minor version
    touch ./.changes/$NEXT_VERSION.md
    
    generate "dev"
}

# This function is expected to be run on the branch of the last minor release. It will make sure
# that backports work properly
function firstbeta {
    # For the maintenance branch we don't want to base our changelog on the unreleased but the backported folder instead
    awk '{sub(/unreleasedDir: unreleased/, "unreleasedDir: backported")}1' ./.changie.yaml > temp && mv temp ./.changie.yaml
    
    # If we have backported changes, we need to remove them now since they were backported into the
    # last version
    rm -f ./.changes/backported/*.yaml
    
    # If we have unreleased changes, they will be released in the next patch, therefore they need
    # to go into the backported folder
    if [ "$(ls -A ./.changes/unreleased/)" ]; then
        mv ./.changes/unreleased/* ./.changes/backported/
    fi

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
      
    firstbeta)
    firstbeta "${@:2}"
    ;;
    *)
      usage
      exit 1

      ;;
  esac
}

main "$@"
exit $?
