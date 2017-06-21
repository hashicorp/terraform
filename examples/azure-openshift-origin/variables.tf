variable "resource_group_name" {
  description = "Name of the azure resource group in which you will deploy this template."
}

variable "resource_group_location" {
  description = "Location of the azure resource group."
  default     = "southcentralus"
}

variable "subscription_id" {
  description = "Subscription ID of the key vault"
}

variable "tenant_id" {
  description = "Tenant ID with access to your key vault and subscription"
}

variable "openshift_script_path" {
  description = "Local path to openshift scripts to prep nodes and install openshift origin"
}

variable "os_image" {
  description = "Select from CentOS (centos) or RHEL (rhel) for the Operating System"
  default     = "centos"
}

variable "bastion_vm_size" {
  description = "Size of the Bastion Virtual Machine. Allowed values: Standard_A4, Standard_A5, Standard_A6, Standard_A7, Standard_A8, Standard_A9, Standard_A10, Standard_A11, Standard_D1, Standard_D2, Standard_D3, Standard_D4, Standard_D11, Standard_D12, Standard_D13, Standard_D14, Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2, Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2, Standard_G1, Standard_G2, Standard_G3, Standard_G4, Standard_G5, Standard_D1_v2, Standard_DS2, Standard_DS3, Standard_DS4, Standard_DS11, Standard_DS12, Standard_DS13, Standard_DS14, Standard_DS1_v2, Standard_DS2_v2, Standard_DS3_v2, Standard_DS4_v2, Standard_DS5_v2, Standard_DS11_v2, Standard_DS12_v2, Standard_DS13_v2, Standard_DS14_v2, Standard_GS1, Standard_GS2, Standard_GS3, Standard_GS4, Standard_GS5"
  default     = "Standard_D2_v2"
}

variable "master_vm_size" {
  description = "Size of the Master Virtual Machine. Allowed values: Standard_A4, Standard_A5, Standard_A6, Standard_A7, Standard_A8, Standard_A9, Standard_A10, Standard_A11, Standard_D1, Standard_D2, Standard_D3, Standard_D4, Standard_D11, Standard_D12, Standard_D13, Standard_D14, Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2, Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2, Standard_G1, Standard_G2, Standard_G3, Standard_G4, Standard_G5, Standard_D1_v2, Standard_DS2, Standard_DS3, Standard_DS4, Standard_DS11, Standard_DS12, Standard_DS13, Standard_DS14, Standard_DS1_v2, Standard_DS2_v2, Standard_DS3_v2, Standard_DS4_v2, Standard_DS5_v2, Standard_DS11_v2, Standard_DS12_v2, Standard_DS13_v2, Standard_DS14_v2, Standard_GS1, Standard_GS2, Standard_GS3, Standard_GS4, Standard_GS5"
  default     = "Standard_DS4_v2"
}

variable "infra_vm_size" {
  description = "Size of the Infra Virtual Machine. Allowed values: Standard_A4, Standard_A5, Standard_A6, Standard_A7, Standard_A8, Standard_A9, Standard_A10, Standard_A11,Standard_D1, Standard_D2, Standard_D3, Standard_D4,Standard_D11, Standard_D12, Standard_D13, Standard_D14,Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2,Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2,Standard_G1, Standard_G2, Standard_G3, Standard_G4, Standard_G5,Standard_D1_v2, Standard_DS2, Standard_DS3, Standard_DS4,Standard_DS11, Standard_DS12, Standard_DS13, Standard_DS14,Standard_DS1_v2, Standard_DS2_v2, Standard_DS3_v2, Standard_DS4_v2, Standard_DS5_v2,Standard_DS11_v2, Standard_DS12_v2, Standard_DS13_v2, Standard_DS14_v2,Standard_GS1, Standard_GS2, Standard_GS3, Standard_GS4, Standard_GS5"
  default     = "Standard_DS3_v2"
}

variable "node_vm_size" {
  description = "Size of the Node Virtual Machine. Allowed values: Standard_A4, Standard_A5, Standard_A6, Standard_A7, Standard_A8, Standard_A9, Standard_A10, Standard_A11, Standard_D1, Standard_D2, Standard_D3, Standard_D4, Standard_D11, Standard_D12, Standard_D13, Standard_D14, Standard_D1_v2, Standard_D2_v2, Standard_D3_v2, Standard_D4_v2, Standard_D5_v2, Standard_D11_v2, Standard_D12_v2, Standard_D13_v2, Standard_D14_v2, Standard_G1, Standard_G2, Standard_G3, Standard_G4, Standard_G5, Standard_D1_v2, Standard_DS2, Standard_DS3, Standard_DS4, Standard_DS11, Standard_DS12, Standard_DS13, Standard_DS14, Standard_DS1_v2, Standard_DS2_v2, Standard_DS3_v2, Standard_DS4_v2, Standard_DS5_v2, Standard_DS11_v2, Standard_DS12_v2, Standard_DS13_v2, Standard_DS14_v2, Standard_GS1, Standard_GS2, Standard_GS3, Standard_GS4, Standard_GS5"
  default     = "Standard_DS3_v2"
}

