variable "resource_group" {
  description = "Name of the resource group in which to deploy your new Virtual Machines"
}

variable "location" {
  description = "The location/region where the virtual network resides."
  default     = "southcentralus"
}

variable "hostname" {
  description = "This variable is used in this template to create various other names, such as vnet name, subnet name, storage account name, et. al."
}

variable "os_type" {
  description = "Type of OS on the existing vhd. Allowed values: 'windows' or 'linux'."
  default     = "windows"
}

variable "existing_storage_acct" {
  description = "The name of the storage account in which your existing VHD and image reside"
}

variable "existing_storage_acct_type" {
  description = "The type of the storage account in which your existing VHD and image reside"
  default     = "Premium_LRS"
}

variable "existing_resource_group" {
  description = "The name of the resource group in which your existing storage account with your existing VHD resides"
}

variable "address_space" {
  description = "The address space that is used by the virtual network. You can supply more than one address space. Changing this forces a new resource to be created."
  default     = "10.0.0.0/16"
}

variable "subnet_prefix" {
  description = "The address prefix to use for the subnet."
  default     = "10.0.0.0/24"
}

variable "storage_account_type" {
  description = "Defines the type of storage account to be created. Valid options are Standard_LRS, Standard_ZRS, Standard_GRS, Standard_RAGRS, Premium_LRS. Changing this is sometimes valid - see the Azure documentation for more information on which types of accounts can be converted into other types."
  default     = "Standard_LRS"
}

variable "vm_size" {
  description = "VM size of new virtual machine that will be deployed from a custom image."
  default     = "Standard_DS1_v2"
}

variable "image_publisher" {
  description = "name of the publisher of the image (az vm image list)"
  default     = "MicrosoftWindowsServer"
}

variable "image_offer" {
  description = "the name of the offer (az vm image list)"
  default     = "WindowsServer"
}

variable "image_sku" {
  description = "image sku to apply (az vm image list)"
  default     = "2012-R2-Datacenter"
}

variable "image_version" {
  description = "version of the image to apply (az vm image list)"
  default     = "latest"
}

variable "admin_username" {
  description = "Name of the local administrator account, this cannot be 'Admin', 'Administrator', or 'root'."
  default     = "vmadmin"
}

variable "admin_password" {
  description = "Local administrator password, complex password is required, do not use any variation of the word 'password' because it will be rejected. Minimum 8 characters."
}

variable "transfer_vm_name" {
  description = "Name of the Windows VM that will perform the copy of the VHD from a source storage account to the new storage account created in the new deployment, this is known as transfer vm. Must be 3-15 characters."
  default     = "transfervm"
}

variable "new_vm_name" {
  description = "Name of the new VM deployed from the custom image. Must be 3-15 characters."
  default     = "myvm"
}

variable "custom_image_name" {
  description = "Name of the VHD to be used as source syspreped/generalized image to deploy the VM, for example 'mybaseimage.vhd'"
}

variable "source_img_uri" {
  description = "Full URIs for one or more custom images (VHDs) that should be copied to the deployment storage account to spin up new VMs from them. URLs must be comma separated."
}
