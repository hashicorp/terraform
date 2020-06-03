variable "network_id" {
  type = string

  validation {
    condition     = can(regex("^network-", var.network_id))
    error_message = "Must be an network id, starting with \"network-\"."
  }
}

variable "start_time" {
  type = string

  validation {
    condition     = can(formatdate("", var.start_time))
    error_message = "Must be a valid RFC 3339 timestamp."
  }
}
