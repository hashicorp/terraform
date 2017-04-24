#!/bin/bash

set -o errexit -o nounset

# cleanup deployed azure resources
docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID; \
         az group delete -y -n $KEY"
