#!/bin/bash

set -o errexit -o nounset

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
      /bin/terraform plan -out=out.tfplan \
        -var os_disk_vhd_uri=$EXISTING_IMAGE_URI \
        -var hostname=$KEY \
        -var resource_group=$KEY \
        -var existing_vnet_resource_group=$EXISTING_RESOURCE_GROUP \
        -var admin_password=$PASSWORD \
        -var existing_subnet_id=/subscriptions/$ARM_SUBSCRIPTION_ID/resourceGroups/permanent/providers/Microsoft.Network/virtualNetworks/$EXISTING_VIRTUAL_NETWORK_NAME/subnets/$EXISTING_SUBNET_NAME \
        -var existing_subnet_name=$EXISTING_SUBNET_NAME \
        -var existing_virtual_network_name=$EXISTING_VIRTUAL_NETWORK_NAME \
        -var existing_storage_acct=$EXISTING_STORAGE_ACCOUNT_NAME; \
      /bin/terraform apply out.tfplan; \
      /bin/terraform show;"

# cleanup deployed azure resources via azure-cli
docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
         az vm show -g $KEY -n $KEY"

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
  -c "/bin/terraform destroy -force \
        -var os_disk_vhd_uri=$EXISTING_IMAGE_URI \
        -var hostname=$KEY \
        -var resource_group=$KEY \
        -var existing_vnet_resource_group=$EXISTING_RESOURCE_GROUP \
        -var admin_password=$PASSWORD \
        -var existing_subnet_id=$EXISTING_SUBNET_ID \
        -var existing_subnet_name=$EXISTING_SUBNET_NAME \
        -var existing_virtual_network_name=$EXISTING_VIRTUAL_NETWORK_NAME \
        -var existing_storage_acct=$EXISTING_STORAGE_ACCOUNT_NAME \
        -target=azurerm_resource_group.rg"