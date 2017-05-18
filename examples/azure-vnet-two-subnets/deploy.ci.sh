#!/bin/bash

set -o errexit -o nounset

# generate a unique string for CI deployment
# KEY=$(cat /dev/urandom | tr -cd 'a-z' | head -c 12)
# PASSWORD=$KEY$(cat /dev/urandom | tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | tr -cd '0-9' | head -c 2)

docker run --rm -it \
  -e ARM_CLIENT_ID \
  -e ARM_CLIENT_SECRET \
  -e ARM_SUBSCRIPTION_ID \
  -e ARM_TENANT_ID \
  -v $(pwd):/data \
  --workdir=/data \
  --entrypoint "/bin/sh" \
  hashicorp/terraform:light \
  -c "/bin/terraform get; \
      /bin/terraform validate; \
      /bin/terraform plan -out=out.tfplan -var resource_group=$KEY; \
      /bin/terraform apply out.tfplan; \
      /bin/terraform show;"

# check that resources exist via azure cli
docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
         az network vnet subnet show -n subnet1 -g $KEY --vnet-name '$KEY'vnet; \
         az network vnet subnet show -n subnet2 -g $KEY --vnet-name '$KEY'vnet;"

# cleanup deployed azure resources via terraform
docker run --rm -it \
  -e ARM_CLIENT_ID \
  -e ARM_CLIENT_SECRET \
  -e ARM_SUBSCRIPTION_ID \
  -e ARM_TENANT_ID \
  -v $(pwd):/data \
  --workdir=/data \
  --entrypoint "/bin/sh" \
  hashicorp/terraform:light \
  -c "/bin/terraform destroy -force -var resource_group=$KEY;"