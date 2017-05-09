variable "resource_group" {
  description = "The name of the resource group in which to create the Service Bus"
}

variable "location" {
  description = "The location/region where the Service Bus is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "unique" {
  description = "a unique string that will be used to comprise the names of the Service Bus, Topic, and Subscription name spaces"
}
