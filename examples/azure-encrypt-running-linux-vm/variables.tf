variable "resource_group" {
  description = "Resource group name."
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "add_client_id" {
  description = "Client ID of AAD app which has permissions to KeyVault"
}

variable "add_client_secret" {
  description = "Client Secret of AAD app which has permissions to KeyVault"
}

variable "disk_format_query" {
  description = "The query string used to identify the disks to format and encrypt. This parameter only works when you set the EncryptionOperation as EnableEncryptionFormat. For example, passing [{\"dev_path\":\"/dev/md0\",\"name\":\"encryptedraid\",\"file_system\":\"ext4\"}] will format /dev/md0, encrypt it and mount it at /mnt/dataraid. This parameter should only be used for RAID devices. The specified device must not have any existing filesystem on it."
}

variable "encryption_operation" {
  description = "EnableEncryption would encrypt the disks in place and EnableEncryptionFormat would format the disks directly"
  default     = "EnableEncryption"
}

variable "volume_type" {
  description = "Defines which drives should be encrypted. OS encryption is supported on RHEL 7.2, CentOS 7.2 & Ubuntu 16.04. Allowed values: OS, Data, All"
  default     = "Data"
}

variable "key_encryption_key_url" {
  description = "URL of the KeyEncryptionKey used to encrypt the volume encryption key"
  default     = ""
}

variable "key_vault_name" {
  description = "Name of the KeyVault to place the volume encryption key"
}

variable "key_vault_resource_group" {
  description = "Resource group of the KeyVault"
}

variable "passphrase" {
  description = "The passphrase for the disks"
}

variable "sequenceVersion" {
  description = "sequence version of the bitlocker operation. Increment this everytime an operation is performed on the same VM"
  default     = 1
}

variable "useKek" {
  description = "Select kek if the secret should be encrypted with a key encryption key. Allowed values: kek, nokek"
  default = "nokek"
}

variable "vm_name" {
  description = "Name of the Virtual Machine"
  default     = "myvm"
}

variable "_artifactsLocation" {
  description = "The base URI where artifacts required by this template are located. When the template is deployed using the accompanying scripts, a private location in the subscription will be used and this value will be automatically generated."
  default = "https://raw.githubusercontent.com/Azure/azure-quickstart-templates/master"
}

variable "_artifactsLocationSasToken" {
  description = "The sasToken required to access _artifactsLocation.  When the template is deployed using the accompanying scripts, a sasToken will be automatically generated."
}
