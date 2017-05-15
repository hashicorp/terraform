#!/bin/bash

set -o errexit -o nounset

if docker -v; then

  # generate a unique string for CI deployment
  export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
  export PASSWORD=$KEY$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | env LC_CTYPE=C tr -cd '0-9' | head -c 2)
  export EXISTING_RESOURCE_GROUP=donotdelete
  export EXISTING_IMAGE_URI=https://donotdeletedisks636.blob.core.windows.net/vhds/mywindowsimage20170510184809.vhd
  export EXISTING_STORAGE_ACCOUNT_NAME=donotdeletedisks636
  export CUSTOM_IMAGE_NAME=mywindowsimage20170510184809

  /bin/sh ./deploy.ci.sh

else
  echo "Docker is used to run terraform commands, please install before run:  https://docs.docker.com/docker-for-mac/install/"
fi