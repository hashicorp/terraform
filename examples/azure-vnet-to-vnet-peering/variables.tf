variable "resource_group" {
  description = "The name of the resource group in which the image to clone resides."
  default     = "myrg"
}

variable "location" {
  description = "The location/region where the virtual network is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}
