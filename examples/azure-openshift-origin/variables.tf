variable "resource_group_name" {
  description = "Name of the azure resource group."
}

variable "resource_group_location" {
  description = "Location of the azure resource group."
  default     = "southcentralus"
}

variable "keyvault_name" {
  description = "Name of the key vault"
}

variable "keyvault_tenant_id" {
  description = "The Azure Active Directory tenant ID that should be used for authenticating requests to the key vault. Get using 'az account show'."
}

variable "keyvault_object_id" {
  description = "The object ID of a service principal in the Azure Active Directory tenant for the key vault. Get using 'az ad sp show'."
}

variable "keys_permissions" {
  description = "Permissions to keys in the vault. Valid values are: all, create, import, update, get, list, delete, backup, restore, encrypt, decrypt, wrapkey, unwrapkey, sign, and verify."
}

variable "secrets_permissions" {
  description = "Permissions to secrets in the vault. Valid values are: all, get, set, list, and delete."
}



# _artifactsLocation: The base URL where artifacts required by this template are located. If you are using your own fork of the repo and want the deployment to pick up artifacts from your fork, update this value appropriately (user and branch), for example, change from https://raw.githubusercontent.com/Microsoft/openshift-origin/master/ to https://raw.githubusercontent.com/YourUser/openshift-origin/YourBranch/
# osImage: Select from CentOS (centos) or RHEL (rhel) for the Operating System
# masterVmSize: Size of the Master VM. Select from one of the allowed VM sizes listed in the azuredeploy.json file
# infraVmSize: Size of the Infra VM. Select from one of the allowed VM sizes listed in the azuredeploy.json file
# nodeVmSize: Size of the Node VM. Select from one of the allowed VM sizes listed in the azuredeploy.json file
# openshiftClusterPrefix: Cluster Prefix used to configure hostnames for all nodes - master, infra and nodes. Between 1 and 20 characters
# openshiftMasterPublicIpDnsLabelPrefix: A unique Public DNS name to reference the Master Node by
# infraLbPublicIpDnsLabelPrefix: A unique Public DNS name to reference the Node Load Balancer by. Used to access deployed applications
# masterInstanceCount: Number of Masters nodes to deploy
# infraInstanceCount: Number of infra nodes to deploy
# nodeInstanceCount: Number of Nodes to deploy
# dataDiskSize: Size of data disk to attach to nodes for Docker volume - valid sizes are 128 GB, 512 GB and 1023 GB
# adminUsername: Admin username for both OS login and OpenShift login
# openshiftPassword: Password for OpenShift login
# sshPublicKey: Copy your SSH Public Key here
# keyVaultResourceGroup: The name of the Resource Group that contains the Key Vault
# keyVaultName: The name of the Key Vault you created
# keyVaultSecret: The Secret Name you used when creating the Secret (that contains the Private Key)
# aadClientId: Azure Active Directory Client ID also known as Application ID for Service Principal
# aadClientSecret: Azure Active Directory Client Secret for Service Principal
# defaultSubDomainType: This will either be xipio (if you don't have your own domain) or custom if you have your own domain that you would like to use for routing
# defaultSubDomain: The wildcard DNS name you would like to use for routing if you selected custom above. If you selected xipio above, then this field will be ignored