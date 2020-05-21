#!/usr/bin/env bash

# This script pushes the docker images for the given version of Terraform,
# along with the "light", "full" and "latest" tags, up to docker hub.
#
# You must already be logged in to docker using "docker login" before running
# this script.

set -eu

VERSION="$1"
VERSION_SLUG="${VERSION#v}"
VERSION_MAJOR_MINOR=$(echo ${VERSION_SLUG} |  awk -F . '{print $1"."$2}')


echo "-- Pushing tags ${VERSION_SLUG}, light, full and latest up to dockerhub --"
echo ""

docker push "hashicorp/terraform:${VERSION_SLUG}"
docker push "hashicorp/terraform:light"
docker push "hashicorp/terraform:full"
docker push "hashicorp/terraform:latest"

docker push "hashicorp/terraform:${VERSION_MAJOR_MINOR}-light"
docker push "hashicorp/terraform:${VERSION_MAJOR_MINOR}-full"
docker push "hashicorp/terraform:${VERSION_MAJOR_MINOR}"
