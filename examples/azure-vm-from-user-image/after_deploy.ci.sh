#!/bin/bash

set -o errexit -o nounset

docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID; \
         az vm delete                 --name $KEY --resource-group permanent -y; \
         az network nic delete       --name $KEY'nic' --resource-group permanent; \
         az network vnet delete       --name $KEY'vnet' --resource-group permanent; \
         az network public-ip delete --name $KEY'-ip' --resource-group permanent;"
