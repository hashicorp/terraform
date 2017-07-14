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
  -e SSH_PUBLIC_KEY \
  -v /:/data \
  --workdir=/data/$(pwd) \
  --entrypoint "/bin/sh" \
  hashicorp/terraform:light \
  -c "/bin/terraform get; \
      /bin/terraform validate; \
      /bin/terraform plan -out=out.tfplan \
        -var subscription_id=$ARM_SUBSCRIPTION_ID \
        -var tenant_id=$ARM_TENANT_ID \
        -var aad_client_id=$ARM_CLIENT_ID \
        -var aad_client_secret=$ARM_CLIENT_SECRET \
        -var resource_group_name=$KEY \
        -var key_vault_name=$KEY_VAULT_NAME \
        -var key_vault_resource_group=$KEY_VAULT_RESOURCE_GROUP \
        -var key_vault_secret=$KEY_VAULT_SECRET \
        -var openshift_cluster_prefix=$KEY \
        -var openshift_password=$PASSWORD \
        -var openshift_script_path=$LOCAL_SCRIPT_PATH \
        -var ssh_public_key=\"$OS_PUBLIC_KEY\" \
        -var connection_private_ssh_key_path=$CONTAINER_PRIVATE_KEY_PATH \
        -var master_instance_count=$MASTER_COUNT \
        -var infra_instance_count=$INFRA_COUNT \
        -var node_instance_count=$NODE_COUNT; \
      /bin/terraform apply out.tfplan;"

# cleanup deployed azure resources via azure-cli
# docker run --rm -it \
#   azuresdk/azure-cli-python \
#   sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
#          az vm show -g $KEY -n $KEY; \
#          az vm encryption show -g $KEY -n $KEY"

# cleanup deployed azure resources via terraform