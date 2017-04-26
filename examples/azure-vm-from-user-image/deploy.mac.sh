#!/bin/bash

set -o errexit -o nounset

# generate a unique string for CI deployment
export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
export PASSWORD=$KEY$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | env LC_CTYPE=C tr -cd '0-9' | head -c 2)
export EXISTING_IMAGE_URI=https://permanentstor.blob.core.windows.net/permanent-vhds/permanent-osdisk1.vhd
export EXISTING_STORAGE_ACCOUNT_NAME=permanentstor
export EXISTING_RESOURCE_GROUP=permanent

/bin/sh ./after_deploy.sh
