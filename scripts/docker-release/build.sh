#!/usr/bin/env bash

# This script builds two docker images for the version referred to by the
# current git HEAD.
#
# After running this, run tag.sh if the images that are built should be
# tagged as the "latest" release.

set -eu

BASE="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$BASE"

if [ "$#" -eq 0 ]; then
    # We assume that this is always running while git HEAD is pointed at a release
    # tag or a branch that is pointed at the same commit as a release tag. If not,
    # this will fail since we can't build a release image for a commit that hasn't
    # actually been released.
    VERSION="$(git describe)"
else
    # This mode is here only to support release.sh, which ensures that the given
    # version matches the current git tag. Running this script manually with
    # an argument can't guarantee correct behavior since the "full" image
    # will be built against the current work tree regardless of which version
    # is selected.
    VERSION="$1"
fi

echo "-- Building release docker images for version $VERSION --"
echo ""
VERSION_SLUG="${VERSION#v}"

docker build --no-cache "--build-arg=TERRAFORM_VERSION=${VERSION_SLUG}" -t hashicorp/terraform:${VERSION_SLUG} -f "Dockerfile-release" .
docker build --no-cache -t "hashicorp/terraform:${VERSION_SLUG}-full" ../../
