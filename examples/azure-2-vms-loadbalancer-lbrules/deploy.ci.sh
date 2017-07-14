#!/bin/bash

set -o errexit -o nounset

docker run --rm -it \
  -e ARM_CLIENT_ID \
  -e ARM_CLIENT_SECRET \
  -e ARM_SUBSCRIPTION_ID \
  -e ARM_TENANT_ID \
  -v $(pwd):/data \
  --entrypoint "/bin/sh" \
  hashicorp/terraform:light \
  -c "cd /data; \
      /bin/terraform get; \
      /bin/terraform validate; \
      /bin/terraform plan -out=out.tfplan -var dns_name=$KEY -var hostname=$KEY -var lb_ip_dns_name=$KEY -var resource_group=$KEY -var admin_password=$PASSWORD; \
      /bin/terraform apply out.tfplan"

# cleanup deployed azure resources via azure-cli
docker run --rm -it \
  azuresdk/azure-cli-python:0.2.10 \
  sh -c "az login --service-principal -u $ARM_CLIENT_ID -p $ARM_CLIENT_SECRET --tenant $ARM_TENANT_ID > /dev/null; \
         az network lb show -g $KEY -n rglb; \
         az network lb rule list -g $KEY --lb-name rglb;"

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
  -c "/bin/terraform destroy -force -var dns_name=$KEY -var hostname=$KEY -var lb_ip_dns_name=$KEY -var resource_group=$KEY -var admin_password=$PASSWORD;"
