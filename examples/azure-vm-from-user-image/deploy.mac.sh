#!/bin/bash

set -o errexit -o nounset

# generate a unique string for CI deployment
export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
export PASSWORD=$KEY$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | env LC_CTYPE=C tr -cd '0-9' | head -c 2)
export IMAGE_URI=https://myrgdisks640.blob.core.windows.net/vhds/original-vm20170424164303.vhd
export STORAGE_ACCOUNT_NAME=myrgdisks640
export RG=myrg

/bin/sh ./deploy.sh

# docker run --rm -it \
#     azuresdk/azure-cli-python \
#     sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID; \
#            az group delete -y -n $KEY"
