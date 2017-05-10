variable "resource_group" {
  description = "The name of the resource group in which to create search service"
}

variable "location" {
  description = "The location/region where the search service is created. Changing this forces a new resource to be created."
  default     = "southcentralus"
}

variable "search_name" {
  description = "Service name must only contain lowercase letters, digits or dashes, cannot use dash as the first two or last one characters, cannot contain consecutive dashes, and is limited between 2 and 60 characters in length."
}

variable "sku" {
  description = "Valid values are 'free', 'standard', 'standard2', and 'standard3' (2 & 3 must be enabled on the backend by Microsoft support). 'free' provisions the service in shared clusters. 'standard' provisions the service in dedicated clusters."
  default     = "standard"
}

variable "replica_count" {
  description = "Replicas distribute search workloads across the service. You need 2 or more to support high availability (applies to Basic and Standard only)."
  default     = 1
}

variable "partition_count" {
  description = "Partitions allow for scaling of document count as well as faster indexing by sharding your index over multiple Azure Search units. Allowed values: 1, 2, 3, 4, 6, 12"
  default     = 1
}

variable "hosting_mode" {
  description = "Applicable only for SKU set to standard3. You can set this property to enable a single, high density partition that allows up to 1000 indexes, which is much higher than the maximum indexes allowed for any other SKU. Allowed values: default, highDensity"
  default     = "default"
}
