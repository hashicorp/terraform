variable "resource_group" {
  description = "The name of the resource group in which the image to clone resides."
  default     = "myrg"
}

variable "image_uri" {
  description = "Specifies the image_uri in the form publisherName:offer:skus:version. image_uri can also specify the VHD uri of a custom VM image to clone."
}

variable "os_type" {
  description = "Specifies the operating system Type, valid values are windows, linux."
  default     = "linux"
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "address_space" {
  description = "The address space that is used by the virtual network. You can supply more than one address space. Changing this forces a new resource to be created."
  default     = "10.0.0.0/24"
}

variable "subnet_prefix" {
  description = "The address prefix to use for the subnet."
  default     = "10.0.0.0/24"
}

variable "storage_account_name" {
  description = "The name of the storage account in which the image from which you are cloning resides."
}

variable "storage_account_type" {
  description = "Defines the type of storage account to be created. Valid options are Standard_LRS, Standard_ZRS, Standard_GRS, Standard_RAGRS, Premium_LRS. Changing this is sometimes valid - see the Azure documentation for more information on which types of accounts can be converted into other types."
  default     = "Premium_LRS"
}

variable "vm_size" {
  description = "Specifies the size of the virtual machine. This must be the same as the vm image from which you are copying."
  default     = "Standard_DS1_v2"
}

variable "hostname" {
  description = "VM name referenced also in storage-related names. This is also used as the label for the Domain Name and to make up the FQDN. If a domain name label is specified, an A DNS record is created for the public IP in the Microsoft Azure DNS system."
}

variable "admin_username" {
  description = "administrator user name"
  default     = "vmadmin"
}

variable "admin_password" {
  description = "The Password for the account specified in the 'admin_username' field. We recommend disabling Password Authentication in a Production environment."
}
