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
      /bin/terraform plan -out=out.tfplan -var hostname=$KEY -var resource_group=$EXISTING_RESOURCE_GROUP -var admin_username=$KEY -var admin_password=$PASSWORD -var image_uri=$EXISTING_IMAGE_URI -var storage_account_name=$EXISTING_STORAGE_ACCOUNT_NAME; \
      /bin/terraform apply out.tfplan"

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
