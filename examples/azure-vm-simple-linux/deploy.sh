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
  --entrypoint "/bin/sh" \
  hashicorp/terraform:light \
  -c "cd /data; /bin/terraform get; /bin/terraform plan -var dns_name=$KEY -var resource_group=$KEY -var admin_username=$KEY -var admin_password=$PASSWORD -out=out.tfplan; /bin/terraform apply out.tfplan"

# TODO: determine external validation, possibly Azure CLI

# echo "Setting git user name"
# git config user.name $GH_USER_NAME
#
# echo "Setting git user email"
# git config user.email $GH_USER_EMAIL
#
# echo "Adding git upstream remote"
# git remote add upstream "https://$GH_TOKEN@github.com/$GH_REPO.git"
#
# git checkout master


#
# NOW=$(TZ=America/Chicago date)
#
# git commit -m "tfstate: $NOW [ci skip]"
#
# echo "Pushing changes to upstream master"
# git push upstream master
