variable "resource_group" {
  description = "The name of the resource group in which to create the virtual network, virtual machines, and traffic manager."
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "dns_name" {
  description = "Relative DNS name for the traffic manager profile, resulting FQDN will be <uniqueDnsName>.trafficmanager.net, must be globally unique."
}

variable "vnet" {
  description = "The name of virtual network"
  default     = "vnet"
}

variable "num_vms" {
  description = "The number of virtual machines you will provision. This variable is also used for NICs and PIPs in this Terraform script."
  default     = "3"
}

variable "address_space" {
  description = "The address space that is used by the virtual network. You can supply more than one address space. Changing this forces a new resource to be created."
  default     = "10.0.0.0/16"
}

variable "subnet_name" {
  description = "The name of the subnet"
  default     = "subnet"
}

variable "subnet_prefix" {
  description = "The address prefix to use for the subnet"
  default     = "10.0.0.0/24"
}

variable "vm_size" {
  description = "The size of the virtual machine"
  default     = "Standard_D1"
}

variable "image_publisher" {
  description = "The name of the publisher of the image (az vm image list)"
  default     = "Canonical"
}

variable "image_offer" {
  description = "The name of the offer (az vm image list)"
  default     = "UbuntuServer"
}

variable "image_sku" {
  description = "The Ubuntu version for the VM. This will pick a fully patched image of this given Ubuntu version. Allowed values: 12.04.5-LTS, 14.04.2-LTS, 15.10."
  default     = "14.04.2-LTS"
}

variable "image_version" {
  description = "the version of the image to apply (az vm image list)"
  default     = "latest"
}

variable "admin_username" {
  description = "Username for virtual machines"
  default     = "vmadmin"
}

variable "admin_password" {
  description = "Password for virtual machines"
}
