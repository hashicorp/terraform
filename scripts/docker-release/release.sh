#!/usr/bin/env bash

# This script is an interactive wrapper around the scripts build.sh, tag.sh
# and push.sh intended for use during official Terraform releases.
#
# This script should be used only when git HEAD is pointing at the release tag
# for what will become the new latest *stable* release, since it will update
# the "latest", "light", and "full" tags to refer to what was built.
#
# To release a specific version without updating the various symbolic tags,
# use build.sh directly and then manually push the single release tag it
# creates. This is appropriate both when publishing a beta version and if,
# for some reason, it's necessary to (re-)publish and older version.

set -eu

BASE="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$BASE"

# We assume that this is always running while git HEAD is pointed at a release
# tag or a branch that is pointed at the same commit as a release tag. If not,
# this will fail since we can't build a release image for a commit that hasn't
# actually been released.
VERSION="$(git describe)"
VERSION_SLUG="${VERSION#v}"

# Verify that the version is already deployed to releases.hashicorp.com.
if curl --output /dev/null --silent --head --fail "https://releases.hashicorp.com/terraform/${VERSION_SLUG}/terraform_${VERSION_SLUG}_SHA256SUMS"; then
  echo "===== Docker image release for Terraform $VERSION ====="
  echo ""
else
  cat >&2 <<EOT

There is no $VERSION release of Terraform on releases.hashicorp.com.

release.sh can only create docker images for released versions. Use
"git checkout {version}" to switch to a release tag before running this
script.

To create an untagged docker image for any arbitrary commit, use 'docker build'
directly in the root of the Terraform repository.

EOT
  exit 1
fi

# Build the two images tagged with the version number
./build.sh "$VERSION"

# Verify that they were built correctly.
echo "-- Testing $VERSION Images --"
echo ""

echo -n "light image version: "
docker run --rm -e "CHECKPOINT_DISABLE=1" "hashicorp/terraform:${VERSION_SLUG}" version
echo -n "full image version:  "
docker run --rm -e "CHECKPOINT_DISABLE=1" "hashicorp/terraform:${VERSION_SLUG}-full" version

echo ""

read -p "Did both images produce suitable version output for $VERSION? " -n 1 -r
echo ""
if ! [[ $REPLY =~ ^[Yy]$ ]]; then
  echo >&2 Aborting due to inconsistent version output.
  exit 1
fi
echo ""

# Update the latest, light and full tags to point to the images we just built.
./tag.sh "$VERSION"

# Last chance to bail out
echo "-- Prepare to Push --"
echo ""
echo "The following Terraform images are available locally:"
docker images --format "{{.ID}}\t{{.Tag}}" hashicorp/terraform
echo ""
read -p "Ready to push the tags $VERSION_SLUG, light, full, and latest up to dockerhub? " -n 1 -r
echo ""
if ! [[ $REPLY =~ ^[Yy]$ ]]; then
  echo >&2 "Aborting because reply wasn't positive."
  exit 1
fi
echo ""

# Actually upload the images
./push.sh "$VERSION"

echo ""
echo "-- All done! --"
echo ""
echo "Confirm the release at https://hub.docker.com/r/hashicorp/terraform/tags/"
echo ""