variable "storage_account_type_map" {
  description = "This is the storage account type that you will need based on the vm size that you choose (value constraints)"
  type        = "map"

  default = {
    Standard_A4      = "Standard_LRS"
    Standard_A5      = "Standard_LRS"
    Standard_A6      = "Standard_LRS"
    Standard_A7      = "Standard_LRS"
    Standard_A8      = "Standard_LRS"
    Standard_A9      = "Standard_LRS"
    Standard_A10     = "Standard_LRS"
    Standard_A11     = "Standard_LRS"
    Standard_D1      = "Standard_LRS"
    Standard_D2      = "Standard_LRS"
    Standard_D3      = "Standard_LRS"
    Standard_D4      = "Standard_LRS"
    Standard_D11     = "Standard_LRS"
    Standard_D12     = "Standard_LRS"
    Standard_D13     = "Standard_LRS"
    Standard_D14     = "Standard_LRS"
    Standard_D1_v2   = "Standard_LRS"
    Standard_D2_v2   = "Standard_LRS"
    Standard_D3_v2   = "Standard_LRS"
    Standard_D4_v2   = "Standard_LRS"
    Standard_D5_v2   = "Standard_LRS"
    Standard_D11_v2  = "Standard_LRS"
    Standard_D12_v2  = "Standard_LRS"
    Standard_D13_v2  = "Standard_LRS"
    Standard_D14_v2  = "Standard_LRS"
    Standard_G1      = "Standard_LRS"
    Standard_G2      = "Standard_LRS"
    Standard_G3      = "Standard_LRS"
    Standard_G4      = "Standard_LRS"
    Standard_G5      = "Standard_LRS"
    Standard_DS1     = "Premium_LRS"
    Standard_DS2     = "Premium_LRS"
    Standard_DS3     = "Premium_LRS"
    Standard_DS4     = "Premium_LRS"
    Standard_DS11    = "Premium_LRS"
    Standard_DS12    = "Premium_LRS"
    Standard_DS13    = "Premium_LRS"
    Standard_DS14    = "Premium_LRS"
    Standard_DS1_v2  = "Premium_LRS"
    Standard_DS2_v2  = "Premium_LRS"
    Standard_DS3_v2  = "Premium_LRS"
    Standard_DS4_v2  = "Premium_LRS"
    Standard_DS5_v2  = "Premium_LRS"
    Standard_DS11_v2 = "Premium_LRS"
    Standard_DS12_v2 = "Premium_LRS"
    Standard_DS13_v2 = "Premium_LRS"
    Standard_DS14_v2 = "Premium_LRS"
    Standard_DS15_v2 = "Premium_LRS"
    Standard_GS1     = "Premium_LRS"
    Standard_GS2     = "Premium_LRS"
    Standard_GS3     = "Premium_LRS"
    Standard_GS4     = "Premium_LRS"
    Standard_GS5     = "Premium_LRS"
  }
}

variable "os_image_map" {
  description = "os image map"
  type        = "map"

  default = {
    centos_publisher = "Openlogic"
    centos_offer     = "CentOS"
    centos_sku       = "7.3"
    centos_version   = "latest"
    rhel_publisher   = "RedHat"
    rhel_offer       = "RHEL"
    rhel_sku         = "7.2"
    rhel_version     = "latest"
  }
}

variable "disk_size_gb" {
  description = "storage os disk size"
  default     = 60
}

variable "openshift_cluster_prefix" {
  description = "Cluster Prefix used to configure domain name label and hostnames for all nodes - master, infra and nodes. Between 1 and 20 characters"
}

variable "master_instance_count" {
  description = "Number of OpenShift Masters nodes to deploy. 1 is non HA and 3 is for HA."
  default     = 1
}

variable "infra_instance_count" {
  description = "Number of OpenShift infra nodes to deploy. 1 is non HA.  Choose 2 or 3 for HA."
  default     = 1
}

variable "node_instance_count" {
  description = "Number of OpenShift nodes to deploy. Allowed values: 1-30"
  default     = 1
}

variable "data_disk_size" {
  description = "Size of data disk to attach to nodes for Docker volume - valid sizes are 128 GB, 512 GB and 1023 GB"
  default     = 128
}

variable "admin_username" {
  description = "Admin username for both OS login and OpenShift login"
  default     = "ocpadmin"
}

variable "openshift_password" {
  description = "Password for OpenShift login"
}

variable "ssh_public_key" {
  description = "Path to your SSH Public Key"
}

variable "connection_private_ssh_key_path" {
  description = "Path to the private ssh key used to connect to machines within the OpenShift cluster."
}

variable "key_vault_resource_group" {
  description = "The name of the Resource Group that contains the Key Vault"
}

variable "key_vault_name" {
  description = "The name of the Key Vault you will use"
}

variable "key_vault_secret" {
  description = "The Secret Name you used when creating the Secret (that contains the Private Key)"
}

variable "aad_client_id" {
  description = "Azure Active Directory Client ID also known as Application ID for Service Principal"
}

variable "aad_client_secret" {
  description = "Azure Active Directory Client Secret for Service Principal"
}

variable "default_sub_domain_type" {
  description = "This will either be 'xipio' (if you don't have your own domain) or 'custom' if you have your own domain that you would like to use for routing"
  default     = "xipio"
}

variable "default_sub_domain" {
  description = "The wildcard DNS name you would like to use for routing if you selected 'custom' above. If you selected 'xipio' above, then this field will be ignored"
  default     = "contoso.com"
}

variable "api_version_compute" {
  default = "2015-06-15"
}

variable "api_version" {
  default = "2015-01-01"
}
