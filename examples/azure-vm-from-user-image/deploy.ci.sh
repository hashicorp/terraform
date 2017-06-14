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
      /bin/terraform plan -out=out.tfplan -var hostname=$KEY -var resource_group=$EXISTING_RESOURCE_GROUP -var admin_username=$KEY -var admin_password=$PASSWORD -var image_uri=$EXISTING_IMAGE_URI -var storage_account_name=$EXISTING_STORAGE_ACCOUNT_NAME; \
      /bin/terraform apply out.tfplan; \
      /bin/terraform show;"

docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
         az vm show --name $KEY --resource-group permanent"

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
        -var hostname=$KEY \
        -var resource_group=$EXISTING_RESOURCE_GROUP \
        -var admin_username=$KEY \
        -var admin_password=$PASSWORD \
        -var image_uri=$EXISTING_IMAGE_URI \
        -var storage_account_name=$EXISTING_STORAGE_ACCOUNT_NAME \
        -target=azurerm_virtual_machine.vm \
        -target=azurerm_network_interface.nic \
        -target=azurerm_virtual_network.vnet \
        -target=azurerm_public_ip.pip;"
