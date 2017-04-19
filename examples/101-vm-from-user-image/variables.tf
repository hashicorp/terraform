variable "resource_group" {
  description = "The name of the resource group in which to create the virtual network."
  default = "myresourcegroup"
}

variable "image_uri" {
  description = "Specifies the image_uri in the form publisherName:offer:skus:version. image_uri can also specify the VHD uri of a custom VM image to clone."
  default = ""
}

variable "os_type" {
  description = "Specifies the operating system Type, valid values are windows, linux."
  default = "linux"
}

variable "rg_prefix" {
  description = "The shortened abbreviation to represent your resource group that will go on the front of some resources."
  default = "rg"
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "virtual_network_name" {
  description = "The name for the virtual network."
  default     = "vnet"
}

# UNCERTAIN OF THIS VARIABLE
variable "address_space" {
  description = "The address space that is used by the virtual network. You can supply more than one address space. Changing this forces a new resource to be created."
  default     = "10.0.0.0/16"
}

# UNCERTAIN OF THIS VARIABLE
variable "subnet_prefix" {
  description = "The address prefix to use for the subnet."
  default     = "10.1.0.0/24"
}

variable "storage_account_type" {
  description = "Specifies the name of the storage account. Changing this forces a new resource to be created. This must be unique across the entire Azure service, not just within the resource group."
  default     = "Premium_LRS"
}

variable "vm_size" {
  description = "Specifies the name of the virtual machine resource. Changing this forces a new resource to be created."
  default     = "Standard_DS1_v2"
}

variable "image_publisher" {
  description = "name of the publisher of the image (az vm image list)"
  default     = "Canonical"
}

variable "image_offer" {
  description = "the name of the offer (az vm image list)"
  default     = "UbuntuServer"
}

variable "image_sku" {
  description = "image sku to apply (az vm image list)"
  default     = "12.04.5-LTS"
}

variable "image_version" {
  description = "version of the image to apply (az vm image list)"
  default     = "latest"
}

variable "hostname" {
  description = "VM name referenced also in storage-related names."
  default     = "myvm"
}

variable "dns_name" {
  description = " Label for the Domain Name. Will be used to make up the FQDN. If a domain name label is specified, an A DNS record is created for the public IP in the Microsoft Azure DNS system."
}

variable "admin_username" {
  description = "administrator user name"
  default     = "vmadmin"
}

variable "admin_password" {
  description = "administrator password (recommended to disable password auth)"
  default     = "T3rr@f0rmP@ssword"
}
