#!/bin/bash

set -o errexit -o nounset

if docker -v; then

  # generate a unique string for CI deployment
  export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
  export PASSWORD=$KEY$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | env LC_CTYPE=C tr -cd '0-9' | head -c 2)
  export EXISTING_RESOURCE_GROUP=permanent
  export EXISTING_IMAGE_URI=https://permanentstor.blob.core.windows.net/permanent-vhds/permanent-osdisk1.vhd
  export EXISTING_STORAGE_ACCOUNT_NAME=permanentstor
  export EXISTING_VIRTUAL_NETWORK_NAME=vqeeopeictwmvnet
  export EXISTING_SUBNET_NAME=vqeeopeictwmsubnet

  /bin/sh ./deploy.ci.sh

else
  echo "Docker is used to run terraform commands, please install before run:  https://docs.docker.com/docker-for-mac/install/"
fi