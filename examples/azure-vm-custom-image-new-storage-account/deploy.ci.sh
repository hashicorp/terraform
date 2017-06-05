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
        -var source_img_uri=$EXISTING_IMAGE_URI \
        -var hostname=$KEY \
        -var resource_group=$KEY \
        -var existing_resource_group=$EXISTING_RESOURCE_GROUP \
        -var admin_password=$PASSWORD \
        -var existing_storage_acct=$EXISTING_STORAGE_ACCOUNT_NAME \
        -var custom_image_name=$CUSTOM_IMAGE_NAME; \
      /bin/terraform apply out.tfplan; \
      /bin/terraform show;"

# cleanup deployed azure resources via azure-cli
docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
         az vm show -g $KEY -n myvm; \
         az storage account show -g $KEY -n $KEY;"

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
        -var source_img_uri=$EXISTING_IMAGE_URI \
        -var hostname=$KEY \
        -var resource_group=$KEY \
        -var existing_resource_group=$EXISTING_RESOURCE_GROUP \
        -var admin_password=$PASSWORD \
        -var existing_storage_acct=$EXISTING_STORAGE_ACCOUNT_NAME \
        -var custom_image_name=$CUSTOM_IMAGE_NAME \
        -target=azurerm_virtual_machine.myvm \
        -target=azurerm_virtual_machine.transfer \
        -target=azurerm_network_interface.transfernic \
        -target=azurerm_network_interface.mynic \
        -target=azurerm_virtual_network.vnet \
        -target=azurerm_public_ip.mypip \
        -target=azurerm_public_ip.transferpip \
        -target=azurerm_storage_account.stor;"