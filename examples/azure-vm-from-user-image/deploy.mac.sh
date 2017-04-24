#!/bin/bash

set -o errexit -o nounset

# generate a unique string for CI deployment
export KEY=$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'a-z' | head -c 12)
export PASSWORD=$KEY$(cat /dev/urandom | env LC_CTYPE=C tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | env LC_CTYPE=C tr -cd '0-9' | head -c 2)
export IMAGE_URI='https://DISK.blob.core.windows.net/vhds/ORIGINAL-VM.vhd'
export PRIMARY_BLOB_ENDPOINT='https://DISK.blob.core.windows.net/'


/bin/sh ./deploy.sh

# docker run --rm -it \
#     azuresdk/azure-cli-python \
#     sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID; \
#            az group delete -y -n $KEY"
