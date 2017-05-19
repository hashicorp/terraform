#!/bin/bash

set -o errexit -o nounset

docker run --rm -it \
  -e ARM_CLIENT_ID \
  -e ARM_CLIENT_SECRET \
  -e ARM_SUBSCRIPTION_ID \
  -e ARM_TENANT_ID \
  -e AAD_CLIENT_ID \
  -e AAD_CLIENT_SECRET \
  -e KEY_ENCRYPTION_KEY_URL \
  -e KEY_VAULT_RESOURCE_ID \
  -v $(pwd):/data \
  --workdir=/data \
  --entrypoint "/bin/sh" \
  hashicorp/terraform:light \
  -c "/bin/terraform get; \
      /bin/terraform validate; \
      /bin/terraform plan -out=out.tfplan \
        -var resource_group_name=$KEY \
        -var hostname=$KEY \
        -var openshift_cluster_prefix=$KEY \
        -var openshift_master_public_ip_dns_label=$KEY \
        -var infra_lb_publicip_dns_label=$KEY \
        -var key_vault_secret=$KEY_VAULT_SECRET \
        -var admin_username=$KEY \
        -var openshift_password=$PASSWORD \
        -var key_vault_name=$KEY_VAULT_NAME \
        -var key_vault_resource_group=$KEY_VAULT_RESOURCE_GROUP \
        -var aad_client_id=$AAD_CLIENT_ID \
        -var aad_client_secret=$AAD_CLIENT_SECRET \
        -var key_vault_tenant_id=$ARM_TENANT_ID \
        -var key_vault_object_id=$AAD_CLIENT_ID \
        -var key_encryption_key_url=$KEY_ENCRYPTION_KEY_URL \
        -var key_vault_resource_id=$KEY_VAULT_RESOURCE_ID \
        -var ssh_public_key=$SSH_PUBLIC_KEY; \
      /bin/terraform apply out.tfplan"

# cleanup deployed azure resources via azure-cli
docker run --rm -it \
  azuresdk/azure-cli-python \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
         az vm show -g $KEY -n $KEY; \
         az vm encryption show -g $KEY -n $KEY"

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
        -var resource_group_name=$KEY \
        -var hostname=$KEY \
        -var openshift_cluster_prefix=$KEY \
        -var openshift_master_public_ip_dns_label=$KEY \
        -var infra_lb_publicip_dns_label=$KEY \
        -var key_vault_secret=$KEY_VAULT_SECRET \
        -var admin_username=$KEY \
        -var openshift_password=$PASSWORD \
        -var key_vault_name=$KEY_VAULT_NAME \
        -var key_vault_resource_group=$KEY_VAULT_RESOURCE_GROUP \
        -var aad_client_id=$AAD_CLIENT_ID \
        -var aad_client_secret=$AAD_CLIENT_SECRET \
        -var key_vault_tenant_id=$ARM_TENANT_ID \
        -var key_vault_object_id=$AAD_CLIENT_ID \
        -var key_encryption_key_url=$KEY_ENCRYPTION_KEY_URL \
        -var key_vault_resource_id=$KEY_VAULT_RESOURCE_ID \
        -var ssh_public_key=$SSH_PUBLIC_KEY;"