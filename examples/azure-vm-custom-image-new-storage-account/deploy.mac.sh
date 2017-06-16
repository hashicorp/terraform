#!/bin/bash

set -o errexit -o nounset

if docker -v; then

  # generate a unique string for CI deployment
  export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
  export PASSWORD=$KEY$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | env LC_CTYPE=C tr -cd '0-9' | head -c 2)
  export EXISTING_RESOURCE_GROUP=donotdelete
  export EXISTING_WINDOWS_IMAGE_URI=https://tfpermstor.blob.core.windows.net/vhds/osdisk_alBZrO4OlX.vhd
  export EXISTING_STORAGE_ACCOUNT_NAME=donotdeletedisks636
  export CUSTOM_WINDOWS_IMAGE_NAME=WindowsImage
  export WINDOWS_DISK_NAME=osdisk_alBZrO4OlX

  /bin/sh ./deploy.ci.sh

else
  echo "Docker is used to run terraform commands, please install before run:  https://docs.docker.com/docker-for-mac/install/"
fi