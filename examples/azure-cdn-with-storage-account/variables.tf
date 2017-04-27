variable "resource_group" {
  description = "The name of the resource group in which to create the virtual network."
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "storage_account_type" {
  description = "Specifies the name of the storage account. Changing this forces a new resource to be created. This must be unique across the entire Azure service, not just within the resource group."
  default     = "Standard_LRS"
}

variable "host_name" {
  description = "A string that determines the hostname/IP address of the origin server. This string could be a domain name, IPv4 address or IPv6 address."
  default     = "www.example.com"
}