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