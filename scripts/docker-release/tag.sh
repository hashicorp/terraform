#!/usr/bin/env bash

# This script tags the version number given on the command line as being
# the "latest" on the local system only.
#
# The following tags are updated:
#  - light (from the tag named after the version number)
#  - full (from the tag named after the version number with "-full" appended)
#  - latest (as an alias of light)
#
# Before running this the build.sh script must be run to actually create the
# images that this script will tag.
#
# After tagging, use push.sh to push the images to dockerhub.

set -eu

VERSION="$1"
VERSION_SLUG="${VERSION#v}"
VERSION_MAJOR_MINOR=$(echo ${VERSION_SLUG} |  awk -F . '{print $1"."$2}')


echo "-- Updating tags to point to version ${VERSION} --"
echo ""

docker tag "hashicorp/terraform:${VERSION_SLUG}" "hashicorp/terraform:light"
docker tag "hashicorp/terraform:${VERSION_SLUG}" "hashicorp/terraform:latest"
docker tag "hashicorp/terraform:${VERSION_SLUG}-full" "hashicorp/terraform:full"

docker tag "hashicorp/terraform:${VERSION_SLUG}" "hashicorp/terraform:${VERSION_MAJOR_MINOR}-light"
docker tag "hashicorp/terraform:${VERSION_SLUG}" "hashicorp/terraform:${VERSION_MAJOR_MINOR}"
docker tag "hashicorp/terraform:${VERSION_SLUG}-full" "hashicorp/terraform:${VERSION_MAJOR_MINOR}-full"

docker images --format "table {{.Repository}}:{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}" | grep hashicorp/terraform:${VERSION_MAJOR_MINOR}