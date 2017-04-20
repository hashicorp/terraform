#!/bin/bash

set -o errexit -o nounset

# generate a unique string for CI deployment
# KEY=$(cat /dev/urandom | tr -cd 'a-z' | head -c 12)
# PASSWORD=$KEY$(cat /dev/urandom | tr -cd 'A-Z' | head -c 2)$(cat /dev/urandom | tr -cd '0-9' | head -c 2)

KEY=$1
PASSWORD=$2

docker run --rm -it -v $(pwd):/data -w /data hashicorp/terraform:light get
docker run --rm -it -v $(pwd):/data -w /data hashicorp/terraform:light plan -var dns_name=$KEY -var admin_password=$PASSWORD -var admin_username=$KEY -var resource_group=$KEY -out=out.tfplan
docker run --rm -it -v $(pwd):/data -w /data hashicorp/terraform:light apply out.tfplan

# terraform get
#
# terraform plan -var 'dns_name='$KEY -var 'admin_password='$PASSWORD -var 'admin_username='$KEY -var 'resource_group='$KEY -out=out.tfplan
#
# terraform apply out.tfplan


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
