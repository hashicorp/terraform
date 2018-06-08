#!/usr/bin/env bash

# This script pushes the docker images for the given version of Terraform,
# along with the "light", "full" and "latest" tags, up to docker hub.
#
# You must already be logged in to docker using "docker login" before running
# this script.

set -eu

VERSION="$1"
VERSION_SLUG="${VERSION#v}"

echo "-- Pushing tags $VERSION_SLUG, light, full and latest up to dockerhub --"
echo ""

docker push "hashicorp/terraform:$VERSION_SLUG"
docker push "hashicorp/terraform:light"
docker push "hashicorp/terraform:full"
docker push "hashicorp/terraform:latest"
