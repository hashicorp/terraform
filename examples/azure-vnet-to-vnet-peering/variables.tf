variable "resource_group" {
  description = "The name of the resource group in which the virtual networks are created"
  default     = "myrg"
}

variable "location" {
  description = "The location/region where the virtual networks are created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}
