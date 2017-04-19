#!/bin/bash

set -o errexit -o nounset

# generate a unique string for CI deployment
KEY=$(cat /dev/urandom | tr -cd 'a-f0-9' | head -c 16)

terraform get

terraform plan \
  -var 'dns_name='$KEY \
  -var 'admin_password='$KEY \
  -var 'admin_username='$KEY \
  -var 'resource_group='$KEY
  -out=out.tfplan

terraform apply out.tfplan


# TODO: determine external validation, possibly Azure CLI

terraform destroy

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
